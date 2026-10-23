# 0004 — A `Checker` value holds the list, with a package-level `Default`

**Status:** accepted, 23 Oct 2026

## Context

Keeping the list in a package-level `atomic.Pointer` and the rules in
package-level maps suits one application, but in a library:

- tests change the list for the whole package and have to put it back;
- two parts of one program (two tenants, or a sign-up form and an admin
  import) could not use different rules;
- callers editing exported maps while requests read them is a data race.

Most callers still want one line: `disposable.Check(email)`.

## Decision

**State lives in a `Checker` built by `New(opts...)`; the package functions
`Check` and `IsDisposable` use `Default = New()`.**

- A `Checker` is safe for concurrent use. The list is behind an
  `atomic.Pointer` and replaced whole by `Load`; the allow and block sets
  are fixed by `New` and never written after it.
- Options are functions (`WithAllowed`, `WithBlocked`), so new ones can be
  added without breaking callers.
- `Check` returns the sentinel `ErrDisposable`, whose text ("disposable
  email addresses are not accepted; please use a permanent email") is safe
  to show the user. Callers compare with `errors.Is` and may show their own
  message.
- `Load` and `Validate` refuse a list with fewer than `MinDomains` (5,000)
  domains or any line that is not a blank, a `#` comment or a host name;
  on refusal the list in use stays.

## Consequences

Tests build their own `Checker` and never touch `Default`. A program that
wants other rules for `Default` replaces it once at start
(`disposable.Default = disposable.New(...)`) before serving requests;
replacing it while requests run is a race, and the README says so.

## Revisit when

Callers need to change allow or block rules while running (an admin screen,
say). Then the sets move behind the same atomic swap as the list.
