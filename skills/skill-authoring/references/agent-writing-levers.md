# The levers of writing for agents

Every lever below applies to any document an agent consumes — a skill, a `CLAUDE.md`, an `AGENTS.md`, a reference file something points at. Each one is stated, then worked on a real passage: a before, an after, and what the rewrite bought.

## Context pointers

A **context pointer** is a reference held in the agent's context that names some out-of-context material and encodes the condition for reaching it. A skill's `description` is one. A line in `CLAUDE.md` naming a doc is the same object. So is "load the characterization reference when coverage is thin" inside a skill body.

The pointer's *wording*, not its target, decides when the agent reaches the material and how reliably. This is the single most common failure in a skill pack: the material is correct, complete, and never loaded. A must-have target behind a weakly worded pointer is a variance bug — the same request routes differently on Tuesday. Sharpen the wording first; inline the material only if sharpening fails.

A pointer does two jobs: state what the material is, and list the **branches** that should trigger reaching it. A branch is a distinct case the document handles, so different runs take different paths through it. Every word of an always-loaded pointer is spent on every turn whether or not it fires, so it earns harder pruning than the body:

- **Front-load the leading word.** The pointer is where triggering happens, and the first clause carries most of it.
- **One trigger per branch.** Synonyms that rename a single branch are one branch written twice — they cost tokens and add no reach.
- **Cut identity the body already carries.** The description does not need to explain what the skill is called or that it is a skill.

### Worked rewrite: a description that never fires

**Before**

```
description: A comprehensive skill for reviewing code with security in mind. It
  helps developers find vulnerabilities and improve the quality and safety of
  their codebase using industry best practices and a thorough methodology.
```

Nothing in that string is a condition. It says what the skill *is* — three times — and never says when to reach it, so the model has to infer the trigger from a self-description. Runs where the user says "can you look at this handler" do not route.

**After**

```
description: Adversarial security review of a diff, file, module, one pull
  request or a batch of them. Produces findings with a concrete exploit path,
  evidence at a line number, and a fix, ranked by severity. Use when someone
  says "review this for security", "is this safe", "security review this PR",
  "check this before it ships", "look at this handler", "did I do this auth
  check right", or after any change to authentication, authorization, input
  handling, crypto or file handling.
```

What it bought: the branches are enumerated (a diff, a file, one PR, a batch), each trigger is a phrase a user really types, and the last clause adds a condition with no phrase at all — a change to an auth path fires the skill even when the user asked for something else. "Comprehensive", "best practices" and "thorough methodology" are gone; they never routed anything.

### Worked rewrite: a pointer inside a body

**Before**

> There is a reference file in the references directory that contains additional
> information about characterization testing which you may find useful.

**After**

> If coverage is thin — and around code people want to refactor it usually is —
> write **characterization tests** first. `references/characterization.md` has the technique.

What it bought: the condition is stated ("if coverage is thin"), and the leading word *characterization* is the hook the file is named for, so the pointer and the target share vocabulary. The before-version encodes no condition at all, which makes loading the file a coin flip.

## The two loads

Every document and every pointer spends one of two budgets. Knowing which one you are spending is most of the design decision:

- **Context load** — always-loaded material on the agent's window. A skill description, a `CLAUDE.md` line, anything present every turn. It costs tokens and attention whether or not it ever fires.
- **Cognitive load** — the cost on the human, who is the index: which documents exist and when to reach for each. Not a cost to minimize. It is the price of human agency, so spend it where human judgement matters and remove it where it does not.

Material reached only through a pointer escapes context load at the price of the pointer's own line. Material with no pointer at all rides entirely on cognitive load — someone has to remember it exists.

In this pack, every skill is model-invoked, so every description is permanent context load and the pack pays all of it on every turn of every session. That is the bill to hold in mind when someone proposes one more skill: the question is not whether the skill is good, it is whether its description earns a permanent seat.

## Information hierarchy

A document is built from two content types that mix freely — **steps** (ordered actions the agent performs) and **reference** (definitions, rules and facts consulted on demand). A skill's Workflow is steps; its Quality bar is reference; a review skill can be almost entirely reference and still be a good skill.

The core decision is where each piece sits on the ladder, ranked by how immediately the agent needs it:

1. **In-file step** — the primary tier. What the agent does, in order.
2. **In-file reference** — consulted on demand. Often a legitimately flat peer-set, like every item of a Quality bar on one rung. That flatness is a fine arrangement, not a smell.
3. **Disclosed reference** — pushed into a separate file behind a pointer, loaded only when the pointer fires. Ranges from a sibling in the same folder to fully external material any document can point at.

Push too little down and the top bloats. Push too much and you hide material the agent actually needs. That tension is the whole decision, and it has no general answer — only a test.

**Progressive disclosure** is the move down the ladder. It is not primarily a token optimization; it is how the top of the file stays legible. Branching is the cleanest test: inline what every branch needs, disclose what only some branches reach. When a document has steps, in-file reference that should have been disclosed buries them, and attending to the steps becomes a coin flip — a variance lever, not merely a tidiness one.

**Co-location** is the within-file companion. The ladder decides how far down a piece sits; co-location decides what sits beside it once there. Keep a concept's definition, rules and caveats under one heading, so reading one part brings its neighbours with it. The test: the document should read like documentation written for the agent. Scattering fragments one meaning across many places, which is a different disease from duplication repeating one meaning in two.

**Sprawl** is the failure mode at this level — a document simply too long, even when every line is live and unique. Attention thins across the excess and every extra line is one more to keep relevant. The cure is the ladder: disclose reference behind pointers, and split by branch or sequence so each path carries only what it needs.

### Worked rewrite: disclosing by branch

**Before** — a skill body carrying, inline, a 60-line table of every firmware advisory class, applicable only when the target is a network device.

**After** — the body keeps one line:

> When the target is a device rather than a repository, the advisory classes and
> what each implies are in the reference file this skill points at.

What it bought: every run that reviews application code stops paying 60 lines it will never use, and the device branch still gets them. The test that produced the decision was mechanical — *does every branch need this?* No. Disclose.

**When not to disclose:** the security-control inventory in `refactor-safely` is short, and every single run needs it. Pushing it behind a pointer would make the most important step of the skill conditional on a pointer firing. Inline it.

## Steps and completion criteria

Every step ends on a **completion criterion** — the condition that tells the agent the work is done. Two properties make it a lever.

**Clarity.** Can the agent tell done from not-done? A vague bound ("understanding reached", "the code is cleaner") invites **premature completion**: the step ends before it is genuinely finished, because attention has slipped toward *being done*. The visible steps still ahead — the **post-completion steps** — supply that pull; the criterion's clarity is the resistance.

Defend in order. Sharpen the bound first: it is local, cheap and usually sufficient. Only if the bound is irreducibly fuzzy *and* you observe the rush should you hide the later steps by splitting the sequence — and hiding only works across a real context boundary, a hand-off or a subagent dispatch. An inline call leaves the later steps sitting in context and clears nothing.

**Demand.** How much the criterion requires. "Every modified model accounted for" forces thorough work where "produce a change list" does not. Demand drives **legwork** — the digging the agent does inside the work, latent in the wording rather than written as its own step. It is not step-bound: "every rule applied" binds a body of flat reference exactly as "every step done" binds a sequence, which is how an all-reference document still carries an exhaustiveness bar.

The strongest criteria are both checkable and exhaustive.

### Worked rewrite: sharpening a bound

**Before**

> 3. Make sure the tests are in good shape before you start refactoring.

**After**

> 3. **Confirm the net is green.** All tests pass against the current code before
>    you touch it. A refactor started over a red suite cannot distinguish "I broke
>    it" from "it was already broken."

What it bought: "in good shape" has no observable state behind it, so the agent decides it is satisfied by looking at the test file. "All tests pass against the current code" is binary and requires running something. The second sentence supplies the reason, which is what stops the step being skipped when it is inconvenient.

### Worked rewrite: raising demand

**Before**

> List the security-relevant parts of the code you are changing.

**After**

> Inventory the security-relevant behavior explicitly. List every control in the
> target: authorization checks, input validation, output encoding, rate limits,
> audit writes, error handling that fails closed. Each gets a test that asserts
> it still happens, if one does not exist.

What it bought: "every control in the target" plus an enumeration of what counts is exhaustive and checkable; the reader can point at a control the list missed. And "each gets a test" converts the inventory from a note into legwork with an artifact at the end of it.

## When to split

Splitting one document into two spends one of the two loads, so split only when the cut earns it.

**By sequence** — split a run of steps where the post-completion steps tempt the agent to rush the one in front of it. Keeping the later steps out of view drives more legwork on the current task. Beware the reverse: merging two sequences exposes each step's successors to what follows, which invites premature completion where none existed before.

**By invocation** — the skill-specific cut, covered in the sibling reference this skill points at.

Neither cut is free. A second file is a second thing to keep relevant and a second pointer to word correctly, and in this pack a second skill is a permanent description in every session's context.

## Leading words

A **leading word** is a compact concept already living in the model's pretraining that the agent thinks with while running the document — *net*, *lesson*, *fog of war*, *tracer bullets*. Repeated as a token and never as a sentence, it accumulates a distributed definition across the document and anchors a whole region of behavior in the fewest possible tokens, because it recruits priors the model already holds.

Coining your own works if you define it clearly, but a made-up word recruits nothing: you pay in definition tokens what a pretrained word gives away. Reach for an existing word first.

It anchors twice:

- **In the body, execution.** The agent reaches for the same behavior every time the word appears, and inside flat reference the word focuses attention on a class of thing to look for.
- **In a pointer, invocation.** When the same word lives in your prompts, your docs and your codebase, the agent links that shared language to the material and reaches it more reliably.

Hunt for opportunities to refactor with leading words. A triad spelled out at three sites, a pointer spending a sentence to gesture at one idea — each is a passage begging to collapse into a single token.

### Worked rewrites: collapsing into a token

- "fast, deterministic, low-overhead" → ***tight*** (a *tight* loop). Three adjectives restated at every site become one word the model already understands.
- "a test suite you believe in, that is passing, that covers the behavior you are about to change" → ***the net*** — and now "the net is green" is a binary observable state, "build the net first" is a step, and "the net had holes" is a diagnosis. One token, three uses. This is why `refactor-safely` reads as tightly as it does.
- "before the change, check whether applying it could cut the path you are applying it over" → ***severance***. Named, it becomes something the agent looks for rather than a sentence it read once.

You win twice: fewer tokens, and a sharper hook for the agent to hang its thinking on. Assume every document you inherit is carrying restatements that leading words retire, and go find them.

### Negation, the failure mode beside it

Steering by prohibition drags the forbidden behavior into context and makes it *more* available, not less. Say *don't think of an elephant* and the elephant is all there is. The negation is a weak modifier that the strongly activated concept overruns, so the ban half-reads as an instruction to do the thing.

Prompt the **positive**: state the target behavior so the banned one is never spoken.

**Before**

> Don't write long rambling comments explaining every parameter, and don't add
> emoji, and don't leave TODO comments everywhere.

**After**

> Write one-line comments that say why, not what.

What it bought: the three banned behaviors are no longer in context at all, and the replacement is a target the agent can hit. A prohibition earns its place only as a hard guardrail you cannot phrase positively — "never commit a live credential" — and even then, pair it with the positive so attention lands on what to do.

## Pruning

**Single source of truth.** Keep each meaning in one authoritative place, so changing the behavior is a one-place edit. **Duplication** — the same meaning in more than one place — costs maintenance and tokens, and inflates a meaning's prominence on the hierarchy past its real rank. It is the accidental inverse of a leading word, which repeats a *token* on purpose and never the meaning.

**The environment is a source of truth too.** `package.json` scripts, config files, the directory layout, `--help` output. A document that restates them is a **cache**: a copy of a lookup, earning its load only when the lookup is expensive. Cache what the agent cannot find by looking — the unwritten convention, the reason behind a choice, the gotcha no config confesses. Leave the one-file, one-command lookups to the environment, where they cannot go stale.

**Relevance.** Does each line still bear on what the document does? A line loses relevance by never bearing on the task — mere exposition, or a branch that should have been disclosed — or by going stale as the behavior it describes moves. Shorter documents are easier to keep relevant. Without a pruning discipline the default fate is **sediment**: stale layers that settle because adding feels safe and removing feels risky, until you have to core down through them to find what is still live.

**The no-op test.** Hunt sentence by sentence: does this line change behavior versus the model's default? An instruction the model already obeys pays load to say nothing. The test is model-relative, not reader-relative — two people disagreeing about a no-op are disagreeing about the default, and they settle it by running the document, not by arguing. When a sentence fails, delete the whole sentence rather than trimming words from it.

The test also grades leading words. A word too weak to beat the default — *be thorough*, when the agent is already thorough-ish — is a no-op, and the fix is a stronger word (*relentless*), not a different technique.

### Worked rewrites: pruning

**Cache with no lookup cost**

> Before running the tests, note that this project uses npm. The test command is
> `npm test`, the build is `npm run build`, and the linter is `npm run lint`.

Delete it. Those three facts are one `package.json` read away, and the copy goes stale the first time someone switches package manager. What is worth caching is the thing no file confesses: *"the integration suite needs the docker compose stack up first, and it fails with a connection error rather than a clear message if it is not."*

**No-op**

> Be careful and think carefully about the code before making changes. Make sure
> you understand the context. Quality matters.

Delete all three sentences. None of them changes what the model does relative to its default, and "quality matters" is not a criterion anything can be checked against. If the real intent was to resist a rushed edit, the fix is a completion criterion with demand in it, not an exhortation.

**Duplication**

Every skill in this pack except `buddy-companion` and `project-memory` carries the same `**Advise first.**` paragraph, word for word. One meaning, stored everywhere. Changing how this pack asks the buddy for advice means editing every skill, and no reader can tell which copy is authoritative because they are identical by construction.

The single-source fix is one canonical statement in `buddy-companion` and a one-line pointer in each skill. It is unpaid because the reporting-contract test requires every skill's body to carry its own qualified name under a `## Buddy` heading, so the pointer cannot swallow the whole section — only the shared paragraph can go — and doing it is a pack-wide edit that belongs in its own commit, not smuggled in beside a new skill.

Named debt with a stated fix and a stated reason is not the same thing as sediment. Sediment is the layer nobody remembers deciding to keep.
