# What a handoff carries, when to write one, and how it leaks

## The phase-boundary decision tree

A **phase** is a chunk of work inside a session: the interview, the build, the QA, the containment. It ends when you think "right, that's that". The **boundary** between two phases is the only place this decision belongs — mid-phase there is nothing to decide, because compacting or handing off in the middle of an edit loses the thread of the edit itself. Mid-phase you either continue, or split what is left into subagents.

Five options are on the board:

| Option | What it does |
| --- | --- |
| **Continue** | Stay in the session. No context switch at all. |
| **Clear** | Empty the window and start from nothing. |
| **Handoff** | Write a portable markdown file that can seed a session anywhere. |
| **Subagent** | Send a scoped task to its own window and get a report back. |
| **Compact** | Compress this context and carry on from the summary. |

Work top to bottom. The first yes wins.

**1. Can you continue in this session?** Yes if the next phase needs this one as a *primary source* — the reasoning verbatim, not a summary of it — or if there is simply enough headroom left for the next phase to fit. Threat model into implementation is the standard yes: the implementation wants the actual argument about the trust boundary, not a bullet that says one exists. Continuing costs nothing and loses nothing, so rule it out before anything else.

**2. Is everything here irrelevant to what comes next?** If the exploration, the decisions and the dead ends are all disposable, clear. It is the cheapest move available and it is not terminal — the old session stays resumable. The cost of getting this one wrong is one-way, though: clear a *relevant* context and the **why** behind what you built is gone, and reading the diff back does not return it.

**3. Does something have to travel?** This is the branch that lands on a handoff, and it is narrow:

- a **new harness** — moving the work from one agent to another,
- a **new machine, directory or repository**,
- a **colleague**, who has your files but not your window,
- a **shift handover on a live incident**, where the person taking over is a different person on a different machine and the clock does not stop for them,
- or forking a **side task you found mid-phase** without derailing what you are doing.

That list is the whole clause. What a handoff buys is **portability**: a file that travels. If nothing is travelling, you do not need one.

**4. Can the task run unattended?** Scoped tightly enough that nobody has to steer it? Send it to a subagent and leave this session untouched. Reviewing a diff is the standard case.

**5. Otherwise, compact.** Relevant context, same harness, same directory, and you need to stay in the loop — the tree lands here often. Pass an instruction with it so the summary keeps what the next phase actually needs.

Compact is the **default, not the first reach**. It sits at the bottom because the four questions above it are each cheaper or more precise. The failure mode of starting here is a fresh session that is confidently wrong about a decision the summary flattened.

### Primary and secondary sources

Every move except *continue* turns a primary source — the session as it happened — into a secondary one.

| Source | Information | Noise | Room to move |
| --- | --- | --- | --- |
| Primary (continue) | Full | Lots | Little |
| Secondary (compact, handoff) | Lossy | Less | Lots |

That trade is why question 1 comes first. You only pay the lossiness when staying costs more than it saves.

These are judgement calls. The same boundary can go two ways on two days, and the value is in asking the questions **in order**, at the boundary, rather than reaching for the file because the session feels long.

## What the handoff has to carry

Five things. A cold reader who has never seen the conversation should be able to act inside a minute.

**The goal, in one sentence.** Why the work is happening, not what was typed into the prompt. "Stop the webhook endpoint accepting unsigned payloads" survives a change of plan; "do what the user asked in message four" does not.

**What is done.** Each item with the artifact that proves it — a commit SHA, a merged PR, a path. If it is in git, one line and a SHA is the whole entry.

**What is in flight, and its exact state.** This is the section that decides whether the handoff works. An activity is not a state. Write file-level facts:

> `src/webhooks/verify.ts` has the HMAC comparison rewritten to constant-time and is finished. `src/webhooks/router.ts` is half-migrated — two of six routes call `verify()`, the other four still trust the `X-Signature` header directly, and the suite is red on `router.test.ts:88` for that reason and no other.

The next session must not have to guess whether an edit was completed, whether a red test is your red test or a pre-existing one, or whether something is deliberately broken.

Include the live environment too, where it is load-bearing: a tunnel open on a port, a feature flag flipped in staging, a container left running, a rule temporarily added to a firewall. Those are invisible in the repository and expensive to rediscover — and a temporary firewall rule nobody removes is its own finding.

**What was ruled out, and why.** Without this the next session tries them all again and reaches the same walls at the same cost. One line each: the approach, and what killed it. "Tried moving verification into middleware — the raw body is already consumed by the JSON parser at that point, so the HMAC never matches."

**The next concrete action.** One thing, not a menu. A handoff ending in three options hands the decision back to a reader with less context than the writer had, which is exactly backwards.

Then two sections that carry the pack's own conventions: **Artifacts**, listing every spec, ADR, issue, report and commit by path or URL; and **Suggested skills**, naming which skills the next session should invoke and why each applies.

## What it must not carry

**Anything already captured somewhere else.** Specs, plans, ADRs, issues, commits, diffs, reports under `output.reports_dir`. Reference them by path, SHA or URL. The handoff is for the part that exists only in the session.

**Durable facts about the codebase.** Those belong in `project-memory`, which loads them into every future session automatically. A constraint, a rejected design, a gotcha with a delayed failure — written into a handoff, it is deleted the moment the task is picked up and rediscovered next quarter. The calendar is the test: still true after this task ships, it is a memory.

**Secrets.** See below.

## The redaction checklist

Walk it. Do not eyeball it. A transcript accumulates far more than you remember putting there, and the risk is highest exactly when the handoff is most useful — mid-incident, where the session has been reading logs, decoding tokens and enumerating attacker infrastructure for an hour.

- **Credentials of any kind.** API keys, bearer tokens, session cookies, JWTs (the payload is readable and often identifies a person), passwords, private keys, connection strings with credentials embedded, cloud access key pairs, webhook signing secrets.
- **Anything pasted from an environment.** `.env` contents, `printenv` output, a Kubernetes secret, a CI variable, a `curl -v` transcript with an `Authorization` header in it.
- **Customer and personal data.** Names, email addresses, account and user IDs, order numbers, IP addresses belonging to real users, anything read out of a production database while investigating.
- **Client identity under NDA.** Where `engagement.authorized_scope` is populated, the client's name, their hostnames, their internal project names, their staff. People are named by role: "the customer's platform lead", "the on-call engineer".
- **Attacker infrastructure and indicators**, when the handoff might travel outside the response team. IPs, domains, hashes and paths are needed by the next responder and are not for wider circulation.
- **Internal topology you are not authorised to spread** — private ranges, jump host names, VPN endpoints, the layout of a segmented network.

Every removal becomes a named placeholder that says what it was and where to get it:

```
DB_PASSWORD=<REDACTED: staging Postgres password — 1Password, vault "Platform", item "staging-db">
Attacker source: <REDACTED: single IPv4, recorded in the incident log, section "Indicators">
```

The placeholder is strictly more useful than the value. It tells the next session that a credential is needed, which one, and where it lives — while the value itself would only have created a second copy to rotate.

Add a **Redacted** section listing what was removed. Silent redaction reads as an omission, and the next session cannot tell whether a credential is missing or was never needed.

## Where the file goes

The OS temporary directory: `${TMPDIR:-/tmp}`. Never the working tree, never `output.reports_dir`.

The working tree is committed, pushed, backed up, synced to cloud storage and screen-shared in stand-up. A file dropped there is a file you have published, and the delay between writing it and noticing is usually a `git add -A`. `output.reports_dir` is worse in one specific way: it exists to be handed to other people, so a handoff placed there is not merely at risk of distribution, it is queued for it.

Confirm rather than assume:

```sh
git rev-parse --show-toplevel   # the handoff path must not sit under this
```

Temp is also where a handoff *should* expire. It is cleared on reboot, and a handoff older than a few days describes a repository that has moved on — the correct fate for it is deletion, not a stale file in a docs directory that someone trusts in March.

## How handoffs fail

**It restates the diff.** Four hundred lines paraphrasing changes that are already committed. It is longer than the source, goes stale on the next commit, and buries the five lines that existed only in your head. The fix is a SHA.

**It omits the rejected approaches.** The most expensive omission there is, because the next session spends the same hours reaching the same wall, and often *lands on the rejected approach* — it is usually the obvious one, which is why it was tried first.

**It is stale before it is read.** Written at the start of a phase rather than the end, or written and then worked past. A handoff describes the state at the moment it was written; if you keep going afterwards, rewrite it or throw it away. Timestamp it and pin the commit SHA so a reader can tell how far reality has moved.

**It leaks.** The transcript held a live key, the redaction pass was skipped because the incident was loud, and the file went into the repository because that is where the editor was pointing. This is the failure this skill is shaped around, and it is why redaction is a numbered step and the destination is checked against the repository root.

**It is written instead of continuing.** The quiet one. Nothing was travelling, the session had headroom, and a lossy summary replaced a primary source for no gain. Question 1 of the tree exists for this.
