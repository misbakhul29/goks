package cache_test

import (
	"testing"
	"time"

	"github.com/misbakhulmunir/goks/pkg/cache"
)

func TestMemoryCache_SetGet(t *testing.T) {
	c := cache.NewMemory()
	c.Set("key", "value", 1*time.Minute)

	v, ok := c.Get("key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if v != "value" {
		t.Fatalf("expected 'value', got %v", v)
	}
}

func TestMemoryCache_Miss(t *testing.T) {
	c := cache.NewMemory()
	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestMemoryCache_Expiry(t *testing.T) {
	c := cache.NewMemory()
	c.Set("key", "value", 10*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected cache miss after expiry")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	c := cache.NewMemory()
	c.Set("key", "value", 1*time.Minute)
	c.Delete("key")

	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestMemoryCache_Remember(t *testing.T) {
	c := cache.NewMemory()
	calls := 0

	for i := 0; i < 3; i++ {
		c.Remember("key", 1*time.Minute, func() any {
			calls++
			return "computed"
		})
	}

	if calls != 1 {
		t.Fatalf("fn should be called once, got %d", calls)
	}
}

func TestMemoryCache_Flush(t *testing.T) {
	c := cache.NewMemory()
	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Flush()

	if _, ok := c.Get("a"); ok {
		t.Fatal("expected miss after flush")
	}
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected miss after flush")
	}
}

func TestMemoryCache_NoExpiry(t *testing.T) {
	c := cache.NewMemory()
	c.Set("forever", "yes", 0) // ttl=0 means no expiry

	v, ok := c.Get("forever")
	if !ok || v != "yes" {
		t.Fatal("expected hit on no-expiry key")
	}
}
