package pricing

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awspricing "github.com/aws/aws-sdk-go-v2/service/pricing"
)

type fakeAPI struct {
	calls   int64
	pages   [][]string
	err     error
	errOnce bool
	failed  int64
}

func (f *fakeAPI) GetProducts(ctx context.Context, in *awspricing.GetProductsInput, _ ...func(*awspricing.Options)) (*awspricing.GetProductsOutput, error) {
	atomic.AddInt64(&f.calls, 1)
	if f.err != nil {
		if f.errOnce && atomic.AddInt64(&f.failed, 1) > 1 {
			// fall through to success after first failure
		} else {
			return nil, f.err
		}
	}
	idx := 0
	if in.NextToken != nil && *in.NextToken != "" {
		idx = int((*in.NextToken)[0] - '0')
	}
	out := &awspricing.GetProductsOutput{PriceList: f.pages[idx]}
	if idx+1 < len(f.pages) {
		tok := string(rune('0' + idx + 1))
		out.NextToken = aws.String(tok)
	}
	return out, nil
}

func TestClientDedup(t *testing.T) {
	api := &fakeAPI{pages: [][]string{{ec2Product}}}
	c := NewPriceListClient(api, ClientConfig{Concurrency: 4})

	var wg sync.WaitGroup
	q := PriceQuery{ServiceCode: "AmazonEC2", RegionCode: "us-east-1", Filters: []Filter{{Field: "instanceType", Value: "t3.medium"}}}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Query(context.Background(), q); err != nil {
				t.Errorf("query: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt64(&api.calls); got != 1 {
		t.Fatalf("expected exactly 1 SDK call, got %d", got)
	}
	if c.Stats().DedupHits < 9 {
		t.Fatalf("expected >=9 dedup hits, got %d", c.Stats().DedupHits)
	}
	if c.Stats().Issued != 1 {
		t.Fatalf("expected Issued=1, got %d", c.Stats().Issued)
	}
}

func TestClientPagination(t *testing.T) {
	api := &fakeAPI{pages: [][]string{
		{`{"product":{"sku":"A"},"terms":{"OnDemand":{"A.1":{"priceDimensions":{"A.1.1":{"unit":"Hrs","pricePerUnit":{"USD":"1"}}}}}}}`},
		{`{"product":{"sku":"B"},"terms":{"OnDemand":{"B.1":{"priceDimensions":{"B.1.1":{"unit":"Hrs","pricePerUnit":{"USD":"2"}}}}}}}`},
	}}
	c := NewPriceListClient(api, ClientConfig{})
	dims, err := c.Query(context.Background(), PriceQuery{ServiceCode: "X", RegionCode: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(dims) != 2 {
		t.Fatalf("expected 2 dims across 2 pages, got %d", len(dims))
	}
	if atomic.LoadInt64(&api.calls) != 2 {
		t.Fatalf("expected 2 SDK calls, got %d", api.calls)
	}
}

func TestClientErrorWrapped(t *testing.T) {
	api := &fakeAPI{err: errors.New("boom")}
	c := NewPriceListClient(api, ClientConfig{})
	_, err := c.Query(context.Background(), PriceQuery{ServiceCode: "X", RegionCode: "us-east-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}
