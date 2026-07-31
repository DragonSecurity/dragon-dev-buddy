---
name: debug-and-fix
description: Reproduces a bug, isolates the root cause, fixes it, and proves the fix with a test. Use when someone says "there's a bug", "this is broken", "it crashes when", "why does this happen", "help me debug", "this returns the wrong result", "intermittent failure", or pastes an error or stack trace. Fixes the cause, not the symptom, and leaves a test so it stays fixed. Treats correctness as a security property, because it is.
---

# Debug and Fix

A bug is a gap between what the code does and what you believed it did, and that gap is exactly where security defects live too. This skill closes the loop properly: reproduce it, find the actual cause, fix that cause, and leave a test that fails if it ever comes back. The failure mode it exists to prevent is the fix that makes the symptom disappear without addressing why it happened, which is how the same bug returns wearing a different hat.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.stack`, `project.primary_language`, `practice.test_command`, `practice.build_command`, `output.reports_dir`, `skill_level`.

## Inputs

Ask only for what is missing:
- **The symptom.** The error, the stack trace, the wrong output, or the description of what happens versus what should.
- **How to trigger it.** The steps, the input, the request. Reliable or intermittent.
- **What changed recently**, if anything. A bug that appeared after a deploy has a much smaller search space.
- **Where it happens.** Environment, which users, how often.

## Workflow

1. **Reproduce it first.** Before theorizing, make it happen. A failing test, a curl command, a script — something that produces the bug on demand. A bug you cannot reproduce, you cannot know you fixed. If it is intermittent, work `references/debugging-method.md` for the techniques that force a race or a state-dependent bug into the open. If you genuinely cannot reproduce it, say so and switch to gathering the evidence that would let you, rather than guessing at a fix.

2. **Capture the reproduction as a test.** Write the failing test now, while the bug is live. It documents the exact trigger, it will verify the fix, and it becomes the regression guard. This is the same test-first discipline as `security-test-writer`; here it starts the work rather than ending it.

3. **Isolate by bisection, not by staring.** Narrow the location with evidence: binary-search the input to find the smallest that triggers it, bisect the git history if it is a regression, add instrumentation at the midpoint of the suspect path and halve the search each time. Resist the first plausible theory; the plausible cause and the actual cause diverge often enough that jumping to a fix wastes more time than isolating properly.

4. **Find the root cause, and say it in one sentence.** Not "there was an issue in the handler" but "the cursor is advanced before the null check, so an empty page dereferences past the end." If you cannot state the cause in one sentence, you have not found it yet — you have found a place where the symptom appears. Keep going.

5. **Check whether the cause is a class.** A root cause is rarely unique. An unhandled null here is usually an unhandled null in the three sibling functions written the same day. Grep for the pattern. Fixing one instance of a class and shipping is how the bug reappears next to itself.

6. **Fix the cause.** The minimal change that addresses the root cause, in the codebase's existing style. Not a broader refactor riding along — that is `refactor-safely`'s job and mixing them makes both harder to review. If the clean fix requires a refactor, say so and scope it separately.

7. **Prove it.** The test from step 2 now passes. The build passes, the rest of the suite passes. State that you ran them and what came back — not "this should fix it" but "the failing test passes and the other 84 still do." If the bug was a class, the fix covers the class and there is a test per instance or one that covers them together.

8. **Note the security dimension, if any.** A null dereference on user input is a denial of service. An off-by-one in an authorization loop is an access control bug. A race in a balance check is a financial one. If the bug you just fixed is reachable by an attacker, say so and consider handing it to `vuln-triage` for a severity and `security-test-writer` for the adversarial cases.

## Output format

```markdown
## Bug: [one-line symptom]

**Reproduced:** [the exact trigger — test, command, or input]
**Root cause:** [one sentence, specific, at file:line]
**Class check:** [other instances found, or "isolated to this call site"]
**Fix:** [what changed and why it addresses the cause, not the symptom]
**Proof:** [test now passes; suite/build status, stated as run]
**Security dimension:** [if attacker-reachable — what it is; else "none"]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Fixed <symptom>: root cause was <one clause>, regression test added."`
- `kind`: `bugfix`
- `skills_used`: `["dragon-dev-buddy:debug-and-fix"]`

Relay the reaction verbatim.

## File output

The fix and its test go into the codebase. For a subtle or recurring bug, a short note in `output.reports_dir` is worth writing; for a routine one, the test and the commit message are the record. This skill modifies source.

## Reference library

Load these for depth when the task calls for it:
- `references/debugging-method.md`: the systematic method for when the cause is unclear, techniques for intermittent and race-condition bugs, bisection strategies, and the common-cause patterns by error type.

## Worked example

See `examples/debug-and-fix-run.md` for a bug fixed from stack trace to regression test. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The bug was reproduced before it was fixed. A fix for an unreproduced bug is labelled a hypothesis.
- The root cause is stated in one specific sentence, at a location. No "there was an issue with."
- The fix addresses the cause, not the symptom, and does not smuggle in an unrelated refactor.
- The class was checked. Sibling instances were found and fixed, or their absence was confirmed.
- The proof states the suite and build were actually run and what came back, not that they "should" pass.
- An attacker-reachable bug is flagged as such rather than fixed as a plain correctness issue.
