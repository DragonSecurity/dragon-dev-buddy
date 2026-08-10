# Worked example: handing a live incident to the next shift

---

**User:** I'm off in ten minutes and Priya is taking over the incident. Write her a handoff.

**Assistant:** Boundary check first, because most requests for a handoff are better served by something else.

*(reads `.dragon-buddy/config.json` — `project.name: "orders-api"`, `security.data_sensitivity: "payment"`, `engagement.authorized_scope: []`, `output.reports_dir: "docs/security"`)*

- Continue? No — you are leaving, and Priya is on her own machine in her own window.
- Clear? No — three hours of containment reasoning that she needs.
- **Travel? Yes.** Shift handover on a live incident: a different person, a different machine. That is the branch, so a handoff it is.

We are also at a real boundary: containment closed twenty minutes ago and eradication has not started. Good place to cut.

**Before I draft — two things from this session are durable and should not go in a file that gets deleted on pickup:**

1. The webhook worker reads `RAW_BODY` from a request the JSON parser has already consumed, which is *why* signature verification was moved out of middleware in March and why it looks oddly placed. That is a permanent property of this codebase.
2. `docs/security` is synced to the shared drive, so it is not a safe place for incident material.

Both are `project-memory` entries, not handoff lines. I'll write them there separately — say the word and I'll do it now, before the handoff, since that is the one that gets skipped when the clock is running.

**Drafting.** Everything already in the incident log gets referenced, not repeated.

*(writes `${TMPDIR:-/tmp}/handoff-orders-api-webhook-incident-20260810-1740.md`)*

```markdown
# Handoff: orders-api webhook forgery incident — 2026-08-10 17:40

**For:** taking over an incident at shift change. Containment is closed;
eradication has not started.
**Repository / branch:** orders-api @ `hotfix/webhook-verify`, at `a41c9e2` (uncommitted
changes present, see In flight)

## Goal
Stop forged webhook callbacks from marking unpaid orders as paid, and establish
how many orders were affected before containment.

## Done
- Containment at 16:58: the `/webhooks/psp` route is behind a deny-all rule at
  the edge. Legitimate callbacks are queuing at the provider and will replay.
- Blast radius query run; 14 orders were marked paid without a matching
  settlement record. IDs are in the incident log, not here.
- Signing secret rotated at 17:05. The old secret is revoked at the provider.
- Timeline maintained live in `docs/security/2026-08-10-incident-whk-04.md`.

## In flight
- `src/webhooks/verify.ts` — HMAC comparison rewritten to constant-time and
  finished. Reviewed, not committed.
- `src/webhooks/router.ts` — half migrated. Two of six routes call `verify()`;
  `refund`, `chargeback`, `dispute` and `payout` still trust the `X-Signature`
  header directly. **The suite is red on `router.test.ts:88` for exactly this
  reason and no other** — do not go hunting for a second fault.
- A temporary deny-all rule is live at the edge for `/webhooks/psp`. It is the
  containment; it also breaks legitimate settlement. It must come out in a
  controlled step, not by someone tidying up.
- Provider dashboard session is open on my machine only. Credentials:
  `<REDACTED: PSP dashboard admin login — 1Password, vault "Platform", item
  "psp-admin">`. Ops can grant Priya access; I have not shared a session.

## Ruled out
- **Verification in Express middleware.** The JSON body parser consumes the raw
  body before middleware runs, so the HMAC never matches. This is the obvious
  fix and it does not work — it was tried in March too.
- **Rotating the signing secret alone as the fix.** The forged requests never had
  a valid signature; the endpoint was not checking one on four routes. Rotation
  was correct hygiene, not remediation.
- **Blocking the attacker source IP.** Single cloud-hosted address, rotates freely.
  Costs a step and buys nothing.

## Next action
Finish the migration in `src/webhooks/router.ts` — the remaining four routes take
the same `verify()` call as `charge` and `capture`. `router.test.ts` goes green
when all six are covered. Do not lift the edge rule until it does.

## Artifacts
- `docs/security/2026-08-10-incident-whk-04.md` — live timeline, affected order
  IDs, indicators. The record of record; this handoff is not.
- `a41c9e2` — last commit, before the verify rewrite.
- `docs/security/2026-06-threat-model-payments.md` §4 — the webhook trust
  boundary, which is the one that failed.

## Suggested skills
- `incident-response` — eradication and recovery are still open, and the timeline
  needs to keep being written as it happens.
- `security-test-writer` — the forged-signature case needs a regression test that
  fails on `a41c9e2` and passes after the migration, before the edge rule lifts.
- `secrets-and-config-audit` — the old signing secret is revoked at the provider
  but may still sit in CI variables and the staging config.

## Redacted
- PSP dashboard admin credentials — 1Password, vault "Platform".
- Attacker source IP and the 14 affected order IDs — in the incident log,
  section "Indicators". Deliberately not duplicated into a file that travels.
```

**Where it is:** `${TMPDIR:-/tmp}/handoff-orders-api-webhook-incident-20260810-1740.md`.

Not in `docs/security` — that is `output.reports_dir`, it is committed and synced, and a handoff carrying a credential pointer and a live containment state has no business in a repository. Confirmed the path is outside `git rev-parse --show-toplevel` before writing. It disappears on reboot, which is right; the incident log is the durable record.

Send Priya the file, not the transcript. It is redacted; the transcript is not.

She runs this pack, so the skill names in the last section are something she can act on directly. If she did not — a different harness, or an on-call engineer from another team — that section would have been written out as plain instructions instead, because a skill name she cannot invoke is a line she has to guess at while the clock is running.

---

## What this run got right

- Worked the boundary tree in order and named the branch — travel, shift handover — instead of writing a handoff because one was asked for.
- Cut at a real boundary: containment closed, eradication not started.
- Pulled two durable facts out and routed them to `project-memory` before drafting, where they survive the task instead of dying with the file.
- Stated in-flight work as file-level fact, down to which four routes are unmigrated and which test line is red and why — no guessing whether an edit was finished.
- Flagged the edge rule as live containment that must be removed deliberately. Environment state is invisible in the repository and is what the next shift breaks by accident.
- Listed the rejected approaches with reasons, including the one that is both obvious and wrong and had already been tried twice.
- One next action, not a menu.
- Referenced the incident log, the threat model and a commit SHA rather than restating any of them.
- Turned a credential into a placeholder naming the vault, and kept order IDs and the attacker IP out of a file that leaves the machine.
- Wrote to the temporary directory and said why the reports directory was refused.
