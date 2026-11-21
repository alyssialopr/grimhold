package open5e

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alyssialopr/grimhold/internal/cache"
)

const goblinJSON = `{"slug":"goblin","name":"Goblin","armor_class":15,"hit_points":7,
"challenge_rating":"1/4","cr":0.25,"actions":[{"name":"Scimitar","attack_bonus":4,"damage_dice":"1d6","damage_bonus":2}]}`

const pageJSON = `{"count":2,"next":"http://x/monsters/?page=2","previous":null,
"results":[{"slug":"goblin","name":"Goblin"},{"slug":"orc","name":"Orc"}]}`

// newTestServer simule l'API Open5e : aucun test ne touche au réseau.
func newTestServer(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/monsters/goblin/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		w.Write([]byte(goblinJSON))
	})
	mux.HandleFunc("/monsters/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/monsters/" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(pageJSON))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestGetMonsterDecodesAndCaches(t *testing.T) {
	var hits int32
	srv := newTestServer(t, &hits)
	c := NewClient(srv.URL, cache.NewWithClock(time.Minute, time.Now))

	for i := 0; i < 3; i++ {
		m, err := c.GetMonster("goblin")
		if err != nil {
			t.Fatalf("GetMonster: %v", err)
		}
		if m.Name != "Goblin" || m.ArmorClass != 15 || m.HitPoints != 7 || m.CR != 0.25 {
			t.Fatalf("bad decode: %+v", m)
		}
		if len(m.Actions) != 1 || m.Actions[0].DamageDice != "1d6" {
			t.Fatalf("bad actions: %+v", m.Actions)
		}
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("server hit %d times, want 1 (cache)", hits)
	}
}

func TestGetMonsterNotFound(t *testing.T) {
	var hits int32
	srv := newTestServer(t, &hits)
	c := NewClient(srv.URL, cache.NewWithClock(time.Minute, time.Now))

	_, err := c.GetMonster("unknown-beast")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestListMonstersPagination(t *testing.T) {
	var hits int32
	srv := newTestServer(t, &hits)
	c := NewClient(srv.URL, cache.NewWithClock(time.Minute, time.Now))

	page, err := c.ListMonsters(srv.URL + "/monsters/")
	if err != nil {
		t.Fatalf("ListMonsters: %v", err)
	}
	if page.Count != 2 || len(page.Results) != 2 {
		t.Fatalf("bad page: %+v", page)
	}
	if page.Next == "" || page.Previous != "" {
		t.Fatalf("bad pagination: next=%q prev=%q", page.Next, page.Previous)
	}
}
