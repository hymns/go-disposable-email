# 0003 — A check walks up the domain, offline: blocked, then allowed, then the list

**Status:** accepted, 23 Oct 2026

## Context

Disposable providers hand out subdomains (`anything.mailinator.com`), so an
exact match is not enough, but a suffix match is wrong:
`acmemailinator.com` is not `mailinator.com`. Upstream entries are sometimes
wrong, and a wrong entry for gmail.com would refuse a large share of real
sign-ups. A caller also needs to refuse a new provider before upstream lists
it, and occasionally to accept one upstream lists.

Checking an allow list only against the full domain lets a listed parent
refuse an allowed provider's subdomains, and allow and block lists kept as
exported package globals cannot be edited safely while requests read them.

## Decision

**`IsDisposable` takes the domain after the last `@`, lower-cases it, trims
spaces and a trailing dot, then checks it and each parent domain in turn.
At each level the first rule that matches decides:**

1. a domain from `WithBlocked` refuses;
2. a domain from `DefaultAllowed` or `WithAllowed` accepts;
3. a domain on the list refuses.

If no level matches, the address is accepted.

- The most specific level wins, so `WithAllowed("team.mailinator.com")`
  accepts that subdomain while `mailinator.com` stays refused.
- At one level, blocked beats allowed: the caller's explicit block is the
  only way to refuse a `DefaultAllowed` provider.
- `DefaultAllowed` is a list of 28 mainstream providers. It is a
  package variable read only by `New`, so changing it after `New` changes
  nothing.
- No DNS, MX or HTTP at check time. The check is a few map lookups (75 ns,
  no allocation, on an M3 Pro).
- It does not check that the address is well formed: an address without `@`
  or without a domain is "not disposable", and format is the caller's
  validator's job.

## Consequences

Sign-up never waits on another service and never fails because one is down.
The check does not prove the address exists or belongs to the person; that
needs a confirmation email.

## Revisit when

A caller needs MX-based detection (a domain whose mail servers are a known
disposable provider's). That is a network call and belongs in a separate,
opt-in function.
