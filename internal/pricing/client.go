package pricing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awspricing "github.com/aws/aws-sdk-go-v2/service/pricing"
	pricingtypes "github.com/aws/aws-sdk-go-v2/service/pricing/types"
	"golang.org/x/sync/singleflight"
)

// getProductsAPI is the minimal slice of the pricing client used here; tests
// substitute a fake.
type getProductsAPI interface {
	GetProducts(ctx context.Context, in *awspricing.GetProductsInput, optFns ...func(*awspricing.Options)) (*awspricing.GetProductsOutput, error)
}

// PriceListClient is the shared low-level Price List client. It is safe for
// concurrent use. It deduplicates identical queries within its lifetime, bounds
// concurrency with a worker pool, and records QueryStats.
type PriceListClient struct {
	api   getProductsAPI
	log   *slog.Logger
	sem   chan struct{}
	group singleflight.Group

	mu    sync.Mutex
	memo  map[string][]PriceDimension
	stats QueryStats
}

// ClientConfig configures a PriceListClient.
type ClientConfig struct {
	Concurrency int
	Logger      *slog.Logger
}

// NewPriceListClient builds a PriceListClient over the given pricing client
// (usually awspricing.NewFromConfig(cfg)).
func NewPriceListClient(api getProductsAPI, cfg ClientConfig) *PriceListClient {
	n := cfg.Concurrency
	if n <= 0 {
		n = 8
	}
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &PriceListClient{
		api:  api,
		log:  log,
		sem:  make(chan struct{}, n),
		memo: map[string][]PriceDimension{},
	}
}

// Query returns the on-demand price dimensions matching q. Identical queries
// within the client's lifetime are served from an in-memory memo and never
// re-issued to AWS.
func (c *PriceListClient) Query(ctx context.Context, q PriceQuery) ([]PriceDimension, error) {
	k := q.key()

	if dims, ok := c.memoGet(k); ok {
		c.bump(func(s *QueryStats) { s.DedupHits++ })
		return dims, nil
	}

	v, err, shared := c.group.Do(k, func() (any, error) {
		if dims, ok := c.memoGet(k); ok {
			return dims, errMemoHit
		}
		dims, err := c.fetch(ctx, q)
		if err != nil {
			return nil, err
		}
		c.memoPut(k, dims)
		return dims, nil
	})
	switch {
	case err == errMemoHit:
		c.bump(func(s *QueryStats) { s.DedupHits++ })
		return v.([]PriceDimension), nil
	case err != nil:
		return nil, err
	case shared:
		c.bump(func(s *QueryStats) { s.DedupHits++ })
		return v.([]PriceDimension), nil
	default:
		return v.([]PriceDimension), nil
	}
}

var errMemoHit = errorString("memo hit")

type errorString string

func (e errorString) Error() string { return string(e) }

func (c *PriceListClient) memoGet(k string) ([]PriceDimension, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	d, ok := c.memo[k]
	return d, ok
}

func (c *PriceListClient) memoPut(k string, d []PriceDimension) {
	c.mu.Lock()
	c.memo[k] = d
	c.mu.Unlock()
}

func (c *PriceListClient) bump(f func(*QueryStats)) {
	c.mu.Lock()
	f(&c.stats)
	c.mu.Unlock()
}

func (c *PriceListClient) fetch(ctx context.Context, q PriceQuery) ([]PriceDimension, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	c.log.Debug("price list query", "service", q.ServiceCode, "region", q.RegionCode, "purpose", q.Purpose)

	filters := make([]pricingtypes.Filter, 0, len(q.Filters)+1)
	filters = append(filters, pricingtypes.Filter{
		Type:  pricingtypes.FilterTypeTermMatch,
		Field: aws.String("regionCode"),
		Value: aws.String(q.RegionCode),
	})
	for _, f := range q.Filters {
		filters = append(filters, pricingtypes.Filter{
			Type:  pricingtypes.FilterTypeTermMatch,
			Field: aws.String(f.Field),
			Value: aws.String(f.Value),
		})
	}

	var all []string
	var token *string
	for {
		out, err := c.api.GetProducts(ctx, &awspricing.GetProductsInput{
			ServiceCode:   aws.String(q.ServiceCode),
			Filters:       filters,
			FormatVersion: aws.String("aws_v1"),
			NextToken:     token,
		})
		if err != nil {
			return nil, c.wrapErr(err)
		}
		all = append(all, out.PriceList...)
		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		token = out.NextToken
	}

	c.mu.Lock()
	c.stats.Issued++
	c.mu.Unlock()

	return parsePriceList(all)
}

func (c *PriceListClient) wrapErr(err error) error {
	var oe interface{ ErrorCode() string }
	if errors.As(err, &oe) {
		switch oe.ErrorCode() {
		case "UnrecognizedClientException", "InvalidClientTokenId", "AuthFailure", "AccessDeniedException", "ExpiredTokenException":
			return fmt.Errorf("%s: %w", credentialHelp, err)
		}
	}
	return fmt.Errorf("price list query failed: %w", err)
}

// Stats returns a copy of the accumulated counters.
func (c *PriceListClient) Stats() QueryStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}

// Close releases resources. It is safe to call once.
func (c *PriceListClient) Close() error { return nil }

// credentialHelp mirrors awsauth.CredentialHelp without importing awsauth (which
// would pull the SDK config package into more of the tree).
const credentialHelp = "AWS credentials not found or invalid — configure them via environment " +
	"variables, a shared config/credentials file, a named profile (--profile), AWS SSO, or an " +
	"instance/container role. Set the pricing region with --aws-region."
