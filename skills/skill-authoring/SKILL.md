---
name: skill-authoring
description: Writes and edits the documents an agent consumes — a skill in this pack, an AGENTS.md, a CLAUDE.md, a reference file something points at — against the levers that decide whether the agent reaches the material and takes the same process every run. Use when someone says "add a skill", "write a new skill", "edit this skill", "this skill never fires", "sharpen this description", "our CLAUDE.md is too long", "write an AGENTS.md", "the agent ignores this doc", "should this be one skill or two", or before adding anything under skills/ in this pack. Every always-loaded line is paid on every turn; a line that does not change behavior pays and returns nothing.
---

# Skill Authoring

A document written for an agent is not documentation. Its reader is a context window with a budget and a model that either reaches the material or does not, and the thing you are trying to make predictable is the *process* the agent takes — not the output it produces. The packaging differs between a skill, a `CLAUDE.md`, an `AGENTS.md` and a reference file something points at; the writing does not. The same levers govern all of them.

Two failures account for most bad agent documents. The material is right and never reached, because the pointer to it is worded weakly. Or the material is reached and buried, because the document grew past the point where attention covers it. Both are fixed by the same discipline: sharpen the wording that triggers, disclose the depth behind it, and delete every line that does not change what the agent does.

Security angle: in this pack a skill that does not fire is a control that was never applied, and a review that never ran is indistinguishable from a clean one. `secure-code-review` missing a diff because its description reads like a summary instead of a trigger is a security failure written in prose.

## First-run check

Read `.dragon-buddy/config.json` for `project.name` and `skill_level` — `skill_level` decides how much of the reasoning to narrate as you go. This skill needs nothing else from config, so a missing file is not fatal: say it is absent, offer `buddy-setup`, and carry on.

The real check is the pack itself. Confirm you are at the pack root — `.claude-plugin/plugin.json` and `skills/` are both there — and read one existing skill end to end before writing a line. `refactor-safely` is the structure exemplar. The files on disk are the contract; a remembered version of them is a cache that has already gone stale, and this skill's Workflow is written to be checked against `internal/skillpack/skillpack_test.go` rather than believed.

When the document is not a skill in this pack — a `CLAUDE.md`, an `AGENTS.md`, a doc in another repository — steps 3 and 9 of the Workflow are pack machinery and do not apply. Every other step does.

## Inputs

Ask only for what you cannot read:

- **What the document is for**, stated as the behavior it should produce rather than the topic it covers. "Explains our auth" is a topic. "Stops the agent inventing a second session store" is a behavior.
- **The branches** — the distinct cases it handles. Branches decide what the pointer must trigger on and what can be disclosed, so a document whose branches are unknown cannot be structured.
- **The trigger phrases**, in the words the user actually types. Invented phrasing routes nothing.
- **New skill or an edit.** An edit begins by reading what is there and cutting; a new skill begins by justifying permanent context load.

## Workflow

1. **Decide whether this is a new skill at all.** Every skill in this pack is model-invoked, so its description is loaded on every turn of every session whether or not it fires — that is the price, and it is charged forever. A new branch inside a skill that already owns the subject costs nothing extra. Split off a new skill only when it has a distinct trigger you genuinely type, or when another skill must reach it by name. `references/skill-mechanics.md` covers the model-invoked and user-invoked choice, the router pattern for when user-invoked skills multiply, and why this pack has no user-invoked skills to choose from.

2. **Write the description before the body.** The description is the skill's context pointer: it names material that is out of context and encodes the condition for reaching it, and its *wording* — not its target — decides when the agent gets there. A must-have skill behind a weak pointer is a variance bug, so sharpen wording first and only inline material if sharpening fails.

   Front-load the leading word, because the pointer is where it does its triggering work. Give one trigger per branch; synonyms that rename a single branch are one branch written twice. Cut identity the body already carries.

   The hard limits: one line, no newlines, under 1024 characters, containing the literal string `Use when` followed by quoted trigger phrases. The phrases sit at the end, so a description over the limit loses exactly them and the skill stops firing while still looking complete.

3. **Lay out the frontmatter and the ten headings.** Frontmatter is exactly two flat keys — `name` and `description` — and `name` must equal the directory name, in lowercase kebab-case. Then the body, in this order, spelled exactly:

   ```
   # <Title>
   <intro prose>
   ## First-run check
   ## Inputs
   ## Workflow
   ## Output format
   ## Buddy (optional, when the MCP server is connected)
   ## File output
   ## Reference library
   ## Worked example
   ## Quality bar
   ```

   Sub-headings under any of them are fine; the ten are a floor, not a ceiling.

4. **Write each Workflow step to end on a criterion the agent can check.** Two properties make a criterion work. **Clarity** — can the agent tell done from not-done? A vague bound invites premature completion: the step ends early because attention has slipped to being finished, pulled by the steps still visible ahead of it. **Demand** — how much the criterion requires. "Every modified model accounted for" forces legwork that "produce a change list" does not, and demand binds flat reference as well as sequences: "every rule applied" is a completion criterion for a document with no steps in it.

   Sharpen the bound before splitting the document. Splitting to hide later steps only works across a real context boundary — a hand-off or a subagent dispatch — because an inline call leaves those steps in context and clears nothing.

5. **Push depth behind pointers and keep the top legible.** Material sits on one of three rungs: an in-file step, in-file reference consulted on demand, or a disclosed reference in its own file reached by a pointer. Inline what every branch needs; disclose what only some branches reach. Within a file, keep a concept's definition, rules and caveats under one heading — scattering one meaning across a document is a different disease from duplicating it, and it reads worse. The failure mode at this rung is sprawl: a document too long even though every line is live, where attention thins across the excess.

   Pack mechanics for the disclosed rung: at least one file under `references/`, exactly one worked example named for the skill under `examples/`, and both directions are tested. Every file you create there must be named in SKILL.md prose exactly as it sits on disk, and every path you name must exist. Only `.md` files may live in those directories — a shell script there is unreferenceable and fails the build. Shell sidecars go in the repo-root `scripts/` directory and are invoked through `${CLAUDE_PLUGIN_ROOT}`; `project-memory` ships its commit guard that way.

6. **Write the Buddy section and the Quality bar.** Copy the `**Advise first.**` paragraph verbatim from `refactor-safely`, then close with a `buddy_observe` block whose `skills_used` names this skill and nothing else. That array is what trains the advice ranking, so a skill carrying another skill's qualified name is a skill reporting someone else's work as its own — the tests read it exactly that way.

   The Quality bar is the completion criterion for the whole run, which is where its demand comes from. Write it as claims about a run that went well, in the past tense and checkable one by one, not as aspirations.

7. **Write the worked example.** One real run in dialogue under `examples/`, named for the skill, showing the judgement rather than only the result: what was rejected, and why. Close it with a short "What this run got right" list. It is the quality target, not a script to replay.

8. **Prune the draft before you read it back as its author.** Go sentence by sentence with the no-op test: does this line change behavior versus the model's default? A line that fails pays load and says nothing — delete the whole sentence rather than trimming words out of it. The test is model-relative, so two people disagreeing about a no-op are disagreeing about the default and settle it by running the document.

   Then the rest of the pruning set: one meaning in one place; the environment is a source of truth too, so a document restating a config file is a cache that earns its load only when the lookup is expensive; check each line still bears on what the document does, because the default fate of an unpruned document is sediment. Where you are tempted to forbid a behavior, prompt the positive instead — a prohibition drags the forbidden behavior into context and makes it more available, not less. `references/agent-writing-levers.md` works each lever with before-and-after rewrites, including the collapse of a restated triad into a single leading word.

9. **Register the skill outside its own directory, then run the tests.** A skill nobody can find is not shipped. Add its row to the right group in the README's Skills table, update the count the README spells out in prose, and add a CHANGELOG entry under `## Unreleased`.

   The version lives in two places that must agree: `version` in `.claude-plugin/plugin.json` and the newest release heading in `CHANGELOG.md`. Leave `.claude-plugin/marketplace.json` alone — it pins the release archive by URL and `sha256`, and the release workflow rewrites both on the tag, because the digest cannot be known before the artifact is built.

   `go test ./...` skips `## Unreleased`, so it stays green on a change that never bumped anything. CI does not, and this is the trap: the `Validate the skill pack` job diffs the shipped bundle against the base branch — `skills`, `hooks`, `.mcp.json`, `config.example.json`, `THIRD-PARTY-NOTICES.md` and the three shipped files under `scripts/` — and fails if any of them moved while the version stood still, because an autoupdating marketplace can never pull a change that ships under a version it already has. So editing any skill at all is a version bump in the same pull request, and a green local test run is not evidence you can skip it. A new skill or an added config key is a minor bump: nothing that pointed at the pack has to change.

   Finish with `go test ./...`. It checks the frontmatter, the description limit and its `Use when` clause, both directions of every sidecar reference, the presence of a worked example and a Quality bar, the buddy reporting contract, near-miss skill names in backticks, absolute home paths in any markdown file, and the README row and count. Green is the bound on this step.

### The duplication this pack is carrying

Apply the levers to the pack honestly, starting with the one it fails. Every skill here except `buddy-companion` and `project-memory` carries the same `**Advise first.**` paragraph, word for word — one meaning stored once per skill, which inflates the buddy handshake's rank on the information hierarchy well past what it is worth. `references/agent-writing-levers.md` works it as the duplication rewrite: what the single-source fix is, and why the reporting-contract test leaves it unpaid.

Write it as debt, do not pay it here. An author adding one skill does not get to change a convention in every other skill as a side effect — the diff would be unreviewable, and the skill it came with would be the last place anyone looks for it.

Note the shape of the sentence naming the exceptions: it names `buddy-companion` and `project-memory` rather than a count of the rest. A count is a cache of something the repository can answer, and it goes stale the next time a skill lands.

## Output format

Report the authoring run itself, not the skill's content — the skill is on disk and speaks for itself:

```markdown
## Skill: <name> — <new | edited>

**Decision:** [a new skill, or a branch added to <existing skill>] — [why the context load is worth it]
**Description:** [the final one-liner, and what changed about its triggers]
**Structure:** [what sits in-file, what went behind a pointer, and the branch that justified the split]
**Pruned:** [sentences deleted by the no-op test; passages collapsed into a leading word; duplication removed]
**Registered:** [README row and group, README count, CHANGELOG entry]
**Tests:** [`go test ./...` — stated as run, with the result]
**Left undone:** [debt named and not paid, with the reason]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Authored the <name> skill: <n> triggers in the description, depth disclosed into <n> reference files, pack tests green."`
- `kind`: `docs`
- `skills_used`: `["dragon-dev-buddy:skill-authoring"]`

Relay the reaction verbatim.

## File output

Writes the new skill's directory — `SKILL.md`, its `references/`, its one `examples/` file — plus the registrations that live outside it: the README table row, the README prose count, and the `CHANGELOG.md` entry under `## Unreleased`. A shell sidecar goes to the repo-root `scripts/` directory, never under `references/` or `examples/`.

Never edits another skill's `SKILL.md` as a side effect. Never edits `.claude-plugin/plugin.json` or `.claude-plugin/marketplace.json` outside a deliberate release. Never writes `.dragon-buddy/config.json`.

## Reference library

Load these for depth when the task calls for it:
- `references/agent-writing-levers.md`: the full lever set — context pointers, the two loads, the information hierarchy and progressive disclosure, completion criteria, when to split, leading words and negation, and the pruning set — each worked with a before-and-after rewrite. Load it when writing prose or cutting it.
- `references/skill-mechanics.md`: frontmatter, the model-invoked versus user-invoked choice, splitting by invocation, router skills, and this pack's test contract as a checklist to run a draft against. Load it before creating a skill directory, and again when a test fails.

## Worked example

See `examples/skill-authoring-run.md` for a bloated draft skill critiqued and rewritten — a no-op line deleted, a restated passage collapsed into one leading word, a section pushed behind a pointer, and a description sharpened until it fires. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

A good run satisfies all of these:

- The description was written before the body, and every trigger phrase in it is one a user would actually type. No invented phrasing, and no synonym renaming a branch that is already listed.
- The description is one line, under the character limit, and carries its `Use when` clause with the triggers after it.
- The ten headings are present, in order, spelled exactly, with frontmatter carrying only `name` and `description`.
- Every Workflow step ends on a criterion that can be checked as done or not-done, and at least one of them demands legwork rather than a summary.
- Material only some branches need went behind a pointer; material every branch needs stayed in the file.
- Every reference and example file created is named in SKILL.md prose exactly as it sits on disk, and every path named there exists. Nothing but `.md` was placed under `references/` or `examples/`.
- The Buddy section reports this skill's own qualified name and no other's.
- The draft was pruned: at least one sentence was deleted by the no-op test, and the deletions were named in the report rather than left implied.
- The README row, the README prose count and the CHANGELOG entry landed in the same change as the skill.
- `go test ./...` was run and green before the work was called done, and the version was bumped in the same change if anything under the shipped bundle moved. Local green is not the bound; CI checks the bundle against the base branch.
- Other skills were left untouched. A pack-wide convention change was named as debt, not smuggled into this diff.
