# 0001 — One flat package, `disposable`, with the standard library only

**Status:** accepted, 23 Oct 2026

## Context

The Go starter ADRs (`~/Sites/ADR/Go/`) are written for services: an HTTP
framework, a database, migrations, sessions, an outbox, containers and an
API contract. A library has none of them, so those questions are dropped
here. The one that remains is layout: Go 0002 proposes
`internal/modules/<domain>/` and says itself that "a flat layout is fine for
a small tool".

The code is about 250 lines. Keeping the list current in an application
usually means a database (a table for the list, a lock so one replica
fetches); a library that imports a driver forces it on every caller.

## Decision

**One package at the module root, `github.com/hymns/go-disposable-email`,
named `disposable`, importing only the standard library.** It is one domain
(is this address disposable?) and one helper (keep the list current); a
module tree would be more folders than code.

- `disposable.go`: the list, `Checker`, `Check`, `IsDisposable`.
- `refresh.go`: `Refresher`, the `Store` interface, `FileStore`.
- `domains.txt`: the embedded list.
- `internal/fetchlist`: the `go generate` command that refreshes
  `domains.txt`. It is `internal/` so it is not part of the API.
- `go.mod` has no `require` line. A pull request that adds one needs an ADR.
- Minimum Go is 1.25 (Go README default); `slog.DiscardHandler` needs 1.24.

## Consequences

`go get` brings in nothing else, and the library builds anywhere Go does.
Database or Redis storage of the list is the caller's code, behind `Store`
([0005](0005-refresh-is-opt-in-behind-a-store.md)).

## Revisit when

A second domain appears (format or MX checks, say), or an adapter is worth
shipping: then it goes in its own sub-package with its own `go.mod`, so the
root stays dependency-free.
