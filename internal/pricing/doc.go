// Package pricing turns AWS resources into cost components using the AWS Price
// List Query API.
//
// It has two layers:
//
//   - PriceListClient is the shared, low-level client. It is the only thing that
//     calls the AWS SDK. It owns the bounded worker pool, per-run query
//     deduplication, and retry/backoff. CachingClient decorates it with an
//     on-disk TTL cache.
//
//   - Pricer (implemented in internal/pricing/aws) is the per-resource-type
//     plugin. A Pricer builds queries, picks price dimensions, and emits
//     schema.CostComponent values. It receives a PriceQuerier and never imports
//     the AWS SDK or touches the cache.
//
// NewCatalog returns every v1 pricer. Adding a resource type means implementing
// Pricer and adding one line to catalog.go.
package pricing
