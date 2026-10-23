package disposable

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// upstream serves body with an ETag and answers 304 to a matching
// If-None-Match.
type upstream struct {
	mu       sync.Mutex
	body     string
	etag     string
	status   int
	requests int
}

func (u *upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.requests++
	if u.status != 0 {
		w.WriteHeader(u.status)
		return
	}
	if u.etag != "" && r.Header.Get("If-None-Match") == u.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", u.etag)
	w.Write([]byte(u.body))
}

func (u *upstream) set(body, etag string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.body, u.etag = body, etag
}

func TestRefreshUsesNewList(t *testing.T) {
	up := &upstream{body: domains(MinDomains) + "fresh-throwaway.example\n", etag: `"v1"`}
	srv := httptest.NewServer(up)
	defer srv.Close()

	store := FileStore(filepath.Join(t.TempDir(), "list.txt"))
	r := &Refresher{Checker: New(), URL: srv.URL, Store: store}
	if err := r.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.Checker.IsDisposable("a@fresh-throwaway.example") {
		t.Fatal("fetched list not in use")
	}
	body, etag, err := store.Get(context.Background())
	if err != nil || body != up.body || etag != `"v1"` {
		t.Fatalf("stored %d bytes, etag %q, err %v", len(body), etag, err)
	}

	// Unchanged upstream: a 304, and the list stays.
	if err := r.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.Checker.IsDisposable("a@fresh-throwaway.example") {
		t.Fatal("list lost after 304")
	}
	if up.requests != 2 {
		t.Fatalf("%d requests, want 2", up.requests)
	}
}

func TestRefreshKeepsListOnBadDownload(t *testing.T) {
	up := &upstream{}
	srv := httptest.NewServer(up)
	defer srv.Close()

	cases := map[string]func(){
		"server error": func() { up.status = http.StatusBadGateway },
		"html page":    func() { up.status = 0; up.set("<html>oops</html>\n", "") },
		"too short":    func() { up.status = 0; up.set("a.example\n", "") },
		"too large": func() {
			up.status = 0
			up.set(strings.Repeat("a.example\n", MaxListBytes/10+1), "")
		},
	}
	store := FileStore(filepath.Join(t.TempDir(), "list.txt"))
	r := &Refresher{Checker: New(), URL: srv.URL, Store: store}
	before := r.Checker.Count()
	for name, setup := range cases {
		setup()
		if err := r.Refresh(context.Background()); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if r.Checker.Count() != before || !r.Checker.IsDisposable("a@mailinator.com") {
			t.Fatalf("%s: list in use changed", name)
		}
	}
	if body, _, _ := store.Get(context.Background()); body != "" {
		t.Fatal("a refused list was stored")
	}
}

func TestLoadStored(t *testing.T) {
	store := FileStore(filepath.Join(t.TempDir(), "list.txt"))
	r := &Refresher{Checker: New(), Store: store}
	if err := r.LoadStored(context.Background()); err != nil {
		t.Fatalf("empty store: %v", err)
	}
	if err := store.Put(context.Background(), domains(MinDomains)+"stored-throwaway.example\n", `"v9"`); err != nil {
		t.Fatal(err)
	}
	if err := r.LoadStored(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.Checker.IsDisposable("a@stored-throwaway.example") || r.etag != `"v9"` {
		t.Fatal("stored list not in use")
	}
}

// TestRunReloadsBetweenFetches is two replicas sharing a Store: one fetches
// a new list, and the other uses it at its next reload, long before its own
// next fetch.
func TestRunReloadsBetweenFetches(t *testing.T) {
	up := &upstream{body: domains(MinDomains), etag: `"v1"`}
	srv := httptest.NewServer(up)
	defer srv.Close()

	store := FileStore(filepath.Join(t.TempDir(), "list.txt"))
	other := &Refresher{Checker: New(), URL: srv.URL, Store: store}
	r := &Refresher{Checker: New(), URL: srv.URL, Store: store,
		Interval: time.Hour, ReloadInterval: 10 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	up.set(domains(MinDomains)+"reloaded-throwaway.example\n", `"v2"`)
	if err := other.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for !r.Checker.IsDisposable("a@reloaded-throwaway.example") {
		if time.Now().After(deadline) {
			t.Fatal("list stored by another replica not picked up")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
