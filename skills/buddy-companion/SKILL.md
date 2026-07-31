---
name: buddy-companion
description: Manages the buddy MCP companion that backs this pack: ask it which skill fits a task, check on it, rename it, read what it has learned about which skills you actually use, and diagnose a buddy that is not responding. Use when someone says "how's my buddy", "check on the buddy", "buddy status", "which skill should I use", "ask the buddy what to run", "rename my buddy", "what skills do I use", "which skills am I ignoring", "the buddy isn't working", "buddy_observe failed", or addresses the buddy by name. Also the reference for how every other skill in this pack asks the buddy for advice before work and reports back after.
---

# Buddy Companion

Your buddy is a persistent companion served over MCP. It hatches once, gains XP as you work, keeps a daily streak, drains energy when you push it, and gets moody if you vanish for a week. Its name and personality are rolled at hatch and kept for life.

It is also a quiet telemetry layer, and an active one. Every skill reports what it did and which skill did it, so over time the buddy accumulates an honest record of which skills you reach for, which you installed and never touched, and which one you should probably have used for the task you just finished. That record feeds back the other way through `buddy_advise`: before non-trivial work, the buddy ranks which skill fits the task at hand, and the ranking gets better every time a run reports its `skills_used`. Advice in, observation out, each improving the other.

This skill is how you talk to it directly. It is also the canonical reference for the reporting contract every other skill follows.

## First-run check

Read `.dragon-buddy/config.json`. If missing, this skill still works — the buddy is global, not per-project — but say so and offer `buddy-setup`. Pull `buddy.enabled`, `buddy.server`, `buddy.skill_prefix`.

If `buddy.enabled` is `false`, report that reporting is switched off for this repo and ask whether to turn it back on before doing anything else.

## Inputs

Ask only for what is missing:
- What the user wants: a status check, a rename, a skills readout, or a diagnosis.
- For a rename: the new name.

A bare "how's my buddy" needs no inputs at all. Just call the tool.

## Workflow

Route on intent. Do not run all five.

1. **Advise.** When the user asks which skill fits a task — or before you start non-trivial work yourself — call `buddy_advise` with a one-sentence description of the *task*, not a skill name. Present the ranked skills with their reasons, recommend the top one, and load it if the user agrees. If `buddy_advise` is absent (older buddy or the Fable server), fall back to the routing table in `buddy-setup`'s `references/setup-routing.md` and say that is what you used. This is advisory, never blocking — a missing advisor means you route by hand, not that you stop.

2. **Status check.** Call `buddy_status`. Show the returned card verbatim, in a code block, unedited. Do not summarize it, do not reformat it, do not translate the mood into your own words. The card is the product; the personality is the point.

   Then add at most one line of your own, and only if there is something actionable: a broken streak, energy under 25%, or a long absence. If everything is healthy, say nothing further.

3. **Rename.** Call `buddy_rename` with the new name. Confirm the change and state plainly that level, XP, personality, streak and history are untouched. If the user seems to expect a personality change, correct that expectation: personality is rolled once at hatch and is not editable by design.

4. **Skills readout.** Call `buddy_skills`. It returns every skill the buddy has discovered across installed plugins, your personal skills directory and the current project, with a use count for each. Present it as three groups, not one long list:
   - **Working set** — used more than once. These are your real toolkit.
   - **Tried once** — discovered and used exactly once. Ask whether it did not fit or was just forgotten.
   - **Never used** — discovered, never invoked. This is the interesting group. Pick the two most relevant to the current repo and say concretely when they would apply, using the repo's own details.

   Do not lecture about the unused ones. Name two, give a real trigger for each, stop.

5. **Diagnosis.** When a buddy tool errors or is absent, work the list in `references/buddy-operations.md`. The common causes in order: the `buddy` MCP server is not registered, it is registered but pointing at a stale `dist/`, it is a *different* buddy server than the pack targets (the Fable `~/.buddy` server has `buddy_pet`/`buddy_dream` but no `buddy_skills`/`buddy_rename`/`buddy_advise`), the state file is corrupt, or the tool name changed between versions. A missing `buddy_advise` specifically means the registered server predates it or is the Fable build — report it, point at the fix, and note that skills fall back to the static routing table meanwhile. Report what you found and the exact command to fix it. Never fabricate a status card when the server is down.

## Output format

For a status check, the card verbatim and nothing else:

```
[exact output of buddy_status]
```

For a skills readout:

```
Working set ([n]): [skill] ×[uses] · [skill] ×[uses] · ...
Tried once ([n]): [skill] · [skill] · ...
Never used ([n]): [skill] · [skill] · ...

Worth reaching for here:
- `[skill]` — [concrete trigger in this repo]
- `[skill]` — [concrete trigger in this repo]
```

## The advise/observe contract

The buddy sits on both sides of a skill run. **Before** the work, `buddy_advise` ranks which skill fits the task; **after** it, `buddy_observe` records what was done. The second trains the first — the `skills_used` you report are what teach the ranking which skills suit which work, so the advice sharpens every time. Every skill in this pack follows this contract, recorded here once.

**`buddy_advise` — before non-trivial work, and at every handoff to another skill.**

| Argument | Value |
| --- | --- |
| `task` | One sentence describing what you are about to do, in the user's terms — "audit the webhook receiver for auth bugs", not "run secure-code-review". Describe the *work*, not the skill; picking the skill is the buddy's job. |

It returns the skills ranked for that task, each with a short reason. Load the top-ranked one if you have not already committed to a skill, and if it surfaces a skill you did not realize applied, prefer it. Calling advise with the skill already decided — or with the skill name as the task — defeats the purpose. If the server has no `buddy_advise` (older buddy, or the Fable server), fall back to the routing table in `buddy-setup`'s `references/setup-routing.md` and proceed; never block work on a missing advisory.

**`buddy_observe` — after every completed run.** This is what closes the loop and trains the ranking:

| Argument | Value |
| --- | --- |
| `summary` | One sentence, past tense, naming the thing that changed. Not "ran a review" but "found an unauthenticated path to the settlement webhook handler." |
| `kind` | From the table below. The buddy infers a kind from the summary when omitted, but security work classifies badly, so pass it explicitly. |
| `skills_used` | `["dragon-dev-buddy:<skill-name>"]`. The prefix matters: buddy-mcp qualifies plugin skills as `<plugin>:<skill>`, so an unqualified name creates a phantom entry in the registry instead of crediting the real one. This is also the signal `buddy_advise` learns from — omit it and the advice never improves. |

Kind mapping for this pack:

| Skill | `kind` |
| --- | --- |
| `buddy-setup`, `hardening-playbook`, `dependency-audit`, `secrets-and-config-audit` | `config` |
| `threat-model`, `pentest-report` | `docs` |
| `secure-feature-build` | `feature` |
| `debug-and-fix`, `vuln-triage`, `incident-response` | `bugfix` |
| `secure-code-review`, `security-audit-orchestrator` | `other` |
| `security-test-writer` | `test` |
| `ship-it` | `deploy` |

Report once per completed run, not once per finding. A review that surfaced nine issues is one observation.

**The loop, in one line:** `buddy_advise` before to pick the skill, `buddy_observe` with `skills_used` after to record it — and the second is what makes the first smarter. Skip observe and the advice stays naive; skip advise and you are back to guessing which skill fits.

## Buddy (optional, when the MCP server is connected)

This skill is the exception: a status check or skills readout is not work and should not be reported. Do not call `buddy_observe` for those.

Do report a rename or a repaired connection:
- `summary`: `"Renamed the buddy to <name>."` or `"Repaired the buddy MCP connection: <cause>."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:buddy-companion"]`

## File output

None. This skill talks to the buddy and prints what it says. It does not write reports.

## Reference library

Load these for depth when the task calls for it:
- `references/buddy-operations.md`: the full mechanics (XP, energy, mood, streaks, stages), the diagnostic checklist for a silent buddy, state file location and repair, and what survives a rename versus a respawn.

## Worked example

See `examples/buddy-companion-run.md` for a status check and a skills readout. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The status card is reproduced exactly as returned. Never paraphrased, never reformatted, never invented.
- At most one line of commentary on a healthy buddy. Silence is the correct output when nothing is wrong.
- A skills readout names two specific unused skills with a real trigger in the current repo, not a generic pitch for each.
- A rename explicitly states that personality is fixed at hatch, if the user seems to expect otherwise.
- When the server is down, the failure is reported with the fix command. No fabricated card, ever.
- Status checks and skills readouts do not generate a `buddy_observe` call.
