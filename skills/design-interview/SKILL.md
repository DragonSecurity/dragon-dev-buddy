---
name: design-interview
description: Interviews you about an unsettled design until every decision has been made deliberately, working a design tree in rounds and asking the whole frontier at once with a recommendation attached to each question. Use when someone says "grill me", "interview me about this", "stress test this plan", "poke holes in this", "help me nail down the design", "I'm not sure what I want yet", "ask me whatever you need", "what am I not thinking of", or before any build where the shape is still argued about. An unasked question is an undocumented assumption, and that is where the abuse case nobody wrote down lives.
---

# Design Interview

A design is a tree of decisions. Most of them never get made — they get guessed, once, by whoever typed first, and the guess hardens into the implementation. This skill makes the tree visible and walks it: every decision branches into the decisions hanging off it, and you work the branches until nothing is left silently assumed.

**The security argument.** An unasked question is an undocumented assumption, and an undocumented assumption is where the abuse case nobody wrote down lives — the tenant boundary nobody named, the retention period nobody chose, the failure mode nobody decided should fail closed. The expensive version is subtler than an omission, though: a decision everyone believes is settled and that is quietly false. "Revocation is immediate" is a decision a team can make in a meeting and ship in a sprint, and it is untrue the moment the bytes are served by a pre-signed URL nobody thought to ask about. The question that would have caught it hangs off a fact rather than an opinion, which is why it never gets asked — nobody is uncertain about it. A threat model built on top of that inherits it, the code inherits it, and neither of them can tell.

Most skills in this pack open with an **Inputs** section that asks once, in one flat list, for whatever is missing. That is the right shape when the design is settled and only the inputs are absent. When the design itself is still argued about, `threat-model`, `secure-feature-build`, `change-window` and `buddy-setup` should run this skill in place of their own Inputs step and carry the settled answers forward — with the reasoning attached, not just the conclusions.

## First-run check

Read `.dragon-buddy/config.json`. This skill can run without it — the interview is the work — but a missing config means you will ask the user for facts the pack already recorded, which is the failure mode this skill exists to avoid. If it is absent, say so and offer `buddy-setup`.

Pull `project.what_it_is`, `project.stack`, `project.primary_language`, `security.exposure`, `security.data_sensitivity`, `security.auth_model`, `security.trust_boundaries`, `skill_level`. Every one of these is an answer you must not ask for. `skill_level` sets how much of the reasoning behind each recommendation you spell out, not how many questions you ask — a beginner gets more explanation, not a shorter interview.

## Inputs

Ask only for what is missing:
- **The subject.** The feature, system, change or plan being designed. One subject; an interview about two things at once produces a frontier that never converges.
- **What the interview closes into** — a spec, a threat model, a change plan, or just shared understanding in this session. This decides the handoff, not the questions.

Nothing else. Asking is the skill; do not front-load a questionnaire before the tree exists.

## Workflow

1. **Read before asking.** Establish what the codebase, the config and the running environment already settle. Every fact you find is a question you do not spend the user's attention on. `references/interview-method.md` lists what is reliably lookupable.

2. **Build the design tree.** Write out the decisions the subject implies, and for each, the decisions that only make sense once it is settled. The tree is yours to hold, not a deliverable — but hold it explicitly, because the frontier is computed from it and a tree kept vaguely in mind produces a frontier that drifts.

3. **Compute the frontier.** The frontier is every decision whose prerequisites are already settled: the questions you can ask *now* without guessing at an answer you have not heard yet. A question whose answer depends on another question still open in this round belongs to a later round.

4. **Dispatch the facts, do not block on them.** When a frontier question needs a fact from the environment — what the schema actually holds, whether that endpoint is authenticated today, which version is deployed — send a subagent to find it. Facts are your job; decisions are the user's. A running exploration is an unsettled prerequisite, so the questions downstream of it drop out of this round. The rest of the frontier goes out now.

5. **Ask the whole frontier in one numbered round.** Every question carries your recommended answer. Then stop and wait. Do not ask one question, absorb the answer, and ask the next — that is a drip, and a design settled by drip converges on whatever you guessed first, because each answer is given without sight of the decisions it constrains.

6. **Recompute and go again.** Each round of answers reshapes the tree: settled decisions push the frontier outward and unblock the questions that were waiting on them. Answers also prune — a decision the user rules out takes its whole subtree with it, and saying so out loud is how the user sees the interview converging. Fold in any subagent reports that landed.

7. **Stop when the frontier is empty.** Every branch visited, nothing left silently assumed. State the settled design back in your own words and get the user's confirmation that you share an understanding of it. Do not start building, writing or changing anything before that confirmation lands.

8. **Close into the handoff.** Convert the settled tree into whatever the interview was for — a spec for `secure-feature-build`, the scope and boundaries for `threat-model`, the plan and rollback for `change-window`, the config for `buddy-setup`. Carry the *reasoning* across, not just the conclusions: a decision that arrives without its why gets re-litigated by the next session.

### Question format

Reproduce this shape exactly. The title makes the round skimmable; the recommendation is what lets a user settle six decisions in one reply instead of six turns.

```markdown
**Q1 — Session storage**

Sessions can live in a signed cookie, in Redis, or in the existing Postgres.
The cookie is stateless but cannot be revoked before expiry, which matters
because you said support needs to kick a compromised account immediately.

**Recommended:** Redis with a 30-minute TTL — revocable, and you already run it.
```

A question without a recommendation makes the user do your thinking. A recommendation without its one-clause reason makes it unarguable, which is worse than not offering one.

Number continuously across the whole interview, never from Q1 again each round. The number is how the user answers — "9 — proxy, I hadn't thought about the redirect" — and how you refer back to a decision three rounds after it was settled. Only give a number to a question you are actually asking now; a question waiting on a dispatched fact is named by its subject until it goes out, because the fact may well change what it asks.

## Output format

One round, in one message, then silence until the user answers:

```markdown
## Round N — <what this round settles>

*Settled in round N-1: <the decisions the previous answers closed, one line each>*
*Fact established: <what a subagent reported since the last round, and what it settled or reopened>*
*Dispatched: <the facts a subagent is currently finding, and the subjects waiting on them — named, not numbered>*

**Q1 — <title>**
<body — the decision, the real options, and what makes them differ here>

**Recommended:** <your answer, with the reason in a clause>

**Q2 — <title>**
...
```

The closing message, once the frontier is empty:

```markdown
## Settled design: <subject>

**Decisions:** <each one, with the reasoning that produced it>
**Facts established:** <what was looked up rather than asked, and by what>
**Ruled out:** <the branches pruned, and why — this is what stops them coming back>
**Open by choice:** <anything deliberately deferred, with what would reopen it>
**Assumptions still standing:** <what remains unverified, stated plainly>

Confirm this matches your understanding and I will <handoff>.
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Interviewed the design of <subject> over <n> rounds; <k> decisions settled, handed off to <next>."`
- `kind`: `docs`
- `skills_used`: `["dragon-dev-buddy:design-interview"]`

Relay the reaction verbatim.

## File output

By default this skill writes nothing. The interview lives in the conversation and its product is a shared understanding.

When the interview closes into a document, the receiving skill owns the file and its location — `threat-model` and `secure-feature-build` write under `output.reports_dir`, `buddy-setup` writes `.dragon-buddy/config.json`. Do not invent a third location for a transcript.

Never modify source during an interview. A decision reached in round two is not settled until the frontier is empty, and code written against it is code written against a moving design.

## Reference library

Load these for depth when the task calls for it:
- `references/interview-method.md`: the frontier algorithm worked through concretely — how to compute it, how to batch a round, what to dispatch versus what to ask — plus the four failure modes that ruin an interview and how to close one into a spec or a threat model.

## Worked example

See `examples/design-interview-run.md` for a three-round interview on a document-sharing feature: the frontier moving between rounds, one fact dispatched to a subagent instead of asked, and the handoff into a threat model. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Nothing was asked that could have been looked up. Environment facts were dispatched to a subagent; config facts were read.
- Each round went out as one numbered batch, and every question in it carried a recommended answer with its reason.
- No question in a round depended on another question in the same round.
- Numbering ran continuously across the interview, and no question was given a number before it was actually asked.
- Dispatched facts did not stall the interview — only the questions downstream of them waited.
- The tree was recomputed between rounds, and pruned branches were named rather than silently dropped.
- Security-relevant decisions — trust boundaries, what fails open versus closed, what is retained and for how long, who can act on whose data — were surfaced as questions rather than assumed.
- The frontier was genuinely empty at the end, and remaining assumptions were stated rather than left implicit.
- The user confirmed shared understanding before anything was built or written.
