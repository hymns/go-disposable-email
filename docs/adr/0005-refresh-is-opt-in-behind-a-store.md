# 0005 — Refreshing the list is opt-in, checked before use, and stored behind a `Store`

**Status:** accepted, 23 Oct 2026; reload schedule superseded by [0006](0006-stored-list-reloaded-hourly.md), 23 Oct 2026

## Context

Upstream changes most days; an embedded list is only as fresh as the
release ([0002](0002-one-list-embedded-from-disposable-email-domains.md)).
An application can fetch daily under a Postgres advisory lock, store the
list in a table and have every replica reload it. Doing that in a library
would tie it to Postgres ([0001](0001-one-flat-package-standard-library-only.md)), and a
library that starts goroutines or calls GitHub on import surprises its
callers.

What a download can do wrong: GitHub answers 5xx or an HTML error page, the
file comes back truncated, or something serves a huge body.

## Decision

**A `Refresher` does the fetching, only when the caller starts it.** Nothing
happens on import.

- `Refresh(ctx)` fetches `URL` once (default `DefaultURL`), with
  `If-None-Match` from the list in use, a 30-second timeout unless `Client`
  is set, and a `MaxListBytes` (10 MB) cap. A 304 changes nothing.
- **A download is used only if it passes `Validate`.** Otherwise the list in
  use stays and `Refresh` returns the error.
- An accepted list is written to `Store` before it is used, so a restart
  starts from it, not from the embedded copy.
- `Store` is two methods, `Get` and `Put`. The library ships `FileStore`
  (the list in a file, its ETag beside it, each replaced by one rename; the
  list before its ETag, so a crash between costs a full download rather
  than hiding a new list behind a 304). A Postgres or Redis store is the
  caller's, a few lines.
- `Run(ctx)` loads the stored list, fetches, then repeats every `Interval`
  (default 24 hours) until `ctx` is done. Loading the stored list each round
  is how a replica picks up a list another replica fetched.
- There is no cross-replica lock: each replica fetches, and an unchanged
  list costs a 304. A lock would save a request per replica per day, which
  does not justify a database dependency.
- `Logger` is a `*slog.Logger`; nil logs nothing. Errors are also returned
  by `Refresh` and `LoadStored`.

## Consequences

A caller who starts `Run` refuses a provider within a day of upstream
listing it. A wrong upstream entry for a provider not in `DefaultAllowed`
refuses its users until upstream or the caller (`WithAllowed`) fixes it.

## Revisit when

A caller runs enough replicas that GitHub starts refusing their raw
downloads (429): then `Store` grows an optional lock, or `Run` adds jitter.

## Not yet

No `Store` for Postgres or Redis, and no HTTP middleware: both need a
dependency, and the README shows the few lines each takes.
