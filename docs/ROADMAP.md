# What gets built, in what order, and what each one must not repeat

## Done

| | |
|---|---|
| `Checker`, `Check`, `IsDisposable`, embedded list | [0002](adr/0002-one-list-embedded-from-disposable-email-domains.md), [0003](adr/0003-lookup-walks-up-the-domain-blocked-then-allowed-then-list.md), [0004](adr/0004-a-checker-value-and-a-package-default.md) |
| `Refresher`, `Store`, `FileStore` | [0005](adr/0005-refresh-is-opt-in-behind-a-store.md), [0006](adr/0006-stored-list-reloaded-hourly.md) |
| `go generate` refresh of `domains.txt` | replaces curling the list and pasting it below the header by hand |

## Next, in order

**1. Tag v0.1.0**
So callers can pin a version with `go get`.

**2. Scheduled refresh of `domains.txt` in CI**
A weekly GitHub Action runs `go generate ./...` and `go test ./...`, and
opens a pull request when the list changed.

*Avoided:* an embedded copy going stale between releases, which is what
[0005](adr/0005-refresh-is-opt-in-behind-a-store.md) works around at run
time.

## Not planned

- **MX or DNS checks.** A network call at sign-up; see
  [0003](adr/0003-lookup-walks-up-the-domain-blocked-then-allowed-then-list.md).
- **Large merged lists** (shrinathsnayak/disposable-domains). They list real
  providers; see [0002](adr/0002-one-list-embedded-from-disposable-email-domains.md).
- **Framework middleware** (Fiber, Gin). A dependency for three lines the
  README already shows.
