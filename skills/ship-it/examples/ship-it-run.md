# Worked example: a no-ship, then a clean ship

The point of this skill is being willing to say no. Both verdicts shown.

---

## Run 1: the gate says no

**User:** ship it — merging the payout batching change to prod

**Assistant** *(reads the diff against `main`, runs build and tests, scans for secrets, checks the migration)*

## Ship check: payout-batching → production

| Gate | Result | Detail |
| --- | --- | --- |
| Build | pass | `npm run build` clean |
| Tests | **fail** | `npm test`: 1 failing — `payout batches respect the per-tenant cap`. Not flaky; fails every run. |
| Sensitive diff | **warn** | Changes `src/payouts/batch.ts` — money movement path. No review on record. |
| Secrets | pass | none in diff |
| Migrations | **fail** | Adds `payout_batches`, and in the same deploy **drops `payouts.legacy_ref`**. The running code still reads `legacy_ref` at `reconcile.ts:31`. |
| Blast radius | — | Money movement. If batching is wrong, payouts are wrong — wrong amounts or double payouts. Not flagged, all-at-once. |
| Rollback | **fail** | Reverting the code hits a dropped `legacy_ref` column. Rollback is now a DB restore, losing anything written since deploy. |

## Verdict: **DO NOT SHIP**

Three hard failures, any one of which blocks:

1. **A test is failing** — and it is *the* test, the per-tenant payout cap. On a money path that is the last thing you ship red.
2. **The migration drops a column the running code still reads.** The instant this deploys, the old still-running instances start throwing at `reconcile.ts:31` until the new code is fully rolled out — and if you need to roll back, the old code has no column to read. This is the classic destructive-migration-in-one-deploy trap.
3. **Therefore there is no clean rollback.** Once `legacy_ref` is dropped, reverting the deploy needs a restore.

What clears it:
- Fix the failing cap test (it may be telling you the batching logic is genuinely wrong — on a payout path, assume it is until proven otherwise).
- Split the migration: this deploy *adds* `payout_batches` only. Stop reading `legacy_ref` in code. A **later** deploy drops the column. Now every step is reversible.
- Run `secure-code-review` on `batch.ts` before it moves money. It's unreviewed and it's the payout path.

I can help with any of those. What I won't do is call this shippable.

---

**User:** ugh you're right about the migration. fixed the test (batching was off by one on the cap), split the migration to additive-only, got the review. re-check?

**Assistant** *(re-reads the diff, re-runs everything)*

## Ship check: payout-batching (rev 2) → production

| Gate | Result | Detail |
| --- | --- | --- |
| Build | pass | clean |
| Tests | pass | 89 green, including the cap test that was red |
| Sensitive diff | pass | `batch.ts` reviewed — `secure-code-review` run today, 0 findings on the money path, 1 low (log verbosity) noted not blocking |
| Secrets | pass | none |
| Migrations | pass | additive only — adds `payout_batches`, drops nothing. Backward-compatible. |
| Blast radius | — | Still the payout path, but now behind `FLAG_BATCH_PAYOUTS`, default off. Deploys dark; you enable it per-tenant when ready. |
| Rollback | pass | Flag off is instant. Code revert is clean — migration is additive, old code ignores the new table. ~2 min. |

## Verdict: **SHIP WITH CONDITIONS**

Go. Two conditions:

1. **Deploy with `FLAG_BATCH_PAYOUTS` off**, which is its default — confirm it's off in the prod config before you push, not just in the code.
2. **After enabling it for the first real tenant, watch the payout reconciliation** for one cycle before rolling wider. The test proves the cap; the flag lets you prove it on one tenant's real money before all of them.

The off-by-one is exactly the kind of bug that passes review and fails in production, so the gradual rollout is doing real work here, not ceremony.

*(calls `buddy_observe`, `summary: "Ship check on payout-batching: ship-with-conditions after a no-ship caught a destructive migration and a red cap test."`, `kind: "deploy"`, `skills_used: ["dragon-dev-buddy:ship-it"]`, relays the reaction)*

---

## What this run got right

- **Said no, clearly, with three specific reasons** — the whole value of the skill.
- Actually ran the build and tests rather than trusting "they passed earlier," and caught the red cap test.
- Caught the destructive-migration-in-one-deploy trap and explained *why* it breaks both the running code and the rollback — the non-obvious part.
- Connected the failing test to a real bug (off-by-one on the cap) rather than treating it as noise to silence.
- On re-check, gave a real verdict again — ship-with-conditions, not a relieved yes — and made the conditions specific and tied to the actual risk (an off-by-one that survived review).
- Reported the check to the buddy even though the first pass was a no-ship, because catching it was the work.
