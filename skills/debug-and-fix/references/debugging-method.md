# Debugging method

For when the cause is not obvious. When it is, reproduce, fix, test, done — do not perform the ritual for a typo.

## The method

1. **State the expected and the actual.** Precisely. "Returns 3 items, should return 4" is workable. "Doesn't work" is not. The gap between them is the entire investigation.
2. **Reproduce on demand.** A trigger you can run repeatedly. Until you have this, you are guessing.
3. **Localize by halving.** Every step, cut the search space in half. Do not read the whole codebase; find the midpoint of the suspect path, determine which half the bug is in, repeat.
4. **Form one hypothesis, and try to disprove it.** Not confirm — disprove. Confirmation bias is the debugger's main enemy; you will find evidence for any theory you love. Look for the evidence against it.
5. **Fix the cause, verify against the reproduction, check the class.**

## Localization techniques

**Bisect the input.** Large input that triggers the bug: cut it in half. Which half still triggers it? Repeat. A 10,000-line input reduces to the one line that matters in about 14 steps. This alone solves a large fraction of parser and data-processing bugs.

**Bisect the history.** Regression that used to work: `git bisect` between the last-good and first-bad commit. `git bisect run <test>` automates it if you have a reproduction script. This turns "somewhere in 200 commits" into one commit, mechanically.

**Bisect the code path.** Add a log line or a breakpoint at the midpoint of the suspect flow. Is the state correct there? Yes → the bug is downstream. No → upstream. Halve again. Faster than reading, because the program tells you the truth and your reading of it might not.

**Rubber-duck the invariant.** State, out loud or in writing, what must be true at each step. The bug is where a thing you asserted turns out to be false. Often you find it mid-sentence, saying "and here X is definitely non-null, because—" and stopping.

## Intermittent and race-condition bugs

These are the ones people give up on and "fix" with a retry. Do better.

- **Force the race.** Add a deliberate delay (`sleep`, a debugger pause, a fault injector) at the point you suspect two things interleave. If a well-placed delay makes it reliable, you have found the window.
- **Increase the pressure.** Run the trigger in a tight loop, with concurrency, thousands of times. A one-in-ten-thousand bug surfaces in seconds under a loop, and now you have a reproduction.
- **Remove the nondeterminism.** Pin the seed, fix the clock, serialize the concurrency. If it stops happening, one of those was the cause — reintroduce them one at a time to find which.
- **Log with ordering.** Timestamps to the microsecond, thread or request ids, on both sides of the suspected interleaving. The log ordering shows you the bad interleaving directly.
- **Suspect shared mutable state first.** Most intermittent bugs are two things touching the same memory, row, or file without coordination. Find the shared thing; the bug is usually there.

A retry that hides an intermittent bug leaves the underlying race live — and a race in an auth check or a balance update is a security bug, not a flake.

## Common causes by symptom

| Symptom | Look first at |
| --- | --- |
| `null`/`undefined` dereference | An optional field assumed present; an empty collection; an async result used before it resolved |
| Off-by-one | A `<=` that should be `<`; an inclusive/exclusive boundary mismatch; a length used as an index |
| Works locally, fails in prod | Environment variable, timezone, locale, case-sensitive filesystem, a dependency version, real data volume |
| Intermittent | A race on shared state; test-order dependence; an unpinned clock or seed; a real network |
| Wrong result, no error | A logic inversion; a filter that excludes instead of includes; wrong operator precedence; integer division |
| Slow, then fails | An unbounded query or loop growing with data; a missing index; N+1; a leak |
| Fails only for some users | Data-dependent: a null in their record, a special character in their input, a role they have, their locale |
| Fails after a deploy | `git bisect` the range; the bug is almost certainly in the diff |

## When you cannot reproduce it

Do not fix by guess. A fix you cannot verify is a change you cannot trust, and it may add a second bug next to the first.

Instead, gather what would let you reproduce it:
- Better logging at the failure point, shipped, then wait for the next occurrence with real data attached.
- The exact input, environment, and user state from someone who saw it.
- Whether it correlates with a deploy, a time, a data condition, a specific account.

Say plainly: "I can't reproduce this yet, so I can't fix it with confidence. Here's the logging that will catch it next time, and here's what I'd need from the next report." That is a more honest and more useful answer than a speculative patch.
