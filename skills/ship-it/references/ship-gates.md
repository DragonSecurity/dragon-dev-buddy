# Ship gates

## Thresholds

| Gate | Pass | Warn | Fail (no-ship) |
| --- | --- | --- | --- |
| Build | clean | — | any error |
| Tests | all green | flaky, named; slow | any real failure; suite not run |
| Sensitive diff | none, or reviewed | changed, review recommended | changed on the money/auth path, unreviewed, going to `public` prod |
| Secrets | none in diff | — | any credential in the diff |
| Migrations | reversible + backward-compatible, or n/a | reversible but needs care | irreversible with no backup; breaks running code |
| Blast radius | bounded, flagged/gradual | wide but understood | unbounded and not understood |
| Rollback | real plan with a time | plan exists but revert carries risk | no credible rollback + wide blast radius |

Warns do not block on their own. Warns *accumulate*: three warns on a Friday-afternoon prod deploy of a 3000-line diff is a no-ship even though no single gate failed. Say when you are escalating on accumulated risk.

## Zero-downtime migration patterns

A migration is backward-compatible when the currently-running code keeps working after it applies but before the new code deploys. This is what makes a deploy safe to roll back — the old code can run against the new schema.

**Safe (expand):**
- Add a nullable column, or one with a default.
- Add a new table.
- Add an index concurrently (Postgres `CREATE INDEX CONCURRENTLY` — a plain `CREATE INDEX` locks the table).
- Backfill in batches, separately from the schema change.

**Dangerous (contract) — split across releases:**
- Dropping a column: first deploy code that stops using it, then drop it a release later. Never in the same deploy as the code change.
- Renaming: expand (add new, write both, backfill), migrate reads, then contract (drop old). Three releases, not one rename.
- `NOT NULL` on an existing column: add it nullable, backfill, then add the constraint.
- Narrowing a type or changing a default: same expand/contract split.

**The rule:** a single deploy should never contain both "the code stops writing column X" and "column X is dropped." If it does, a rollback to the previous code hits a missing column. Split it.

## Blast radius estimation

Name concretely what breaks if this deploy is wrong.

- **What imports the changed code?** A leaf handler affects one endpoint. A shared library, an auth middleware, or a base model affects everything downstream — say how much.
- **What data does it touch?** Read-only is recoverable. A write, a migration, or a delete is not, cleanly.
- **Is it gated?** Behind a feature flag, deployable dark, or does it take effect for everyone the instant it lands?
- **Can it roll out gradually?** Canary, percentage, one region first — or is it all-at-once?
- **What is downstream?** A change that sends a webhook, enqueues a job, or writes a message means the effect escapes the system and cannot be rolled back by reverting the deploy.

The output is one or two concrete sentences: "This changes the shared auth middleware that all 34 routes use; if the token check is wrong, every authenticated request fails closed — an outage, not a breach. No flag, all-at-once, but it fails safe." That lets the user weigh it. "Low risk" does not.

## Rollback patterns by deploy type

| Deploy type | Rollback |
| --- | --- |
| Stateless service | Redeploy the previous image/version. Fast and clean if no migration ran. |
| With an additive migration | Revert the code; leave the migration (it is backward-compatible, so old code tolerates it). |
| With a destructive migration | This is why destructive migrations split across releases — otherwise rollback needs a restore, which is slow and loses data written since. |
| Behind a flag | Flip the flag off. The fastest rollback there is; the deploy stays, the behavior reverts. |
| Config change | Revert the config. Watch for cached values that outlive the change. |
| Anything that sent a message / called a webhook / charged a card | The side effect already left. Rollback reverts the code, not the effect — plan the compensating action (refund, retraction, correction) separately. |

"We can roll back" must survive the question "how, and what does the rollback itself break." If it does not, that is the finding.

## Release shapes that warrant extra caution

Not blockers, but each turns nearby warns into no-ships faster:

- **Friday or pre-holiday deploys.** Fewer people around when it breaks. A warn on a Friday is worth two on a Tuesday.
- **First deploy of a new service or path.** No production baseline; unknown unknowns.
- **A deploy bundling many changes.** Large diffs hide the risky line among the safe ones and make rollback coarse — you cannot revert just the bad part.
- **A deploy that also changes infrastructure or CI.** Two variables at once; if it breaks you do not know which.
- **A dependency major-version bump on a hot path.** The changelog is not the whole story; behavior changes that were not documented surface in prod.
- **Anything touching money movement or auth**, going to `public` exposure. The bar is simply higher; an unreviewed sensitive change here is a no-ship, not a warn.

Name the shape when it applies. "This is a Friday prod deploy of a first-time payment path with a destructive migration" is three cautions stacked, and the verdict should reflect the stack even if each gate individually only warned.
