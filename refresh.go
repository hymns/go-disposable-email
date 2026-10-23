package disposable

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultURL is the upstream list a Refresher fetches when URL is empty.
const DefaultURL = "https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/main/disposable_email_blocklist.conf"

// MaxListBytes is the largest download a Refresher reads. Upstream is about
// 100 KB.
const MaxListBytes = 10 << 20

// A Store keeps the last accepted list, so that a restart does not need the
// network and replicas sharing a Store use the same list.
type Store interface {
	// Get returns the stored list and its ETag. An empty body, with a nil
	// error, means nothing is stored yet.
	Get(ctx context.Context) (body, etag string, err error)
	// Put stores a list that has passed Validate.
	Put(ctx context.Context, body, etag string) error
}

// A Refresher keeps a Checker's list current by fetching URL. A download is
// used only if it passes Validate; otherwise the list in use stays.
type Refresher struct {
	Checker        *Checker      // nil uses Default
	URL            string        // "" uses DefaultURL
	Interval       time.Duration // between fetches in Run; 0 means 24 hours
	ReloadInterval time.Duration // between loads from Store in Run; 0 means 1 hour
	Store          Store         // optional
	Client         *http.Client  // nil uses one with a 30-second timeout
	Logger         *slog.Logger  // nil logs nothing

	mu   sync.Mutex
	etag string // of the list in use, sent as If-None-Match
}

// Run uses the stored list and fetches once, then fetches every Interval and
// loads from Store every ReloadInterval until ctx is done. The reloads are
// how a replica picks up a list another replica fetched (docs/adr/0006).
// Errors are logged and the list in use stays. Run blocks; start it with go.
func (r *Refresher) Run(ctx context.Context) {
	fetch := time.NewTicker(orDefault(r.Interval, 24*time.Hour))
	defer fetch.Stop()
	reload := time.NewTicker(orDefault(r.ReloadInterval, time.Hour))
	defer reload.Stop()
	r.reload(ctx)
	r.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-fetch.C:
			r.refresh(ctx)
		case <-reload.C:
			r.reload(ctx)
		}
	}
}

func (r *Refresher) reload(ctx context.Context) {
	if err := r.LoadStored(ctx); err != nil {
		r.logger().Warn("disposable: stored list not used", "err", err)
	}
}

func (r *Refresher) refresh(ctx context.Context) {
	if err := r.Refresh(ctx); err != nil {
		r.logger().Warn("disposable: kept the current list", "domains", r.checker().Count(), "err", err)
	}
}

// LoadStored uses the list in Store, if there is one. Another replica may
// have stored a newer list since this one last looked. A stored list with
// the ETag of the list in use is not parsed again.
func (r *Refresher) LoadStored(ctx context.Context) error {
	if r.Store == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	body, etag, err := r.Store.Get(ctx)
	if err != nil || body == "" {
		return err
	}
	if etag != "" && etag == r.etag {
		return nil
	}
	if _, err := r.checker().Load(body); err != nil {
		return err
	}
	r.etag = etag
	return nil
}

// Refresh fetches URL once. An unchanged list (304) or a refused one leaves
// the list in use; a new one is stored, then used.
func (r *Refresher) Refresh(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	body, etag, changed, err := r.download(ctx)
	if err != nil || !changed {
		return err
	}
	n, err := Validate(body)
	if err != nil {
		return err
	}
	if r.Store != nil {
		if err := r.Store.Put(ctx, body, etag); err != nil {
			return err
		}
	}
	if _, err := r.checker().Load(body); err != nil {
		return err
	}
	r.etag = etag
	r.logger().Info("disposable: list updated", "domains", n, "url", r.url())
	return nil
}

// download fetches the list, sending the current ETag as If-None-Match.
// changed is false when the server answers 304 Not Modified.
func (r *Refresher) download(ctx context.Context) (body, etag string, changed bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url(), nil)
	if err != nil {
		return "", "", false, err
	}
	if r.etag != "" {
		req.Header.Set("If-None-Match", r.etag)
	}
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return "", "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", false, fmt.Errorf("GET %s: %s", r.url(), resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxListBytes+1))
	if err != nil {
		return "", "", false, err
	}
	if len(raw) > MaxListBytes {
		return "", "", false, fmt.Errorf("list larger than %d bytes", MaxListBytes)
	}
	return string(raw), resp.Header.Get("ETag"), true, nil
}

func orDefault(d, def time.Duration) time.Duration {
	if d <= 0 {
		return def
	}
	return d
}

func (r *Refresher) checker() *Checker {
	if r.Checker == nil {
		return Default
	}
	return r.Checker
}

func (r *Refresher) url() string {
	if r.URL == "" {
		return DefaultURL
	}
	return r.URL
}

func (r *Refresher) logger() *slog.Logger {
	if r.Logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return r.Logger
}

// FileStore is a Store in a file at this path, with the ETag beside it in
// path + ".etag".
type FileStore string

func (f FileStore) Get(context.Context) (string, string, error) {
	body, err := os.ReadFile(string(f))
	if errors.Is(err, os.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	etag, err := os.ReadFile(string(f) + ".etag")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	return string(body), string(etag), nil
}

// Put writes the list before its ETag: if it stops in between, the old ETag
// only costs a full download next time, where a new ETag beside an old list
// would keep the old list behind a 304.
func (f FileStore) Put(_ context.Context, body, etag string) error {
	if err := writeFile(string(f), body); err != nil {
		return err
	}
	return writeFile(string(f)+".etag", etag)
}

// writeFile replaces path in one rename, so a reader never sees half a file.
func writeFile(path, data string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
