// Package aws contains the per-resource-type Pricer implementations. It must not
// import the AWS SDK — all AWS access goes through pricing.PriceQuerier.
package aws

import (
	"context"
	"sort"
	"strconv"

	"github.com/mtosin123/tf-price_checker/internal/pricing"
)

func i64(n int64) string { return strconv.FormatInt(n, 10) }

// Pricers returns every v1 pricer.
func Pricers() []pricing.Pricer {
	return []pricing.Pricer{
		instancePricer{},
		ebsVolumePricer{},
		ebsSnapshotPricer{},
		eipPricer{},
		rdsInstancePricer{},
		auroraPricer{},
		elastiCachePricer{},
		dynamoDBPricer{},
		loadBalancerPricer{},
		natGatewayPricer{},
		lambdaPricer{},
		s3Pricer{},
		eksClusterPricer{},
		eksNodeGroupPricer{},
		cloudWatchPricer{},
	}
}

// NewCatalog builds the catalog from Pricers().
func NewCatalog() pricing.Catalog {
	return pricing.NewCatalog(Pricers()...)
}

// query1 runs q and returns the single cheapest price dimension, if any.
func query1(ctx context.Context, qr pricing.PriceQuerier, q pricing.PriceQuery) (pricing.PriceDimension, bool, error) {
	dims, err := qr.Query(ctx, q)
	if err != nil {
		return pricing.PriceDimension{}, false, err
	}
	if len(dims) == 0 {
		return pricing.PriceDimension{}, false, nil
	}
	sort.Slice(dims, func(i, j int) bool { return lessUSD(dims[i].USD, dims[j].USD) })
	return dims[0], true, nil
}

// lessUSD compares two decimal strings numerically enough for ranking.
func lessUSD(a, b string) bool {
	ra, ea := pricing.RatFromDecimal(a)
	rb, eb := pricing.RatFromDecimal(b)
	if ea != nil || eb != nil {
		return a < b
	}
	return ra.Cmp(rb) < 0
}

func f(field, value string) pricing.Filter { return pricing.Filter{Field: field, Value: value} }
