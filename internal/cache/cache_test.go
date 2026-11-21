package cache

import (
	"testing"
	"time"
)

// fakeClock permet d'avancer le temps à la main.
type fakeClock struct{ t time.Time }

func (f *fakeClock) now() time.Time          { return f.t }
func (f *fakeClock) advance(d time.Duration) { f.t = f.t.Add(d) }

func TestCacheExpiry(t *testing.T) {
	clock := &fakeClock{t: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	c := NewWithClock(5*time.Minute, clock.now)

	c.Add("goblin", []byte("data"))
	if v, ok := c.Get("goblin"); !ok || string(v) != "data" {
		t.Fatalf("fresh entry: got %q, %v", v, ok)
	}

	clock.advance(4 * time.Minute)
	if _, ok := c.Get("goblin"); !ok {
		t.Fatal("entry expired too early")
	}

	clock.advance(2 * time.Minute)
	if _, ok := c.Get("goblin"); ok {
		t.Fatal("entry should be expired after TTL")
	}

	if n := c.Reap(); n != 1 {
		t.Fatalf("Reap removed %d entries, want 1", n)
	}
	if c.Len() != 0 {
		t.Fatalf("Len = %d after reap, want 0", c.Len())
	}
}

func TestReaperGoroutine(t *testing.T) {
	c := New(10*time.Millisecond, 5*time.Millisecond)
	defer c.Stop()
	c.Add("k", []byte("v"))

	deadline := time.Now().Add(time.Second)
	for c.Len() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("reaper did not purge the expired entry")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
