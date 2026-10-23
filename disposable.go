// Package disposable tells whether an email address belongs to a disposable
// (throwaway) provider, so that sign-up can refuse it.
//
// The list is disposable_email_blocklist.conf from
// github.com/disposable-email-domains/disposable-email-domains (CC0). A copy
// is embedded at build time (domains.txt, refreshed with go generate) and is
// what every Checker starts with. A Refresher keeps a Checker's list current
// without a release (docs/adr/0005).
//
// A check never touches the network: no DNS, no MX lookup, no third-party
// API. It does not check that the address is well formed or that it exists.
package disposable

import (
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
)

//go:generate go run ./internal/fetchlist -o domains.txt

//go:embed domains.txt
var embedded string

// MinDomains is the smallest list Load accepts: a shorter one means a broken
// download, not a real change upstream.
const MinDomains = 5000

// ErrDisposable is what Check returns for a disposable address. Its text is
// safe to show to the person signing up.
var ErrDisposable = errors.New("disposable email addresses are not accepted; please use a permanent email")

// DefaultAllowed are mainstream providers that a Checker never refuses,
// whatever its list says, so that a wrong upstream entry cannot lock out
// their users. WithBlocked still wins over them.
var DefaultAllowed = []string{
	"gmail.com", "googlemail.com", "outlook.com", "hotmail.com", "live.com", "msn.com",
	"yahoo.com", "yahoo.com.my", "ymail.com", "icloud.com", "me.com", "mac.com",
	"proton.me", "protonmail.com", "pm.me", "gmx.com", "gmx.net", "aol.com",
	"zoho.com", "yandex.com", "mail.com", "qq.com", "163.com", "126.com",
	"fastmail.com", "tutanota.com", "tuta.io", "hey.com",
}

// embeddedList is parsed once and shared, read-only, by every Checker until
// it loads a list of its own.
var embeddedList = func() map[string]struct{} {
	m, err := parse(embedded)
	if err != nil {
		panic("disposable: embedded list: " + err.Error())
	}
	return m
}()

// Default is the Checker behind the package-level functions.
var Default = New()

// IsDisposable reports whether Default refuses the address.
func IsDisposable(email string) bool { return Default.IsDisposable(email) }

// Check returns ErrDisposable if Default refuses the address.
func Check(email string) error { return Default.Check(email) }

// A Checker holds one list and its allow and block rules. It is safe for
// concurrent use, including Load while other goroutines check.
type Checker struct {
	list    atomic.Pointer[map[string]struct{}]
	allowed map[string]struct{}
	blocked map[string]struct{}
}

// An Option changes a Checker made by New.
type Option func(*Checker)

// WithAllowed adds domains, and their subdomains, that are never refused.
func WithAllowed(domains ...string) Option {
	return func(c *Checker) { add(c.allowed, domains) }
}

// WithBlocked adds domains, and their subdomains, that are always refused,
// for providers upstream has not listed yet. It wins over every allow rule.
func WithBlocked(domains ...string) Option {
	return func(c *Checker) { add(c.blocked, domains) }
}

// New returns a Checker using the embedded list, DefaultAllowed and opts.
func New(opts ...Option) *Checker {
	c := &Checker{allowed: map[string]struct{}{}, blocked: map[string]struct{}{}}
	add(c.allowed, DefaultAllowed)
	for _, o := range opts {
		o(c)
	}
	m := embeddedList
	c.list.Store(&m)
	return c
}

// IsDisposable reports whether the address's domain, or any domain it is a
// subdomain of, is refused: "a@x.mailinator.com" is. At each level, from the
// full domain up, a blocked domain refuses, an allowed one accepts, and a
// listed one refuses. An address without a domain is not disposable.
func (c *Checker) IsDisposable(email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}
	domain := normalize(email[at+1:])
	list := *c.list.Load()
	for domain != "" {
		if _, ok := c.blocked[domain]; ok {
			return true
		}
		if _, ok := c.allowed[domain]; ok {
			return false
		}
		if _, ok := list[domain]; ok {
			return true
		}
		dot := strings.IndexByte(domain, '.')
		if dot < 0 {
			break
		}
		domain = domain[dot+1:]
	}
	return false
}

// Check returns ErrDisposable if the address is refused, and nil otherwise.
func (c *Checker) Check(email string) error {
	if c.IsDisposable(email) {
		return ErrDisposable
	}
	return nil
}

// Load replaces the list in use with body, one domain per line, if it passes
// Validate. On error the current list stays.
func (c *Checker) Load(body string) (int, error) {
	m, err := parse(body)
	if err != nil {
		return 0, err
	}
	c.list.Store(&m)
	return len(m), nil
}

// Count is the number of domains in the list in use.
func (c *Checker) Count() int { return len(*c.list.Load()) }

// Validate checks a list without using it and returns how many domains it
// holds. A list is refused if it has fewer than MinDomains domains or any
// line that is not a blank, a # comment or a host name: either means a broken
// download.
func Validate(body string) (int, error) {
	m, err := parse(body)
	return len(m), err
}

var hostname = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

func parse(body string) (map[string]struct{}, error) {
	m := make(map[string]struct{}, 10000)
	for i, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !hostname.MatchString(line) {
			return nil, fmt.Errorf("line %d is not a domain", i+1)
		}
		m[line] = struct{}{}
	}
	if len(m) < MinDomains {
		return nil, fmt.Errorf("only %d domains, want at least %d", len(m), MinDomains)
	}
	return m, nil
}

func normalize(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}

func add(m map[string]struct{}, domains []string) {
	for _, d := range domains {
		if d = normalize(d); d != "" {
			m[d] = struct{}{}
		}
	}
}
