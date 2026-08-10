---
name: refactor-safely
description: Changes the structure of code without changing its behavior, under a test net that proves behavior held. Use when someone says "refactor this", "clean this up", "this function is too big", "extract this", "reduce duplication", "make this more maintainable", "modernize this code", or "improve this without breaking it". The invariant is behavior; if behavior changes, it was not a refactor, it was an edit that needs review.
---

# Refactor Safely

Refactoring is changing how code is organized while keeping what it does identical. The danger is that "identical" is an assumption, and the assumption is usually only tested by whether anyone notices in production. This skill refactors the way it should be done: establish a net that captures current behavior, change structure under it, and use the net to prove behavior held. If there is no net, building one is the first half of the job.

Security angle: refactors silently drop security controls more than almost any other activity. An authorization check that looks like clutter gets "cleaned up." A validation step gets lost in an extraction. The net is what catches that.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.stack`, `project.primary_language`, `practice.test_command`, `practice.build_command`, `practice.lint_command`, `skill_level`.

## Inputs

Ask only for what is missing:
- **What to refactor**, and **why** — the specific pain. "Too big" and "hard to test" and "duplicated three times" lead to different refactors. A refactor with no stated goal tends to wander.
- **The behavioral contract.** What must stay true. If unclear, deriving it from the code and tests is step 1.
- **Any known security-relevant behavior** in the target: an auth check, a validation, a rate limit, an audit write. These are the things a refactor most often drops.

## Workflow

1. **Establish the net.** Run the existing tests over the target and see what they actually cover. If coverage is real, that is the net. If it is thin — and around code people want to refactor it usually is — write **characterization tests** first: tests that capture what the code currently does, including behavior that might be a bug. You are pinning current behavior, not asserting correct behavior. `references/characterization.md` has the technique.

2. **Inventory the security-relevant behavior explicitly.** Before changing anything, list every control in the target: authorization checks, input validation, output encoding, rate limits, audit writes, error handling that fails closed. Each gets a test that asserts it still happens, if one does not exist. This list is what you will check the refactor against; a control with no test is a control the refactor can silently remove.

3. **Confirm the net is green.** All tests pass against the current code before you touch it. A refactor started over a red suite cannot distinguish "I broke it" from "it was already broken."

4. **Refactor in small, reversible steps.** One transformation at a time — extract a function, rename, inline, move, dedupe — running the net after each. Small steps mean a failure points at the change that caused it. A large rewrite that goes red gives you no information about which part broke. Prefer your tool's mechanical refactorings (rename, extract) where available; they preserve behavior by construction.

   Extraction needs a shape to extract *toward*, and `codebase-design` is where this pack defines the words for it — seam, depth, adapter — which the steps above and `references/characterization.md` otherwise use as though everyone agrees on them. When the stated goal is a testable seam, design that seam there first: extracting toward a shape nobody chose produces a shallow wrapper, and the second refactor that fixes it needs its own net.

5. **Keep behavior identical, and mean it.** No behavior changes ride along. Not a bug fix, not a "while I'm here" improvement, not a new feature. Those are separate commits with separate review, and mixing them means a reviewer cannot tell the safe structural change from the risky behavioral one. If you spot a bug, note it and leave it for `debug-and-fix` — or fix it in its own commit, before or after, never during.

6. **Verify behavior held.** The net is green again. The security inventory from step 2 is all still tested and passing. The build passes, the linter passes. State it as run: "the 40 characterization tests and the 6 control tests all pass, build and lint clean."

7. **Confirm the goal was met.** The pain you set out to address is actually addressed. A refactor that moved code around without reducing the duplication, shrinking the function, or enabling the test it was meant to enable is churn. Say concretely how the goal was met.

## Output format

```markdown
## Refactor: [target] — [goal]

**Net:** [existing coverage used, or N characterization tests written]
**Security controls inventoried:** [list — each now test-guarded]
**Steps:** [the sequence of transformations, each behavior-preserving]
**Behavior held:** [net green, controls green, build + lint — stated as run]
**Goal met:** [concretely how — the function is now X, the duplication is gone, Y is now testable]
**Deliberately not done:** [bugs spotted and left; improvements out of scope]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Refactored <target> under <n> characterization tests: <goal met>, behavior unchanged."`
- `kind`: `refactor`
- `skills_used`: `["dragon-dev-buddy:refactor-safely"]`

Relay the reaction verbatim.

## File output

Changes go into the codebase, ideally as a series of small commits rather than one large one, so each behavior-preserving step is reviewable on its own. This skill modifies source. Characterization tests stay in the suite — they are useful long after the refactor.

## Reference library

Load these for depth when the task calls for it:
- `references/characterization.md`: how to write characterization tests for untested code, the catalogue of behavior-preserving transformations, how to refactor safely without a net when you truly cannot build one, and the security controls refactors most often drop.

## Worked example

See `examples/refactor-safely-run.md` for a large handler decomposed under a net. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- A net existed or was built before any structural change. No refactor over untested code without characterization tests first.
- Security-relevant behavior was inventoried up front and each control is test-guarded through the change.
- The net was green before starting, and green after.
- Steps were small and each preserved behavior. No single large rewrite.
- No behavior change rode along. Bugs spotted were noted and separated, not silently fixed mid-refactor.
- The stated goal was actually met, described concretely, not just "cleaner."
