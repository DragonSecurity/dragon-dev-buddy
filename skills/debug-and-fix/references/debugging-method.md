# Debugging method

For when the cause is not obvious. When it is, reproduce, fix, test, done — do not perform the ritual for a typo.

## Build the feedback loop first

A feedback loop is one command that goes red on this bug and green when it is gone. It is not a step in the method, it is the thing the method runs on: bisection consumes it, hypothesis testing consumes it, the fix is verified by it. With a loop, the bug is most of the way to fixed. Without one, every technique below degrades into reading code and believing yourself.

So spend disproportionate effort here, and be inventive about it. The loop does not have to be pretty and it does not have to survive the session.

### Ways to build one, roughly best first

1. **A failing test** at whatever layer reaches the bug — unit, integration, end-to-end. Best because it is the artifact you were going to need anyway.
2. **A curl or HTTP script** against a dev server, asserting on status and body.
3. **A CLI invocation with a fixture**, diffing stdout against a known-good snapshot.
4. **A headless browser script** (Playwright, Puppeteer) that drives the UI and asserts on the DOM, the console, or the network.
5. **A replayed capture.** Save the real request, payload, or event log to disk and push it through the code path in isolation. Redact credentials on the way in — a saved HAR is full of them.
6. **A throwaway harness.** The smallest slice of the system that reaches the bug from one function call, with everything else stubbed.
7. **A property or fuzz loop.** For "sometimes the output is wrong": a thousand generated inputs and an assertion about what must always hold.
8. **A bisection harness.** If the bug appeared between two known states, automate "put the system in state X, check, report" so `git bisect run` can drive it.
9. **A differential loop.** Same input through two versions or two configs, diff the outputs. The diff is the bug.
10. **A human-in-the-loop script.** Last resort, when a person must click. Script the prompts and capture the output so the loop still has structure and a record.

### Tighten it

Treat the loop as something you are building, not something you found. Once it exists, make it better on three axes:

- **Faster.** Cache the setup, skip unrelated init, narrow the scope to the failing path.
- **Sharper.** Assert the specific symptom the user reported, not "it did not crash."
- **More deterministic.** Pin the clock, seed the RNG, isolate the filesystem, freeze the network.

A thirty-second flaky loop is barely better than no loop. A two-second deterministic one is a different tool entirely — it changes what you are willing to try, because trying is free.

### The gate

You are done building the loop when you can name **one command** that you have **already run at least once**, and that is:

- **Red-capable** — it drives the actual bug path and asserts the user's exact symptom, so it can go red now and green after the fix. "Runs without erroring" is not red-capable.
- **Deterministic** — same verdict every run. For intermittent bugs, a pinned and high reproduction rate counts.
- **Fast** — seconds, not minutes.
- **Agent-runnable** — it runs unattended, or with the human step scripted.

Until that command exists, do not read code to build a theory. Jumping to a plausible hypothesis before there is anything that can contradict it is the specific failure this discipline exists to prevent, and it is the one that costs whole afternoons.

## The method

1. **State the expected and the actual.** Precisely. "Returns 3 items, should return 4" is workable. "Doesn't work" is not. The gap between them is the entire investigation.
2. **Reproduce on demand** — the loop above. Until you have it, you are guessing.
3. **Localize by halving.** Every step, cut the search space in half. Do not read the whole codebase; find the midpoint of the suspect path, determine which half the bug is in, repeat.
4. **Form one hypothesis, and try to disprove it.** Not confirm — disprove. Confirmation bias is the debugger's main enemy; you will find evidence for any theory you love. Look for the evidence against it.
5. **Fix the cause, verify against the reproduction, check the class.**

## Localization techniques

**Bisect the input.** Large input that triggers the bug: cut it in half. Which half still triggers it? Repeat. A 10,000-line input reduces to the one line that matters in about 14 steps. This alone solves a large fraction of parser and data-processing bugs.

**Bisect the history.** Regression that used to work: `git bisect` between the last-good and first-bad commit. `git bisect run <test>` automates it if you have a reproduction script. This turns "somewhere in 200 commits" into one commit, mechanically.

**Bisect the code path.** Add a log line or a breakpoint at the midpoint of the suspect flow. Is the state correct there? Yes → the bug is downstream. No → upstream. Halve again. Faster than reading, because the program tells you the truth and your reading of it might not.

**Rubber-duck the invariant.** State, out loud or in writing, what must be true at each step. The bug is where a thing you asserted turns out to be false. Often you find it mid-sentence, saying "and here X is definitely non-null, because—" and stopping.

## Instrumentation hygiene

Every probe should be answering a specific prediction. Instrumentation added without a question attached produces output nobody reads.

- **One variable at a time.** Change two things and a green run tells you nothing about which one mattered.
- **A debugger beats ten log lines.** If the environment gives you a breakpoint or a REPL, one inspection of live state answers what a sprinkling of prints only gestures at. Fall back to targeted logs at the boundaries that separate your hypotheses.
- **Never log everything and grep.** That is a search for a needle you have not described yet, and it buries the signal you already had.
- **Tag every debug log with a unique prefix**, such as `[DEBUG-a4f2]`. Cleanup then becomes one grep, and the grep is provably complete. Untagged debug logs survive the session, reach production, and eventually print something they should not.

## Performance regressions

Logs are usually the wrong tool here — they tell you the order of events and not where the time went. Establish a baseline measurement first: a timing harness, a profile, a query plan, a request trace with real numbers on it. Then bisect against that measurement the same way you would bisect a wrong result, with "slower than baseline by X" as the red condition.

Measure first, fix second. A performance fix applied without a before-number cannot be shown to have worked, and the change most people are sure about is usually not the one that mattered.

## Intermittent and race-condition bugs

These are the ones people give up on and "fix" with a retry. Do better.

The goal here is not a clean reproduction — you may never get one — but a **higher reproduction rate**. A bug that fires half the time is debuggable; one that fires one time in a hundred is not. Raising the rate is the work.

- **Force the race.** Add a deliberate delay (`sleep`, a debugger pause, a fault injector) at the point you suspect two things interleave. If a well-placed delay makes it reliable, you have found the window.
- **Increase the pressure.** Run the trigger in a tight loop, in parallel, thousands of times, under stress. A one-in-ten-thousand bug surfaces in seconds under a loop, and now you have a reproduction.
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

## When you cannot build a loop

Do not fix by guess. A fix you cannot verify is a change you cannot trust, and it may add a second bug next to the first.

Stop and say so plainly, and list what you actually tried — the ranked list above is the checklist, so name which of the ten you attempted and why each failed. Then ask for the one thing that would unblock it:

- **Access to an environment where it reproduces**, even read-only, even for one session.
- **A redacted captured artifact** — a HAR file, a log dump with the surrounding minute, a core dump, a screen recording with timestamps. Redact credentials and personal data before it moves; a capture from production is production data.
- **Permission to instrument production temporarily**, with the tagged-prefix discipline above and a stated removal plan.

Also worth asking for: the exact input, environment and user state from someone who saw it, and whether it correlates with a deploy, a time of day, a data condition or a specific account.

"I can't reproduce this yet, so I can't fix it with confidence. Here is what I tried, here is the logging that will catch it next time, and here is what I need from the next report" is a more honest and more useful answer than a speculative patch.

## The regression test needs a correct seam

Write the regression test before the fix — but only where there is a seam that exercises the **real bug pattern** as it occurs at the call site. A test at a seam too shallow to reach the pattern is worse than no test: it passes, it looks like coverage, and it will keep passing while the bug walks back in. A single-caller unit test does not lock down a bug that needs two callers interleaving; a test below the routing layer does not lock down a routing bug.

If no correct seam exists, **that absence is itself a finding**. Say it. The architecture is what is preventing the bug from being locked down, and the next occurrence of this bug is already paid for. Note it against the fix and raise it after the fix lands, when you know the most about the shape of the problem.

For the vocabulary — seam, module, interface, depth — and for what to do about a codebase with no seam where you need one, load `codebase-design`.
