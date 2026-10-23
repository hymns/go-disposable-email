# Architecture Decision Records

One file per decision that is expensive to change. The reasoning stays here;
[ARCHITECTURE.md](../ARCHITECTURE.md) is the map and links back.

| # | Decision | Status |
|---|---|---|
| [0001](0001-one-flat-package-standard-library-only.md) | One flat package, `disposable`, with the standard library only | accepted, 23 Oct 2026 |
| [0002](0002-one-list-embedded-from-disposable-email-domains.md) | One list, disposable-email-domains, embedded and refreshed with go generate | accepted, 23 Oct 2026 |
| [0003](0003-lookup-walks-up-the-domain-blocked-then-allowed-then-list.md) | A check walks up the domain, offline: blocked, then allowed, then the list | accepted, 23 Oct 2026 |
| [0004](0004-a-checker-value-and-a-package-default.md) | A `Checker` value holds the list, with a package-level `Default` | accepted, 23 Oct 2026 |
| [0005](0005-refresh-is-opt-in-behind-a-store.md) | Refreshing the list is opt-in, checked before use, and stored behind a `Store` | accepted, 23 Oct 2026; reload schedule superseded by 0006 |
| [0006](0006-stored-list-reloaded-hourly.md) | The stored list is reloaded hourly, apart from the daily fetch | accepted, 23 Oct 2026 |

## Rules

- `NNNN-short-slug.md`, numbered in order, never reused.
- Status: `proposed`, `accepted, <date>`, or `superseded by NNNN, <date>`.
- An accepted ADR is not rewritten. A changed decision is a new ADR that
  supersedes the old one, and the old one's status says so.
- Write an ADR when the choice is expensive to reverse, when someone will ask
  "why not X?", or when it avoids a mistake an earlier project made.
