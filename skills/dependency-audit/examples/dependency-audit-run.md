# Worked example: triaging a noisy npm audit

---

# Dependency audit: ledger-api
2026-03-17 · 41 direct, 1,284 resolved · scanner reported **47**, reachable **4**

## Inventory

Lockfile: `package-lock.json`, present and committed. Manifest uses carets throughout, which is fine given the lockfile. Two packages resolve to duplicate versions (`semver` at 6.3.1 and 7.5.4) — noted below.

## Reachable

### D1 — `jsonwebtoken@8.5.1` — GHSA-8cf7-32gw-wr33   **CRITICAL**

**Advisory:** Versions before 9.0.0 allow algorithm confusion when `algorithms` is not explicitly specified — a token signed with the public key using HS256 is accepted where RS256 was intended.

**Why it's reachable here:** `src/auth/verify.ts:14` calls `jwt.verify(token, PUBLIC_KEY)` with no options object. This runs in `preHandler` on every route except `/health`, so the input is an anonymous attacker's raw `Authorization` header. Your public key is served at `/.well-known/jwks.json`, so the material needed to forge is published by design.

**Severity here:** Critical. Unauthenticated full authentication bypass. The advisory rates it High; it is Critical for this project because the key is public and the affected call sits on every authenticated path.

**Fix:** `8.5.1` → `9.0.2`. Two breaking changes: callbacks are no longer supported (this code already uses the sync form, so no impact) and `algorithms` is now required rather than defaulted. That second one *is* the fix:

```ts
jwt.verify(token, PUBLIC_KEY, { algorithms: ['RS256'], issuer: ISS, audience: AUD });
```

Add the `algorithms` option **first**, as a one-line patch that can ship today, then do the version bump. The option closes the vulnerability on 8.5.1 as well.

---

### D2 — `semver@6.3.1` — GHSA-c2qf-rxjj-qqgw   **LOW**

**Advisory:** ReDoS in range parsing.

**Why it's reachable here:** Reachable only through `npm` itself and two build-time transitive parents. No application code imports `semver`; `grep -r "from 'semver'" src/` returns nothing.

**Severity here:** Low. No untrusted version string reaches it at runtime. Listed only because the duplicate resolution (6.3.1 and 7.5.4 both present) suggests one path is unpatched and it is cheap to collapse.

**Fix:** Add `"overrides": { "semver": "^7.5.4" }`. No API surface in play.

## Dismissed (43)

| Package | Advisory | Verdict | Why |
| --- | --- | --- | --- |
| `tar@4.4.19` | GHSA-r628-mhmh-qjhw | Dev-only | Reached only via `node-gyp` during install. CI does not build fork PRs, so no untrusted input. Would be Medium if it did. |
| `minimatch@3.0.4` | GHSA-f8q6-p94x-37v3 | Present but unreachable | ReDoS on glob patterns. Patterns come from `.eslintrc`, which is in-repo. |
| `postcss@7.0.39` | GHSA-7fh5-64p2-3v2j | Dev-only | Build-time CSS processing over first-party files. |
| `lodash@4.17.20` ×3 paths | GHSA-35jh-r3h4-6jhm | Present but unreachable | Prototype pollution in `_.merge`/`_.set`. Grepped: 7 call sites, all `_.pick` and `_.omit`, none of the affected functions. |
| `ws@7.4.6` | GHSA-6fc8-4gx4-v693 | Not applicable | DoS via many headers. You terminate at Fly's proxy, which caps header count below the threshold. Revisit if that changes. |
| ...38 more | | Dev-only (31), unreachable (7) | |

**The headline: 47 reported, 4 reachable, 1 that actually matters.** The reason nobody on the team reads `npm audit` output is that 91% of it is noise. That is a tooling problem, addressed at the end.

## Supply chain

- **`postinstall` scripts in 3 dependencies:** `esbuild`, `sharp`, `@fly/wire`. The first two download platform binaries and are expected. **`@fly/wire` is not a package I can find on the public registry** — it resolves from a git URL in `package.json:44`, added 2026-01-09 in commit `9c1f2ab` with the message "wire helper". It runs an install script on every developer machine and every CI runner. Find out who added it and what it does. This is the single most concerning line in this audit and no scanner will ever report it.
- **`date-parse-lite`** — last published 2021-08, single maintainer, archived repository, and it parses user-supplied date strings at `src/reports/filters.ts:22`. No current advisory. That combination is a finding anyway: unmaintained code on an untrusted-input path.
- **41 direct dependencies for a service with 34 routes** is reasonable. No trend concern.
- **GitHub Actions pinned by tag**, not SHA, in all four workflows. `actions/checkout@v4` is a moving target and moving it is a supported way to change what runs in a pipeline that holds your deploy credentials.

## Upgrade plan

**Stage 1 — one PR, existing tests cover it**
- `jsonwebtoken` 8.5.1 → 9.0.2 (see D1; ship the `algorithms` option separately and first)
- `semver` override to ^7.5.4
- 12 patch-level bumps on dev dependencies, batched

**Stage 2 — one PR each**
- Replace `date-parse-lite` with `date-fns`. Migration: 3 call sites, `parse()` signature differs in argument order. Half a day.
- Pin all GitHub Actions to commit SHAs. Mechanical, 4 files.

**Stage 3 — scheduled**
- Remaining 31 dev-only advisories. Batch quarterly; none are urgent.

## No fix available

| Package | Advisory | Compensating control |
| --- | --- | --- |
| `@fly/wire` | Unknown provenance | Do not treat as a patch problem. Identify the author, read the source, and either vendor it or remove it. Until then, it is unreviewed code with install-time execution. |

## Tooling

`npm audit` at 47:4 signal-to-noise is why this is not read. Two changes:

1. Run `npm audit --omit=dev` in CI and fail the build on it. That is a 4-advisory gate today, which a human will actually look at.
2. Add `osv-scanner` for the supply-chain checks above; it catches unmaintained and non-registry packages, which `npm audit` does not model at all.

---

## What this run got right

- Led with 47 → 4, and named the noise ratio as the reason the existing tool is ignored.
- Every dismissal states the check performed (grep results, call-site counts), not a vibe.
- D1's severity was **raised** above the advisory's rating with a project-specific reason, and the fix separates a same-day mitigation from the version bump.
- The most serious finding — an unidentified git-URL dependency with an install script — came from the manual checklist, not the scanner.
- `date-parse-lite` was flagged with **no advisory at all**, on maintenance status plus input path.
- The plan names what breaks per upgrade. Nothing says "upgrade to latest."
- No upgrades were run.
