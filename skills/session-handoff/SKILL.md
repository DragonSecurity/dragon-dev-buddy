---
name: session-handoff
description: Compacts the session in flight into a document a fresh agent can pick the work up from cold — the goal, what is done, what is mid-change and its exact state, what was ruled out and why, and the next concrete action. Use when someone says "write a handoff", "hand this off", "I'm running out of context", "pick this up in a new session", "continue this on another machine", "summarise this for another agent", "hand this to the next shift", "brief my colleague on where we are", or "carry this over to Codex". Runs the phase-boundary check first, because continuing, clearing or compacting is usually the better move. Redacts every secret, and writes outside the working tree.
---

# Session Handoff

A session ends and everything it worked out goes with it — which approach was already tried and abandoned, which file is half-edited, why the obvious fix is wrong. The next session starts from the diff and re-derives all of it, badly. This skill writes the part that does not survive: the state of one task in flight, in a form a cold agent can act on within a minute of reading it.

**It is not `project-memory`, and confusing the two ruins both.** `project-memory` records durable facts about the codebase — constraints, rejected designs, gotchas that will still be true next month — and it is loaded into every future session automatically. A handoff records the state of *one task* and is disposable the moment it is picked up. The test is the calendar: if the fact will still be true after this task ships, it is a memory, and it belongs in `project-memory` whether or not you also write a handoff. Route it there when you notice one, out loud, rather than burying it in a file that gets deleted.

The second thing this skill takes seriously is that in this pack a handoff is most often written mid-incident, at a shift change, from a session whose transcript holds live credentials, customer identifiers and attacker infrastructure. Redaction here is a step in the workflow, not a caution at the end of it.

## First-run check

Read `.dragon-buddy/config.json`. A handoff is frequently written under time pressure, so a missing config is not fatal — work from what the session already knows, say so, and offer `dragon-dev-buddy:buddy-setup` afterwards rather than stopping to run it now. Pull if present: `project.name` for the header, `security.data_sensitivity` and `engagement.authorized_scope` to set how aggressive the redaction pass has to be, `engagement.contact` for who is allowed to receive the file, and `output.reports_dir` **so you can avoid it** — that directory is committed and shared, and a handoff never goes there.

## Inputs

Most of the input is the conversation you are already in. Ask only for:

- **What the next session is for**, if the user did not say. A handoff aimed at "finish the migration" and one aimed at "review what I did" keep different things.
- **Where it is going** — same machine and a fresh window, another harness, another machine, or a colleague. This decides both the paths you can use (a `/private/tmp` path is useless to a colleague) and how much local context has to be spelled out.
- **Whether anything in scope is under NDA or belongs to a client**, if the config does not already say.

Do not ask for the goal, the file list, or what was tried. You were there.

## Workflow

1. **Check that a handoff is the right move.** This is the step that stops the skill being invoked reflexively. Make the decision **at a phase boundary** — the seam between grilling and building, building and QA, containment and eradication — never mid-edit. Work the tree top to bottom; the first yes wins.

   1. **Can you just continue?** If the next phase wants this session as a primary source, or there is enough headroom left for it to fit, stay. Continuing costs nothing and loses nothing, so rule it out first.
   2. **Is everything here disposable?** Then clear. Cheapest move available, and the old session is still resumable.
   3. **Does something have to travel?** A new harness, a different machine or repository, a colleague, a shift handover on a live incident, or a side task you want forked off without derailing this one. Portability is the whole reason a handoff exists. **This is the only branch that lands on this skill.**
   4. **Can the work run unattended?** Scope it and dispatch a subagent; this session stays intact.
   5. **Otherwise compact**, with an instruction saying what the next phase needs.

   Say which branch you took and why, in one line, before writing anything. `references/handoff-contents.md` has the tree in full with the reasoning behind each branch.

2. **Route the durable facts out first.** Walk what this session learned and separate the two kinds. "The staging database rejects connections without the proxy" is durable — hand it to `project-memory`. "I have the proxy running on port 5433 in tab two" is task state — it goes in the handoff. Do this before drafting, or the durable facts get written into a file that is deleted on pickup and learned again next month.

3. **Draft what a cold reader needs**, in this order: the goal in one sentence; what is done; what is in flight and its **exact** state; what was ruled out and why; the next concrete action. The in-flight section is where handoffs fail — "refactoring the auth middleware" is not a state, "`src/auth/mw.ts` has the new signature applied to three of five call sites, `billing.ts` and `webhook.ts` still call the old one and the build is red" is.

4. **Reference, do not restate.** Anything already captured in an artifact — a spec, a threat model, an ADR, an issue, a commit, a diff, a report under `output.reports_dir` — is named by path, SHA or URL, never copied in. A handoff that restates the diff is longer than the diff, goes stale the moment someone commits, and buries the five lines that were actually only in your head.

5. **Redact. Every run, no exceptions.** Go through the draft looking for what the transcript put there: API keys, tokens, passwords, session cookies, connection strings with credentials in them, private IPs and hostnames you are not authorised to spread, customer names and identifiers, attacker infrastructure, and anything from an engagement under NDA. Every secret value becomes a named placeholder (`<REDACTED: staging DB password, in 1Password under "staging-db">`) so the next session knows what to fetch and from where. People under NDA are named by role — "the customer's platform lead" — not by identity. The checklist is in `references/handoff-contents.md`; walk it rather than eyeballing.

   A redacted placeholder is more useful than the real value anyway: the next session needs to know a credential is required and where it lives, not to carry a copy of it around in a file.

6. **Name the skills the next session should invoke.** A "Suggested skills" section, each with the reason it applies here — `debug-and-fix` because the build is red for a known reason, `incident-response` because containment is still open, `refactor-safely` because the net exists and the change is structural. This is what makes a cold pickup start correctly instead of improvising a plan.

   If the handoff is leaving this pack — another harness, or a colleague who has not installed it — write that section as plain instructions instead: "verify the fix with a test that fails on the current commit and passes after". A bare skill name is an instruction the reader cannot follow, and it is the one part of the format that does not travel.

7. **Write it to the OS temporary directory, and nowhere else.**

   ```sh
   handoff="${TMPDIR:-/tmp}/handoff-<slug>-$(date +%Y%m%d-%H%M).md"
   ```

   Never into the working tree, and never into `output.reports_dir`. The working tree gets committed, backed up, synced and screen-shared; `output.reports_dir` is written specifically to be handed to other people. A mid-incident handoff placed in either becomes a permanent record of an incident in a repository, and the redaction pass in step 5 is the only thing standing between that and a second incident. Confirm the destination is outside the repository before writing; if the check below errors, you are not inside a repository at all, which settles it:

   ```sh
   git rev-parse --show-toplevel   # the handoff path must not start with this
   ```

   Temporary directories are cleared on reboot, which is correct — a handoff not picked up in a week describes a codebase that has moved on.

8. **Hand over the path and one line of orientation.** Print the absolute path, say what the next session should do first, and note that the file self-destructs with the temp directory. If it is going to another machine or a colleague, say so plainly — the file has to be moved deliberately, and a redacted handoff is safe to send while an unredacted one is not.

## Output format

```markdown
# Handoff: <task> — <YYYY-MM-DD HH:MM>

**For:** <what the next session is meant to do>
**Repository / branch:** <name> @ <branch>, at <commit SHA or "uncommitted">

## Goal
<One sentence. Why this work is happening, not what was typed.>

## Done
- <thing>, see <path or SHA>

## In flight
<Exact state. Which files are half-changed, what is red, what is running where,
what is deliberately left broken. Enough that the next session does not have to
guess whether an edit was finished.>

## Ruled out
- <approach> — <why it failed or was rejected>

## Next action
<One concrete thing to do first, not a list of options.>

## Artifacts
- <path / URL / SHA> — <what it holds>

## Suggested skills
- `<skill>` — <why it applies here>

## Redacted
- <what was removed, and where the real value lives>
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Wrote a handoff for <task> at a phase boundary: <n> secrets redacted, <n> facts routed to project memory."`
- `kind`: `docs`
- `skills_used`: `["dragon-dev-buddy:session-handoff"]`

Relay the reaction verbatim.

## File output

One markdown file in the OS temporary directory (`${TMPDIR:-/tmp}`). Nothing else is created or modified.

Never writes into the working tree, never into `output.reports_dir`, never commits, and never prints the handoff body into a chat transcript that will outlive it — the path is the deliverable. Durable facts noticed along the way are written by `project-memory`, not here.

## Reference library

Load these for depth when the task calls for it:
- `references/handoff-contents.md`: what a handoff must carry to be picked up cold, the phase-boundary decision tree in full with the reasoning behind each branch, the redaction checklist worked line by line, and the four ways handoffs fail — restating the diff, omitting the rejected approaches, going stale before it is read, and leaking what the transcript held.

## Worked example

See `examples/session-handoff-run.md` for a handoff written mid-incident at a shift change: the boundary check, a live credential turned into a placeholder, a durable fact routed to `project-memory` instead of into the file, and a suggested-skills section that starts the next session in the right place. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The phase-boundary tree was worked in order and the branch that landed on a handoff was named. A handoff written because it was asked for, with continuing or compacting never considered, does not clear this bar.
- The handoff was written at a boundary, not in the middle of an edit.
- Durable facts were separated out and routed to `project-memory` before drafting; the file that was written holds only task state.
- The in-flight section stated exact file-level state rather than an activity. A reader could tell which edits were finished, and which failure was expected, without opening anything.
- Rejected approaches were listed with what killed each one, so the next session does not spend the same hours reaching the same wall.
- The handoff ended in one concrete next action, not a menu of options.
- Nothing already in a spec, ADR, issue, commit or diff was restated; each was referenced by path, SHA or URL.
- Every secret became a placeholder naming what it is and where the real value lives. No key, token, password, connection string, customer identifier or NDA-covered name survived into the file.
- The file was written to the OS temporary directory, outside the repository, and not into `output.reports_dir` — verified against the repository root, not assumed.
- The suggested-skills section named skills that exist in this pack, each with the reason it applies — or, where the handoff was leaving this pack, spelled the same guidance out as instructions a reader without it can follow.
