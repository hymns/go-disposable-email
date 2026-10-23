# 0002 — One list, disposable-email-domains, embedded and refreshed with go generate

**Status:** accepted, 23 Oct 2026

## Context

A sign-up check needs a list of disposable providers. Two were weighed:

- [disposable-email-domains/disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains),
  `disposable_email_blocklist.conf`: CC0, curated by pull request, 9,925
  domains at commit `b5e699f` (23 Oct 2026).
- shrinathsnayak/disposable-domains: MIT, 292,000 domains merged
  automatically from 31 sources. Refused because it lists
  protonmail.com, gmx.com, qq.com and 163.com, real providers with real
  users.

A library that refuses real users is worse than one that misses a few
throwaway domains: the caller cannot tell which of their sign-ups it lost.

## Decision

**The only source is `disposable_email_blocklist.conf`, embedded with
`go:embed` as `domains.txt`.** Every `Checker` starts with it, so the library
works with no network and no setup.

- `go generate ./...` runs `internal/fetchlist`, which downloads the list,
  refuses it if it fails `Validate`, and writes it with a header naming the
  upstream commit. The refreshed file is committed and released as a patch
  version.
- The embedded list is parsed once, when the package is loaded, and shared
  read-only by every `Checker` that has not loaded its own (about 10,000 map
  entries).
- The list is CC0, so it ships inside the module with no licence condition;
  the header says where it came from.

## Consequences

A caller who never refreshes is as current as the release they built with.
New disposable providers get through until then, or until a `Refresher` runs
([0005](0005-refresh-is-opt-in-behind-a-store.md)).

## Revisit when

Upstream stops merging pull requests for three months, or another curated
list (not an automatic merge) is shown to catch more without listing real
providers.
