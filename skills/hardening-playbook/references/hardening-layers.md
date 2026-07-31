# Hardening layers, scoring, and durability

## Scoring: risk reduction per effort

For each gap, place it on both axes, then order by the ratio.

**Risk reduced** — how much worse things are with this gap open, given `exposure` and `data_sensitivity`:
- **High** — closes an internet-reachable path to the sensitive data class, or removes a whole class of bug.
- **Medium** — meaningful defense in depth, or closes a gap that needs a precondition to exploit.
- **Low** — marginal, or only matters if several other controls already failed.

**Effort** — S (minutes to an hour), M (up to a day), L (multiple days or a design decision).

**Order:** all High/S first, then High/M and Medium/S, and so on. A High/S — a one-line change that closes an internet-facing hole — is the best money you will ever spend. An L that is only Low risk goes to the bottom regardless of how satisfying it would be to do.

State the placement reasoning per item. "High risk / S effort — `0.0.0.0/0` on the database security group, one-line CIDR change" lets the user agree or move it. A bare ranking does not.

## Structural versus additive

**Structural** removes the possibility of a bug class. Ranks higher at equal effort, because it prevents rather than mitigates and does not depend on being remembered:
- Row-level security → cross-tenant leaks become impossible, not merely checked.
- A query layer that cannot emit raw identifiers → that injection class is gone.
- OIDC federation for deploys → no static credential exists to leak.
- Rendering that escapes by construction → XSS via that path cannot happen.

**Additive** adds a layer that helps only if something else already failed. Real value, but as defense in depth:
- A Content-Security-Policy → helps *if* an XSS already got through.
- A WAF rule → helps *if* the input validation missed something.
- Rate limiting → caps the damage of an attack that is otherwise working.

Both matter. But given equal effort, spend it on the structural change first, because the additive one is only ever your second line.

## Application layer

- [ ] Authn: sessions regenerated on login; tokens verified with explicit algorithm/issuer/audience/expiry; MFA available for privileged accounts.
- [ ] Authz: decided centrally, not per-handler; ownership checked at every object access; default-deny.
- [ ] Input: validated at the boundary; parameterized queries; identifiers allowlisted; size caps before buffering.
- [ ] Output: escaped by default; correct content types; no sensitive data in responses beyond what the caller needs.
- [ ] Session/cookies: `HttpOnly`, `Secure`, `SameSite`; sensible expiry; server-side revocation.
- [ ] Crypto: vetted libraries only; no `Math.random` for security values; constant-time comparison for secrets; TLS everywhere.
- [ ] Secrets: injected at runtime, never in images or repo; a manager, not env files; rotation possible.
- [ ] Errors/logs: fail closed; no stack traces to users; no secrets or PII in logs; audit records on privileged actions.

## Runtime layer

- [ ] Container runs as non-root; read-only filesystem where possible; no unnecessary capabilities; no mounted docker socket.
- [ ] Network: minimal exposure; database and internal services not internet-reachable; egress restricted where feasible.
- [ ] Resource limits set (CPU, memory, connections) so one workload cannot starve others.
- [ ] The platform's own controls are on: security groups scoped, NetworkPolicies present, WAF/DDoS if applicable.
- [ ] Base images pinned by digest and patched; minimal images (distroless/alpine) to shrink attack surface.

## Pipeline layer

- [ ] CI does not expose secrets to untrusted PRs; no `pull_request_target` checking out untrusted code with secrets.
- [ ] Actions/plugins pinned to commit SHAs, not mutable tags.
- [ ] Deploy credentials via OIDC federation, not long-lived static keys.
- [ ] Branch protection on deploy branches; required reviews; required status checks.
- [ ] Dependency scanning and update automation that someone actually acts on.
- [ ] Artifact integrity: signed builds or provenance where the ecosystem supports it.

## Data layer

- [ ] Encrypted at rest (database, disks, object storage) and in transit (TLS on every hop, including internal).
- [ ] Access scoped: the app connects with least privilege, not as owner/superuser.
- [ ] Backups encrypted, access-controlled, and actually restorable (tested).
- [ ] Retention and deletion policies exist and are enforced, especially under a compliance regime.
- [ ] Multi-tenant isolation enforced structurally (row-level security) rather than per-query.

## Durability: make it stick

Hardening decays as code changes. Wire in the guard so the improvement defends itself:

| Gap closed | Durability mechanism |
| --- | --- |
| Raw query removed | Lint rule banning `$queryRawUnsafe` / string-built SQL |
| Non-root container | CI check on the Dockerfile for a `USER` directive |
| Secrets out of repo | gitleaks pre-commit hook + CI scan |
| Actions pinned | Dependabot configured to update SHA pins; CI check rejecting tag refs |
| Dependency hygiene | `npm audit --omit=dev` (or equivalent) as a failing CI gate |
| Security headers set | A test asserting the headers on a representative response |
| Authz centralized | A lint rule or test that fails if a handler reaches the raw data client |
| TLS required | A config check rejecting non-TLS connection strings |

A one-time fix by hand is worth doing. A fix plus a gate that fails when someone undoes it is worth far more, because the second time this weakness appears, nobody has to be paying attention.
