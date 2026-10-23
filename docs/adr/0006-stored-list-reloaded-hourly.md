# 0006 — The stored list is reloaded hourly, apart from the daily fetch

**Status:** accepted, 23 Oct 2026

Supersedes one point of [0005](0005-refresh-is-opt-in-behind-a-store.md):
"`Run(ctx)` loads the stored list, fetches, then repeats every `Interval`".
0005's other decisions stand.

## Context

Under 0005, `Run` loaded from `Store` only once per `Interval` (24 hours by
default). Replicas sharing a `Store` start at different times, so a list one
replica fetched could reach another almost a day later, nearly 48 hours
after upstream listed the domain. Fetching daily and reloading the store
hourly gets a new list to every replica within about 25 hours.

## Decision

**`Run` has two tickers: fetch every `Interval` (default 24 hours) and load
from `Store` every `ReloadInterval` (default 1 hour).** At start it loads,
then fetches, as before.

- A fetch tick only fetches; reloads come from their own ticker.
- `LoadStored` skips a stored list whose ETag is non-empty and equals the
  ETag of the list in use, so an unchanged store costs one `Get`, not a
  10,000-line parse, each hour.
- Without a `Store`, the reload ticker does nothing.

## Consequences

With a shared `Store`, every replica uses a newly fetched list within
`ReloadInterval`. Each replica calls `Store.Get` once an hour, which is one
row read for a database store.

## Revisit when

A caller needs a list to reach replicas in seconds: then `Store` needs a
notification (Postgres `LISTEN`, Redis pub/sub), not a shorter interval.
