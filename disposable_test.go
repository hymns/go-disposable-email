package disposable

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestIsDisposable(t *testing.T) {
	cases := map[string]bool{
		"a@mailinator.com":     true,
		"a@MAILINATOR.com":     true,
		"a@mailinator.com.":    true,
		" a@mailinator.com ":   true,
		"a@x.mailinator.com":   true, // subdomain of a listed domain
		"a@guerrillamail.com":  true,
		"a@10minutemail.com":   true,
		"a@gmail.com":          false,
		"a@outlook.com":        false,
		"a@company.com.my":     false,
		"a@acmemailinator.com": false, // suffix, not a subdomain
		"no-at-sign":           false,
		"a@":                   false,
	}
	for email, want := range cases {
		if got := IsDisposable(email); got != want {
			t.Errorf("IsDisposable(%q) = %v, want %v", email, got, want)
		}
	}
	if Default.Count() < MinDomains {
		t.Fatalf("list has %d domains; embedded file missing?", Default.Count())
	}
}

func TestCheck(t *testing.T) {
	if err := Check("a@mailinator.com"); !errors.Is(err, ErrDisposable) {
		t.Errorf("Check(mailinator) = %v, want ErrDisposable", err)
	}
	if err := Check("a@gmail.com"); err != nil {
		t.Errorf("Check(gmail) = %v, want nil", err)
	}
}

func TestOptions(t *testing.T) {
	c := New(
		WithAllowed("Sub.Mailinator.com"),
		WithBlocked("new-throwaway.example", "gmail.com"),
	)
	cases := map[string]bool{
		"a@sub.mailinator.com":      false, // allowed below a listed domain
		"a@x.sub.mailinator.com":    false,
		"a@mailinator.com":          true,
		"a@new-throwaway.example":   true,
		"a@x.new-throwaway.example": true,
		"a@gmail.com":               true, // blocked wins over DefaultAllowed
		"a@outlook.com":             false,
	}
	for email, want := range cases {
		if got := c.IsDisposable(email); got != want {
			t.Errorf("IsDisposable(%q) = %v, want %v", email, got, want)
		}
	}
	if Default.IsDisposable("a@new-throwaway.example") {
		t.Error("options of one Checker changed Default")
	}
}

func TestLoadRefusesBadLists(t *testing.T) {
	c := New()
	before := c.Count()
	cases := map[string]string{
		"too short":    "a.example\nb.example\n",
		"not a domain": domains(MinDomains) + "<html>\n",
	}
	for name, body := range cases {
		if _, err := c.Load(body); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if c.Count() != before {
			t.Fatalf("%s: list in use changed after a refused load", name)
		}
	}
}

func TestLoadReplacesList(t *testing.T) {
	c := New()
	n, err := c.Load(domains(MinDomains) + "brand-new-throwaway.example\n")
	if err != nil {
		t.Fatal(err)
	}
	if n != MinDomains+1 || !c.IsDisposable("a@brand-new-throwaway.example") {
		t.Fatalf("new list not in use (%d domains)", n)
	}
	if c.IsDisposable("a@mailinator.com") {
		t.Fatal("old list still in use")
	}
	if !IsDisposable("a@mailinator.com") || IsDisposable("a@brand-new-throwaway.example") {
		t.Fatal("Load on one Checker changed Default")
	}
}

func TestAllowedWinsOverList(t *testing.T) {
	c := New()
	if _, err := c.Load(domains(MinDomains) + "gmail.com\nhotmail.com\n"); err != nil {
		t.Fatal(err)
	}
	for _, d := range DefaultAllowed {
		if c.IsDisposable("a@"+d) || c.IsDisposable("a@x."+d) {
			t.Errorf("%s refused", d)
		}
	}
}

func BenchmarkIsDisposable(b *testing.B) {
	for b.Loop() {
		IsDisposable("someone@x.y.mailinator.com")
	}
}

func domains(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "d%d.example\n", i)
	}
	return b.String()
}
