package pricing

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type countingQuerier struct {
	n    int64
	dims []PriceDimension
	err  error
}

func (c *countingQuerier) Query(ctx context.Context, q PriceQuery) ([]PriceDimension, error) {
	atomic.AddInt64(&c.n, 1)
	return c.dims, c.err
}

func newTestCache(t *testing.T, opts CacheOptions) (*CachingClient, *countingQuerier) {
	t.Helper()
	if opts.Dir == "" {
		opts.Dir = t.TempDir()
	}
	inner := &countingQuerier{dims: []PriceDimension{{SKU: "S", USD: "0.01", Unit: "Hrs"}}}
	c, err := NewCachingClient(inner, opts)
	if err != nil {
		t.Fatal(err)
	}
	return c, inner
}

var testQuery = PriceQuery{ServiceCode: "AmazonEC2", RegionCode: "us-east-1", Filters: []Filter{{Field: "instanceType", Value: "t3.medium"}}}

func TestCacheMissThenHit(t *testing.T) {
	c, inner := newTestCache(t, CacheOptions{})
	if _, err := c.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt64(&inner.n) != 1 {
		t.Fatalf("expected 1 inner call, got %d", inner.n)
	}
	s := c.Stats()
	if s.CacheHits != 1 || s.CacheMisses != 1 {
		t.Fatalf("stats: %+v", s)
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	dir := t.TempDir()
	c, inner := newTestCache(t, CacheOptions{Dir: dir, TTL: time.Millisecond})
	if _, err := c.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := c.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt64(&inner.n) != 2 {
		t.Fatalf("expected refetch after TTL, inner calls = %d", inner.n)
	}
}

func TestCacheNoCache(t *testing.T) {
	c, inner := newTestCache(t, CacheOptions{NoCache: true})
	for i := 0; i < 3; i++ {
		if _, err := c.Query(context.Background(), testQuery); err != nil {
			t.Fatal(err)
		}
	}
	if atomic.LoadInt64(&inner.n) != 3 {
		t.Fatalf("no-cache should always delegate, got %d", inner.n)
	}
}

func TestCacheRefresh(t *testing.T) {
	dir := t.TempDir()
	c1, inner1 := newTestCache(t, CacheOptions{Dir: dir})
	if _, err := c1.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	_ = inner1

	c2, inner2 := newTestCache(t, CacheOptions{Dir: dir, Refresh: true})
	if _, err := c2.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt64(&inner2.n) != 1 {
		t.Fatalf("refresh should ignore existing entry, inner calls = %d", inner2.n)
	}
}

func TestCacheCorruptEntryIsMiss(t *testing.T) {
	dir := t.TempDir()
	c, inner := newTestCache(t, CacheOptions{Dir: dir})
	k := testQuery.key()
	p := c.store.path(k)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt64(&inner.n) != 1 {
		t.Fatalf("corrupt entry should be a miss, inner calls = %d", inner.n)
	}
}

func TestCacheConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, _ := newTestCache(t, CacheOptions{Dir: dir})
			if _, err := c.Query(context.Background(), testQuery); err != nil {
				t.Errorf("query: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestCacheFilePerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not enforce POSIX permission bits the same way as Unix")
	}
	dir := t.TempDir()
	c, _ := newTestCache(t, CacheOptions{Dir: dir})
	if _, err := c.Query(context.Background(), testQuery); err != nil {
		t.Fatal(err)
	}
	var mode fs.FileMode
	_ = filepath.WalkDir(filepath.Join(dir, "v1"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, e := d.Info(); e == nil {
				mode = info.Mode()
			}
		}
		return nil
	})
	if mode == 0 {
		t.Skip("no cache file found")
	}
	if mode.Perm() != 0o600 {
		t.Fatalf("cache file perms = %v, want 0600", mode.Perm())
	}
}
