---
name: codebase-design
description: The shared vocabulary for designing deep modules — a lot of behavior behind a small interface, at a clean seam. Use when someone says "design this module", "what should the interface be", "where does the seam go", "this is just a shallow wrapper", "make this testable", "we check tenancy in forty places", "deepen this", or when another skill in this pack uses the words seam, depth or adapter and someone needs them defined.
---

# Codebase Design

A **deep module** is a lot of behavior behind a small interface, placed at a clean seam, tested through that interface. The point of naming it is that most design arguments are really vocabulary arguments: two people say "component" and "boundary" at each other for an hour and never discover they disagree about where the interface lives. This skill fixes the words so the argument can be about the design.

This is a reference to consult, not a session to run. Its Workflow is the procedure for designing or deepening **one** module; more often it is loaded partway through someone else's work than invoked on its own. `refactor-safely` extracts and moves code and needs a target shape to extract toward. `security-test-writer` step 2 says "find the real seam" and does not define seam — this is the definition it points at. `secure-feature-build` says to prefer a design that makes the abuse case structurally impossible over one that checks for it; this skill is where that structure goes.

**The security argument.** A security control behind a shallow interface is a control every caller has to remember. Callers forget. "Every query must filter by tenant" is not a control, it is a convention with forty enforcement points and no enforcement, and the forty-first query is written by someone who read the other forty and copied the one that was already wrong. Depth is what converts it: put the tenant predicate behind an interface that cannot express an unscoped query, and omission stops being possible rather than merely discouraged. That is the same claim `secure-feature-build` makes about structural impossibility, stated in terms of where to put the structure. Depth is how a control becomes structural.

### Glossary

Use these terms exactly. Consistency is the whole value; a synonym is a new concept until proven otherwise.

**Module** — anything with an interface and an implementation. Deliberately scale-agnostic: a function, a class, a package, a slice spanning tiers. *Avoid*: unit, component, service — each drags in a size or a deployment assumption that modules do not have.

**Interface** — everything a caller must know to use the module correctly. The type signature, and also the invariants, the ordering constraints, the error modes, the required configuration, the performance characteristics, and the security properties the caller is entitled to assume. *Avoid*: API, signature — both name only the type-level surface, which is the smaller half.

**Implementation** — what is inside a module. Distinct from adapter: a thing can be a small adapter with a large implementation (a Postgres repository) or a large adapter with a small implementation (an in-memory fake). Say "adapter" when the seam is the subject, "implementation" otherwise.

**Depth** — leverage at the interface: how much behavior a caller or a test can exercise per unit of interface it has to learn. **Deep** is a large amount of behavior behind a small interface. **Shallow** is an interface nearly as complicated as the implementation behind it.

**Seam** *(Michael Feathers)* — a place where behavior can be altered without editing in that place; the *location* at which a module's interface lives. Where the seam goes is its own decision, separate from what goes behind it. *Avoid*: boundary — it collides with DDD's bounded context and with trust boundary, and those are three different things.

**Adapter** — a concrete thing satisfying an interface at a seam. Names a role (which slot it fills), not a substance (what is inside it).

**Leverage** — what callers get from depth: more capability per unit of interface learned. One implementation pays back across N call sites and M tests.

**Locality** — what maintainers get from depth: change, bugs, knowledge and verification concentrate in one place instead of spreading across callers. Fix once, fixed everywhere — which is the same sentence as "patch once, patched everywhere".

### Deep and shallow

Deep: a small interface over a large implementation. A caller learns three functions and gets pagination, retries, tenant scoping and audit writes.

Shallow: a large interface over a thin implementation. The worst case is the pass-through — a wrapper whose parameters mirror the thing it wraps, so the caller pays to learn two interfaces and gets the behavior of one.

Three questions when an interface is on the table: can it have fewer entry points, can the parameters be simpler, can more complexity move inside.

**The deletion test.** Imagine the module deleted and its callers rewritten. If complexity vanishes, it was a pass-through and deserved to go. If the same complexity reappears at every caller — the retry loop, the tenant predicate, the encoding step — it was earning its keep, and the count of places it reappears is the measure of what it earns.

**The interface is the test surface.** Callers and tests cross the same seam. Wanting to test *past* the interface means the module is the wrong shape, or the property under test belongs to a different module.

**One adapter means a hypothetical seam. Two adapters means a real one.** A port with a single implementation is indirection wearing a design pattern. Production plus a test adapter counts as two.

**Depth is a property of the interface, not the implementation.** A deep module can be internally composed of small, swappable parts — they are simply not part of its interface. Internal seams are legitimate and private; exposing one through the interface because a test wanted it is how a deep module goes shallow.

### Seam and trust boundary

A **seam** is where an interface lives. A **trust boundary** is where the level of trust in data changes — the edge that `threat-model` draws its data flows across. They are not the same thing and confusing them is expensive in both directions.

They often coincide, and should: the request parser, the tenant-scoped repository, the template renderer are all places where a seam is worth putting precisely because trust changes there, and an interface at that spot can make the trust change mandatory rather than customary.

They also come apart. A seam between two pure in-process modules crosses no trust boundary at all and needs no validation. A trust boundary with no seam at it is the dangerous case: untrusted data changing status by convention, at forty call sites, with nothing in the type system or the module structure recording that it happened. Put a seam there. That is the shape of the fix.

### Rejected framings

**Depth as the ratio of implementation lines to interface lines** (Ousterhout's formulation): rewards padding the implementation, and scores a bloated module as a good one. Depth here is leverage — behavior reachable per unit of interface learned.

**"Interface" as the language keyword, or as a class's public methods**: too narrow. Every fact a caller must know is part of the interface, including the ones no compiler checks.

**"Boundary"**: overloaded three ways. Say **seam** for where the interface lives and **trust boundary** for where trust changes.

## First-run check

Read `.dragon-buddy/config.json`. This skill needs very little from it, and the vocabulary is language-independent, so a missing config is not fatal — say so once and offer `buddy-setup` rather than stopping.

Pull `project.primary_language` (it decides what a port is called in this codebase — an interface, a trait, a protocol, a callable), `security.auth_model` and `security.trust_boundaries` (they name the controls most likely to be scattered across call sites, which are the deepening candidates that matter), and `practice.test_command` (a design whose tests you cannot run is a proposal, not a design).

## Inputs

Ask only for what cannot be read from the code:

- **The module or cluster under design**, and what is wrong with it now — shallow wrapper, too many entry points, untestable, or a control repeated everywhere. Each leads somewhere different.
- **What actually varies across the candidate seam.** This is the question that decides whether a port is a design or an affectation. If nothing varies, the answer is one module and no seam.
- **Which callers are in scope to change.** A seam you cannot move callers onto is a second way to do the thing, and the old way is the one that stays wrong.

Do not ask for the call sites, the dependency list or the current signatures. Grep for them.

## Workflow

1. **Write the current interface out in full.** Not the signatures — the whole thing: parameters, invariants, ordering constraints ("call `init` first"), error modes, required configuration, and what the caller must do that the compiler does not enforce. Shallow modules look small until this list exists. A module whose real interface is a paragraph of prose the caller has to remember has an interface the size of that paragraph.

2. **Apply the deletion test.** Delete the module in your head and rewrite the callers. Say what happens: complexity vanishes (it was a pass-through — the answer may be to delete it) or complexity reappears at N callers (name N, and name what reappears). N is the leverage number and everything downstream is argued against it.

3. **Find the scatter, and grep for it.** For each thing that reappears, count the call sites for real. Security controls first: the tenant predicate, the authorization check, the escaping call, the audit write, the rate-limit decrement. Every site is a place the control can be omitted, and the count is the argument for depth. Report the sites you found *and* the sites you found that were already missing it — a scatter audit almost always finds one, and that one is a finding, not a design detail. Hand it to `secure-code-review` or `vuln-triage` rather than quietly fixing it inside a design change.

4. **Classify the dependencies and place the seam.** In-process, local-substitutable, remote-but-owned, or true external — the category decides how the deepened module is tested and whether a port is needed at all. `references/deepening.md` has the categories and what each implies. Placing the seam is a separate decision from what goes behind it; make it explicitly and say why there and not one layer up.

   Check the seam against the trust boundaries from `threat-model` or from `security.trust_boundaries`. A trust boundary with no seam at it is the highest-value place to put one.

5. **Design the interface more than once.** Your first interface is unlikely to be your best, and the cost of finding that out later is every caller. Produce three or more radically different designs — minimal surface, maximal flexibility, optimized for the common caller, ports-and-adapters — in parallel subagents where the module is big enough to justify it. Compare them on depth, locality and seam placement, then recommend one in your own voice. `references/deepening.md` has the briefing pattern and what each design must return. A menu with no recommendation is not an answer.

6. **Count the adapters.** If the winning design puts a port at the seam, name both adapters. One adapter is a hypothetical seam: collapse it and inline the implementation. Production plus test counts, but only if the test adapter is real and used — a fake nobody instantiated is a fourth thing to maintain.

7. **Make the wrong path unreachable, not merely discouraged.** This is the step that turns a deepening into a control, and the step most often skipped. If the raw query builder, the unescaped renderer or the unscoped repository is still importable, the control is still optional and step 3's count will grow again. Close it: make the old constructor private, put the escape hatch behind a name that says what it costs, add the lint rule or the visibility restriction, delete the old path in the same change. State plainly which mechanism is holding it shut, and whether that mechanism is checked by the compiler, by CI, or by nobody.

8. **Move the tests to the interface, and replace rather than layer.** New tests exercise the deepened module through its interface. Old tests against the shallow parts are waste once those exist — delete them, do not keep both. Tests assert observable outcomes across the seam, never internal state, so they survive the next internal change. A test that must be edited when the implementation changes was testing past the interface.

9. **Hand the code motion to `refactor-safely`.** This skill decides the shape. Moving live code into that shape without changing behavior is a different job with a different discipline, done under a net, and it is that skill's. Say so at the handoff instead of starting to move code.

### Designing for testability

Three rules, applied while designing rather than discovered while testing:

**Accept dependencies, do not construct them.** A module that news up its own gateway, clock or database has hidden the seam inside its implementation where no test can reach it. Take it as a parameter.

**Return results, do not mutate at a distance.** A function that returns a computed value is tested by calling it. A function that reaches into a shared structure and changes a field is tested by reconstructing the world around it first.

**Keep the surface small.** Fewer entry points, fewer tests to write. Fewer parameters, less setup per test. The interface being cheap to learn and cheap to test is the same property measured twice.

## Output format

```markdown
## Design: [module] — [deep | shallow now, deep after]

**Current interface:** [everything a caller must know today, including the unenforced parts]
**Deletion test:** [what vanishes / what reappears at N callers — N stated]
**Scatter found:** [control, call-site count, and any site already missing it]
**Dependencies:** [each one, with its category]
**Seam:** [where it goes, why there; whether it coincides with a trust boundary]
**Alternatives considered:** [the designs, one line each, compared on depth / locality / seam]
**Recommended interface:** [the entry points, plus invariants, ordering, error modes]
**Adapters:** [two named, or the seam was collapsed and why]
**Wrong path closed by:** [mechanism, and what checks it]
**Test surface:** [tests at the interface; old tests deleted, named]
**Handoff:** [what `refactor-safely` has to move]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Designed <module> as a deep module behind <n> entry points; <control> now enforced at the seam instead of at <m> call sites."`
- `kind`: `refactor`
- `skills_used`: `["dragon-dev-buddy:codebase-design"]`

Relay the reaction verbatim.

## File output

A design note, and nothing else by default. Where `output.reports_dir` is set, write it there; otherwise put it in the conversation and let the user place it. If the design is worth keeping — particularly the rejected alternatives and why — `project-memory` is where it belongs, because the reasoning behind a seam is exactly the thing that is not derivable from the code.

This skill does not modify source. When it is loaded inside another skill's run, that skill's file-output rules apply and the code motion belongs to `refactor-safely`.

## Reference library

- `references/deepening.md` — the four dependency categories and what each implies for the seam and its adapters, seam discipline (internal versus external, the two-adapter rule), replace-don't-layer testing, and the design-it-twice procedure: framing the problem space, briefing parallel subagents with different constraints, and comparing the results on depth, locality and seam placement. Load it at Workflow steps 4 and 5.

## Worked example

See `examples/codebase-design-run.md` for a shallow repository whose tenant predicate was copied across nineteen call sites, one of which had lost it, deepened behind a three-function interface with the old path closed and the scatter audit reported as a finding. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

A good run satisfies all of these:

- The current interface was written out in full, including the constraints no compiler checks, before any new design was proposed.
- The deletion test was applied and its answer was a number of callers, not an adjective.
- The scatter was counted by grep, not estimated, and any call site already missing the control was reported as a finding and routed, not silently fixed.
- Every dependency was placed in a category, and the seam placement was justified against that categorization rather than assumed.
- Where the seam was proposed as a port, two adapters were named. Single-adapter seams were collapsed.
- More than one interface was designed, and the alternatives were compared on depth, locality and seam placement before one was recommended.
- The recommendation was stated as a recommendation, not offered as a menu.
- The old, shallow path was closed by a named mechanism, and the run said whether the compiler, CI or nobody enforces it.
- Tests were moved to the new interface and the superseded tests were named for deletion, not left alongside.
- The code motion was handed to `refactor-safely` rather than performed here.
