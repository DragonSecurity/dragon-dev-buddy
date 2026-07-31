# Worked example: full audit chain with checkpoints

Abbreviated. What matters is the checkpoint rhythm, the stop condition, and the consolidation.

---

**User:** we have a pentest in four weeks, run a full security pass on ledger-api

**Assistant:** Pentest in four weeks justifies the full chain, stages 1 through 8. Before I start:

- Scope: the whole repo? It's ~28k lines, which is viable.
- Anything I must not touch — active migrations, files someone's mid-rewrite?
- Can I write findings into the repo as issues, or report only?

---

**User:** whole repo, don't touch src/legacy-import/ it's being deleted next week, report only for now

**Assistant:** Noted. `src/legacy-import/` excluded — I'll still flag it if something there is reachable from a live route, since "being deleted" and "deleted" are different.

**Stage 1: orient** — done

ledger-api moves money between customer bank accounts for B2B clients. Public, bearer-token auth on everything but `/health`. Input enters at 34 HTTP routes, one webhook receiver, and one queue consumer. Authorization is decided per-handler, not centrally — that's the structural fact that will shape most of this. An attacker wants: other tenants' financial records, or the ability to forge a settlement.

Next: stage 2, threat model. Redirect now if that framing is wrong.

---

**User:** that's right, go

**Assistant** *(runs `threat-model` over the confirmed scope)*

**Stage 2: threat model** — done

Four trust boundaries. Eleven threats, two Critical:

1. **T1 Critical** — cross-tenant read on `GET /reports/:id`, sequential ids, no tenant predicate
2. **T2 Critical** — same defect on the export filter path
3. **T4 High** — 7-day presigned S3 URLs for financial exports, no revocation
4. **T7 High** — no audit record on export
5. **T9 Medium** — queue starvation, no per-tenant cap

Full model written to `docs/security/2026-03-14-threat-model-ledger-api.md`.

The pattern behind T1 and T2 is that tenancy is enforced per-query, so it's one forgotten `where` clause away from failing anywhere. That's what I'll point stage 5 at.

Does that ranking match your gut? Anything obviously missing?

---

**User:** yeah T1 doesn't surprise me. What about the webhook thing I mentioned in setup?

**Assistant:** It's in the model as T6, rated High — homegrown HMAC verification with a non-constant-time compare. I ranked it below T1/T2 because it needs thousands of requests over a stable path, versus T1 which needs one authenticated user and a for-loop.

Fair challenge though: T6 forges *inbound money movement* and T1 only reads. If you'd rather I weight integrity-over-money above bulk read, I'll re-rank and stage 5 will target the webhook first.

---

**User:** no your ranking is right, reads are the compliance problem. continue

**Assistant** *(runs `secrets-and-config-audit`)*

**Stage 3: secrets and config** — **stopping the chain.**

`.env.staging` is committed at `a3f8c21`, present in the working tree, and contains what looks like a live Stripe restricted key (`rk_live_...`) plus the staging Postgres password. The key prefix is `rk_live`, not `rk_test`.

I have not tested the key against Stripe — that would be me using a credential I found, which isn't mine to do.

This is not finding number seven. Until you confirm whether that key is live:

1. Rotate it in the Stripe dashboard now
2. Check Stripe's API logs for use from IPs you don't recognize
3. `git log --all -- .env.staging` to see how long it's been there and who has cloned since

If it's live and has been for a while, the right next skill is `incident-response`, not stage 4 of an audit. Tell me which way this goes.

---

**User:** ugh. checked, it's live but it's restricted to read-only on one product. rotating now. no weird usage in the logs. keep going

**Assistant:** Rotating with a clean access log and a read-only scope — that's a finding rather than an incident. Recording it as **Critical, contained**: the scope limited the blast radius but the same commit habit put a database password next to it, and that one isn't scoped.

Resuming.

*(runs `dependency-audit`)*

**Stage 4: dependencies** — done

`npm audit` claimed 47 vulnerabilities. Reachable-and-real: **4**.

- `jsonwebtoken@8.5.1` — algorithm confusion, and you verify tokens with it on every request. **Critical.** Upgrade to 9.x, one breaking change in the callback signature.
- `tar@4.4.19` — path traversal, only reachable from a dev-time build script. **Low**, not Medium as the tool says.
- Two moderate ReDoS advisories in transitive deps of `eslint`. **Not applicable**, dev-only, never sees untrusted input.
- 43 others: unreachable, dev-only, or advisories against code paths this app doesn't call.

The gap between 47 and 4 is the point. Ninety percent of what your scanner reports is noise, and the one that matters sits on the authentication path.

Next: stage 5, targeted review. I'll go at the tenancy enforcement points from T1/T2, the webhook verifier, the JWT handling, and `known_risk_areas`. Not the whole 28k lines.

---

*(stage 5 runs, findings reported, then:)*

**Stage 6: consolidate** — done

Merged 11 threats + 9 review findings + 4 dependency findings = **17 items**, not 24.

De-duplicated:
- T1 and review finding R2 are the same defect at `reports.ts:44`. One item, keeping the model's risk reasoning and the review's line-level evidence.
- T2 and R3 likewise.

Re-ranked, two changes:
- `jsonwebtoken` moved **up** to Critical. Stage 4 rated it on the advisory; stage 2 showed it sits on the unauthenticated-to-authenticated boundary for every request.
- R7 (unbounded `SELECT` in an internal admin export) moved **down** to Low. It's real, but nothing routes to it — the admin panel was cut in January and the handler was left behind. Moved to "worth cleaning up."

**Promoted chain:** R4 (verbose error messages distinguish "not found" from "not yours") + T1 (sequential ids) compose. R4 alone is Low. Together they turn enumeration from a guessing game into a directory listing. Filed as one **Critical** with both halves named, because neither stage would have caught it alone.

---

## What this run got right

- Checkpointed after every stage, and the user redirected the framing at stage 2 — which is exactly what checkpoints are for.
- The assistant defended its ranking with reasoning, then accepted the user's call without re-litigating.
- Stage 3 **stopped the chain** rather than filing a live credential as finding #7, and refused to test the key it found.
- The 47-vs-4 dependency gap was stated as the headline, not buried.
- Stage 5 reviewed what stage 2 pointed at. The link is visible.
- Consolidation genuinely merged (24 → 17), moved severities in **both** directions with reasons, and surfaced a chain that no single stage could have found.
- The demoted finding went to "worth cleaning up" instead of padding the Critical count.
