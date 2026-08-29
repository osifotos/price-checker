package pricing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheOptions controls the on-disk price cache.
type CacheOptions struct {
	Dir     string        // cache root; default resolveCacheDir()
	TTL     time.Duration // default 7 days
	NoCache bool          // bypass read and write
	Refresh bool          // ignore existing entries on read, still write
}

func (o CacheOptions) ttl() time.Duration {
	if o.TTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return o.TTL
}

// CachingClient decorates a PriceQuerier with an on-disk TTL cache. Cache
// entries contain only the query key and the Price List response — never any
// plan-derived data.
type CachingClient struct {
	inner PriceQuerier
	store *diskStore
	opts  CacheOptions

	mu    sync.Mutex
	stats QueryStats
}

// NewCachingClient wraps inner with a disk cache. If opts.NoCache is set the
// returned client is a thin pass-through that still counts stats.
func NewCachingClient(inner PriceQuerier, opts CacheOptions) (*CachingClient, error) {
	dir := opts.Dir
	if dir == "" {
		dir = resolveCacheDir()
	}
	opts.Dir = dir
	store, err := newDiskStore(dir)
	if err != nil {
		return nil, err
	}
	return &CachingClient{inner: inner, store: store, opts: opts}, nil
}

// Query serves from cache when a fresh entry exists and reads are allowed,
// otherwise delegates to the inner querier and stores the result.
func (c *CachingClient) Query(ctx context.Context, q PriceQuery) ([]PriceDimension, error) {
	k := q.key()

	if !c.opts.NoCache && !c.opts.Refresh {
		if entry, ok := c.store.get(k); ok {
			if time.Since(entry.StoredAt) <= c.opts.ttl() {
				c.count(func(s *QueryStats) { s.CacheHits++ })
				return entry.Dimensions, nil
			}
		}
	}

	c.count(func(s *QueryStats) { s.CacheMisses++ })
	dims, err := c.inner.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	if !c.opts.NoCache {
		_ = c.store.put(k, cacheEntry{StoredAt: time.Now().UTC(), Dimensions: dims})
	}
	return dims, nil
}

// Stats merges the inner client's stats with this cache's hit/miss counts.
func (c *CachingClient) Stats() QueryStats {
	c.mu.Lock()
	s := c.stats
	c.mu.Unlock()
	if inner, ok := c.inner.(interface{ Stats() QueryStats }); ok {
		is := inner.Stats()
		s.Issued = is.Issued
		s.DedupHits = is.DedupHits
		s.ThrottleRetries = is.ThrottleRetries
	}
	return s
}

func (c *CachingClient) count(f func(*QueryStats)) {
	c.mu.Lock()
	f(&c.stats)
	c.mu.Unlock()
}

// ---- disk store ----

type cacheEntry struct {
	StoredAt   time.Time        `json:"stored_at"`
	Dimensions []PriceDimension `json:"dimensions"`
}

type diskStore struct {
	root string
}

func newDiskStore(root string) (*diskStore, error) {
	dir := filepath.Join(root, "v1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &diskStore{root: dir}, nil
}

func (s *diskStore) path(key string) string {
	shard := "00"
	if len(key) >= 2 {
		shard = key[:2]
	}
	return filepath.Join(s.root, shard, key+".json")
}

func (s *diskStore) get(key string) (cacheEntry, bool) {
	b, err := os.ReadFile(s.path(key))
	if err != nil {
		return cacheEntry{}, false
	}
	var e cacheEntry
	if err := json.Unmarshal(b, &e); err != nil {
		return cacheEntry{}, false // corrupt -> treat as miss
	}
	return e, true
}

func (s *diskStore) put(key string, e cacheEntry) error {
	p := s.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// resolveCacheDir returns the default cache root: $PRICE_CHECKER_CACHE_DIR, else
// ~/.price-checker/cache, else a temp dir.
func resolveCacheDir() string {
	if d := os.Getenv("PRICE_CHECKER_CACHE_DIR"); d != "" {
		return d
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".price-checker", "cache")
	}
	return filepath.Join(os.TempDir(), "price-checker-cache")
}
