package pricing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Filter is a term-match attribute filter for a Price List query.
type Filter struct {
	Field string
	Value string
}

// PriceQuery identifies a set of Price List products.
type PriceQuery struct {
	ServiceCode string // e.g. "AmazonEC2"
	RegionCode  string // e.g. "us-east-1"
	Filters     []Filter
	Purpose     string // human label for logs; NOT part of the cache/dedup key
}

// PriceDimension is one on-demand price point extracted from a Price List
// product.
type PriceDimension struct {
	SKU         string            `json:"sku"`
	RateCode    string            `json:"rate_code"`
	Unit        string            `json:"unit"`
	USD         string            `json:"usd"` // decimal string, price per unit
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"`
}

// QueryStats accumulates observability counters for one run.
type QueryStats struct {
	Issued          int
	DedupHits       int
	CacheHits       int
	CacheMisses     int
	ThrottleRetries int
}

// CacheHitRatio returns hits / (hits + misses), or 0 when nothing was looked up.
func (s QueryStats) CacheHitRatio() float64 {
	total := s.CacheHits + s.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(s.CacheHits) / float64(total)
}

// PriceQuerier is the only pricing capability a Pricer sees.
type PriceQuerier interface {
	Query(ctx context.Context, q PriceQuery) ([]PriceDimension, error)
}

// key returns the canonical, order-independent identity of a query. Purpose is
// excluded so that logging labels never change the cache entry.
func (q PriceQuery) key() string {
	parts := make([]string, 0, len(q.Filters))
	for _, f := range q.Filters {
		parts = append(parts, f.Field+"="+f.Value)
	}
	sort.Strings(parts)
	raw := q.ServiceCode + "|" + q.RegionCode + "|" + strings.Join(parts, ";")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
