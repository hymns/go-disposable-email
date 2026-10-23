# Architecture

go-disposable-email: a Go library that tells a sign-up handler whether an
email address belongs to a disposable provider, from a list embedded in the
binary and, optionally, kept current in the background. This file is the
map. The reasoning behind each choice is in [`adr/`](adr/).

## Shape

```
 sign-up handler                      your process, optional
       |                                      |
 disposable.Check(email)              Refresher.Run(ctx)
       |                               |            |
    Checker  <------ Load(list) -------+        Store (FileStore,
  (list in memory,                     |         or your DB / Redis)
   allow / block sets)          GET DefaultURL
                                (ETag, 30 s, 10 MB)
```

A check reads memory only. The only network call is the `Refresher`, in the
caller's goroutine, and only when the caller starts it.

## Domain

| Concept | Owned by | Invariant |
|---|---|---|
| List | `Checker` | Every list in use passed `Validate`: at least 5,000 domains, every line a host name ([0004](adr/0004-a-checker-value-and-a-package-default.md)) |
| Allow / block sets | `Checker` | Fixed by `New`; blocked beats allowed beats the list, most specific level first ([0003](adr/0003-lookup-walks-up-the-domain-blocked-then-allowed-then-list.md)) |
| Embedded list | `domains.txt` | Changed only by `go generate`, with the upstream commit in its header ([0002](adr/0002-one-list-embedded-from-disposable-email-domains.md)) |
| Stored list | `Store` | Written only after `Validate`; list before ETag; every replica reloads it hourly ([0005](adr/0005-refresh-is-opt-in-behind-a-store.md), [0006](adr/0006-stored-list-reloaded-hourly.md)) |

## Files

| File | Owns | Must not |
|---|---|---|
| `disposable.go` | `Checker`, `Check`, `IsDisposable`, `Validate`, `DefaultAllowed` | touch the network or the file system |
| `refresh.go` | `Refresher`, `Store`, `FileStore` | start anything on import |
| `domains.txt` | the embedded list | be edited by hand |
| `internal/fetchlist` | `go generate` command | be imported |
| `examples/<framework>` | a runnable sign-up server per framework, each its own module | be required by the root module |

## External services

| Service | Used for | Failure mode | ADR |
|---|---|---|---|
| raw.githubusercontent.com | the upstream list, by `Refresher` and `go generate` | list in use stays; error logged and returned | [0005](adr/0005-refresh-is-opt-in-behind-a-store.md) |

## Decisions

- [0001](adr/0001-one-flat-package-standard-library-only.md) — One flat package, standard library only
- [0002](adr/0002-one-list-embedded-from-disposable-email-domains.md) — One list, embedded, refreshed with go generate
- [0003](adr/0003-lookup-walks-up-the-domain-blocked-then-allowed-then-list.md) — Lookup walks up the domain, offline
- [0004](adr/0004-a-checker-value-and-a-package-default.md) — A `Checker` value with a package `Default`
- [0005](adr/0005-refresh-is-opt-in-behind-a-store.md) — Refresh is opt-in, validated, behind a `Store`
- [0006](adr/0006-stored-list-reloaded-hourly.md) — Stored list reloaded hourly, apart from the daily fetch
