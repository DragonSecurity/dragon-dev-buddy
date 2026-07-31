# Characterization tests and safe transformations

## What a characterization test is

A test that documents what the code *currently does*, not what it *should* do. You are pinning behavior so a refactor cannot change it unnoticed. If the current behavior is a bug, the characterization test asserts the bug — deliberately. You fix the bug separately, afterward, and update the test in that commit, where the change is visible and reviewed.

This feels wrong the first time. It is correct. The refactor's job is to preserve behavior; correctness is a different job with different risk.

## Writing them for untested code

1. **Call the code and see what comes out.** Feed it representative input and capture the actual output — the return value, the database writes, the calls it makes, the errors it throws.
2. **Assert exactly that.** Even if it looks wrong. `expect(result).toBe(-1)` when you suspect it should be 0. You are recording reality.
3. **Cover the branches.** Walk the code and ensure an input reaches each path. The net has holes exactly where no test exercises a branch, and the refactor will fall through those holes.
4. **Include the ugly inputs.** Null, empty, the boundary, the malformed. These are where refactors change behavior, because the original handling was often accidental and easy to "tidy" away.
5. **Pin the side effects, not just the return.** If the function writes a row, sends an email, or logs an audit record, assert those happen. A refactor that drops a side effect while returning the same value is exactly the dangerous case.

**Golden-master variant.** For code with large output (a rendered document, a serialized structure, a report), capture the entire output as a stored snapshot, refactor, and diff against the snapshot. Any behavior change shows up as a diff. Good for code too tangled to unit-test cleanly.

## The transformation catalogue

Each of these preserves behavior when done in isolation. Run the net after each.

| Transformation | What it does | Watch for |
| --- | --- | --- |
| Extract function | Pull a block into a named function | Closed-over variables; early returns changing meaning |
| Inline | The reverse | Side effects that ran once now running per call site |
| Rename | Better name, same thing | Reflection, string references, serialized names |
| Move | Relocate to a better home | Import cycles; visibility changes |
| Extract variable | Name a subexpression | Evaluation order; a side-effecting expression now bound once |
| Replace conditional with polymorphism | Types instead of branches | The default/fallthrough case |
| Introduce parameter object | Group related args | Argument order at every call site |
| Dedupe | Three copies become one | The copies were subtly *different*, and you just erased the difference |

That last one is the most dangerous and the most common goal. "These three blocks are identical" is a claim to verify character by character before merging them, because if block two had an extra check that blocks one and three lacked, deduplicating to block one's version silently removes it. Diff them explicitly first.

## Security controls refactors most often drop

Inventory these in the target before you start. Each is something that has looked like removable clutter to someone.

- **An authorization check** that sits at the top of a handler. Extracting the handler body can leave the check behind, or move it below the sensitive operation.
- **Input validation** that a refactor "consolidates" and thereby skips for one path.
- **Output encoding / escaping** lost when a template or serializer is swapped.
- **A rate limit or lock** dropped when the surrounding function is restructured.
- **An audit write** removed because it looked unrelated to the function's "real" job.
- **Fail-closed error handling** turned into fail-open when a try/catch is refactored — a `catch` that returned `403` becoming one that returns the data.
- **A tenant or ownership predicate** on a query, lost when the query is moved into a shared helper that does not know about tenancy.

For each present in the target, there must be a test asserting it still happens after the refactor. If writing that test is hard because the control is tangled into everything, that difficulty *is* the reason to refactor — but write the test first anyway, even a crude one.

## Refactoring without a net

Sometimes you genuinely cannot build one: no test harness, external dependencies that cannot be faked, time pressure. Reduce the risk instead of pretending it is absent:

- Use only the mechanical refactorings your IDE or language tooling guarantees (rename, extract, move). These preserve behavior by construction, not by hope.
- Make the smallest possible change and stop. Do not compound unverified changes.
- Keep each change in its own commit so any single one can be reverted cleanly.
- Say explicitly, in the output and to the user, that this refactor was done without a behavioral net and what the residual risk is. Do not present unverified structural change as safe.

An honest "I changed this structurally without a test net, here's the residual risk" is worth far more than silent confidence.
