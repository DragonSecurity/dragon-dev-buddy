# Worked example: pre-launch hardening pass

---

**User:** we launch ledger-api publicly in two weeks. Harden it. I've got maybe two days for this.

**Assistant:** Two-day appetite, public payments launch — so I'm ranking hard by risk-per-effort and showing you the plan before I touch anything. You'll cap it wherever the two days run out, and what's left stays ranked for later.

*(assesses the four layers against the repo and the Fly config)*

Here's the ranked plan. Ordered by risk reduced per unit of effort, not by layer.

## Ranked plan

| # | Gap | Layer | Risk | Effort | Type | Why here |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | DB security group allows `0.0.0.0/0:5432` | runtime | **H** | **S** | structural | Postgres open to the internet. One CIDR change. Best hour you'll spend. |
| 2 | `pull_request_target` checks out PR head w/ deploy token | pipeline | **H** | **S** | structural | Any stranger's PR can steal your Fly token. Public launch makes it worse. |
| 3 | Tenancy enforced per-query, no RLS | data | **H** | **L** | structural | The cross-tenant bug class. Row-level security ends it structurally. |
| 4 | Container runs as root | runtime | **M** | **S** | additive | Contains a write-primitive bug to the container. |
| 5 | Deploy uses a static Fly API token | pipeline | **M** | **M** | structural | OIDC federation → nothing to leak. |
| 6 | No CSP, no security headers | app | **M** | **S** | additive | Defense in depth for XSS. Cheap. |
| 7 | Sessions never expire server-side | app | **M** | **M** | structural | Add revocation + TTL. |
| 8 | Actions pinned to tags | pipeline | **M** | **S** | additive | Supply-chain via mutable tags. |
| 9 | No rate limiting on `/auth/login` | app | **M** | **M** | additive | Credential stuffing at launch. |
| 10 | Verbose server version headers | app | **L** | **S** | additive | Marginal. Bottom for a reason. |

Two things worth your attention before I start:

- **#3 is the big one and it's an L.** Row-level security makes the cross-tenant bug class *impossible* rather than fixed one query at a time. It's most of a day. If you'd rather I spend the two days on breadth — items 1,2,4,5,6,8 done and durable — say so. My recommendation is to do #3 anyway, because it's the one that would turn a forgotten `where` clause into a non-event, and you're about to go public with financial data.
- **#10 is on the list only for completeness.** Don't let me spend your two days on it.

Cap me where you want. Where should I stop?

---

**User:** do 1-6 and make them stick. skip RLS for now, we'll do it as its own project — but put it top of the backlog.

**Assistant:** Agreed on RLS as its own project — it deserves real testing, not a rushed afternoon. It goes to the top of the backlog, flagged as the highest-value single thing you can do post-launch.

*(applies 1-6, verifying each)*

## Applied

**#1 — DB security group.** `infra/security-groups.tf:22`, `0.0.0.0/0` → the app subnet CIDR only. Verified: app still connects, external connection times out. **Durable:** added a `tflint` rule rejecting `0.0.0.0/0` on any port but 443.

**#2 — CI trust model.** `preview.yml` switched from `pull_request_target` to `pull_request`; preview deploys on forks now run in a separate `workflow_run` job with no secrets. Verified: a fork PR builds, gets no token. **Durable:** a CI check failing any workflow that combines `pull_request_target` with a head checkout.

**#3 — deferred to backlog** (RLS).

**#4 — non-root container.** `Dockerfile`: added `adduser -D app` and `USER app`. Verified: binds 8080, writes exports to `/tmp` fine. **Durable:** CI check for a `USER` directive.

**#5 — OIDC deploy.** Replaced the static `FLY_API_TOKEN` with OIDC federation. Verified: deploy works, the secret is deleted from GitHub. **Durable:** the token no longer exists, which is the best durability there is.

**#6 — security headers.** Added CSP, `X-Content-Type-Options`, `Strict-Transport-Security`, `X-Frame-Options` via a Fastify hook. Verified against the real endpoints; CSP is report-only for the first week so it can't break the app silently. **Durable:** a test asserting the headers on a representative response.

## Backlog (ranked)

1. **Row-level security (#3)** — the highest-value remaining item. Own project. Makes cross-tenant leaks structurally impossible.
2. Server-side session revocation + TTL (#7)
3. Rate limit on `/auth/login` (#9)
4. Actions pinned to SHAs (#8)
5. Version header suppression (#10)

## Made durable

Five CI/lint gates added (SG CIDR, CI trust model, Dockerfile USER, security headers test, plus the existing gitleaks). Each fails the build if the weakness comes back, so none of today's work silently decays.

---

## What this run got right

- Ranked by risk-per-effort, so the open-to-the-internet database (H/S) came first and the version header (L/S) came last, regardless of layer.
- Showed the plan and asked where to stop **before** doing the work — the scope control.
- Made a clear recommendation on RLS, accepted the user's different call, and put it top of the backlog rather than dropping it.
- Distinguished structural (1,2,3,5) from additive (4,6) and said why structural ranks higher.
- Verified each change worked *and* didn't break legitimate use (the app still connects, the fork still builds).
- Wired a durability gate for every applied fix, so the second time each weakness appears, CI catches it instead of a person.
