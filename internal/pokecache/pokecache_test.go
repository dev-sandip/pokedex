package pokecache

import (
	"testing"
	"time"
)

func TestCacheAddAndGet(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	key := "https://example.com"
	value := []byte("hello, world")

	cache.Add(key, value)

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected cache entry to exist")
	}

	if string(got) != string(value) {
		t.Errorf("expected %q, got %q", value, got)
	}
}

func TestCacheGetMissing(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	_, ok := cache.Get("missing")
	if ok {
		t.Fatal("expected cache entry not to exist")
	}
}
