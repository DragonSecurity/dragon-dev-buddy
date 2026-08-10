# Deepening a cluster, and designing its interface twice

How to take a cluster of shallow modules and put one deep interface in front of it, given what it depends on — and how to avoid shipping the first interface you thought of. Assumes the vocabulary in the skill body: module, interface, seam, adapter, depth, leverage, locality.

## Dependency categories

Classify every dependency of the candidate before placing the seam. The category decides how the deepened module is tested and whether a port is warranted at all. Most bad seams are a dependency filed in the wrong category — usually category 2 mistaken for category 4, which produces a mock where a real stand-in was available and tests that pass against behavior the production dependency does not have.

### 1. In-process

Pure computation, in-memory state, no I/O. Always deepenable: merge the modules, test through the new interface directly, introduce no port. A seam here buys nothing and costs a layer of indirection.

### 2. Local-substitutable

Dependencies with a real local stand-in — an embedded Postgres, an in-memory filesystem, a container that starts in a second. Deepenable if the stand-in exists. Run the stand-in in the test suite and test the deepened module against it. The seam is internal to the implementation; nothing appears at the module's external interface.

This is the right category for a database in most codebases, and it matters for security work: a tenant predicate or a row-level policy tested against a mock repository proves that your fake honors your fake. Only the real engine tells you whether the policy holds.

### 3. Remote but owned

Your own services across a network — internal APIs, queues, another team's gRPC endpoint. Define a port at the seam. The deep module owns the logic; the transport is injected as an adapter. Production gets the HTTP or gRPC or queue adapter; tests get an in-memory one. The recommendation reads: *define a port at the seam, implement an HTTP adapter for production and an in-memory adapter for testing, so the logic sits in one deep module even though it is deployed across a network.*

A network hop is usually also a trust boundary. If it is, the validation and authorization belong on the module's side of the port, not inside the adapter, or swapping the adapter swaps out the control.

### 4. True external

Third-party services you do not control — a payment processor, an SMS gateway, an identity provider. The deepened module takes the dependency as an injected port; tests supply a mock adapter. Accept that the mock encodes your belief about the third party rather than its behavior, and keep at least one test against the real sandbox for anything security-relevant: signature verification, webhook authenticity, token exchange.

## Seam discipline

**One adapter means a hypothetical seam. Two adapters means a real one.** Do not introduce a port unless at least two adapters are justified — typically production and test. A single-adapter port is indirection that has to be maintained, navigated and explained forever in exchange for a flexibility nobody asked for.

**Internal seams are private.** A deep module may have as many internal seams as its implementation wants, used by its own tests. They do not belong in the external interface. Exposing one because a test found it convenient is the standard route from deep to shallow: the interface grows, callers start depending on the internal shape, and the module can no longer change behind its own back.

**A seam is not automatically a trust boundary.** Place seams to concentrate change; place validation where trust changes. When the two coincide — and at a request edge, a tenant scope or an output encoder they should — put the control at the seam so it cannot be bypassed. When they do not coincide, do not add ceremonial validation at a seam that crosses no trust change; it costs real code and teaches readers that validation is decorative.

## Testing strategy: replace, don't layer

New tests go at the deepened module's interface. The interface is the test surface.

Old unit tests against the shallow pieces become waste the moment the interface-level tests exist. Delete them in the same change. Keeping both is not extra safety: it is two suites that will disagree, and the one testing the old shape is the one that will block the next internal change for no reason.

Tests assert observable outcomes across the interface, never internal state. A test that has to be edited when the implementation changes is testing past the interface, and is a signal about the module's shape rather than about the test.

For security properties, the property must be asserted at the deepened interface and not only inside. "The repository scopes by tenant" is a claim about the interface, and the test is a call through it with tenant A's session asking for tenant B's row. `security-test-writer` writes these; this reference only says where they attach.

## Design it twice

Your first interface is unlikely to be your best, and by the time that becomes obvious every caller is written against it. For any module big enough that callers will be expensive to move, design it several radically different ways before choosing.

### 1. Frame the problem space

Before spawning anything, write the frame for the chosen candidate:

- The constraints any new interface must satisfy — invariants, ordering, error modes, the security property that must hold.
- Every dependency and its category from above.
- A rough illustrative sketch to make the constraints concrete. It is a grounding device, not a proposal, and saying so prevents the subagents from converging on it.

Show the frame to the user, then start the subagents immediately. The user reads while the work happens.

### 2. Brief parallel subagents

Spawn three or more in parallel. Each gets a separate technical brief — file paths, the coupling that actually exists, the dependency categories, what sits behind the proposed seam — and each gets a **different design constraint** so the results diverge:

- *Minimize the interface.* One to three entry points. Maximize leverage per entry point.
- *Maximize flexibility.* Support many use cases and future extension.
- *Optimize for the common caller.* Make the overwhelmingly common case trivial and let the rare case be verbose.
- *Ports and adapters.* Design around the cross-seam dependencies explicitly.

Where the module carries a security control, add a fifth: *make the unsafe call unrepresentable.* Design an interface in which the scatter from the audit cannot be written — no unscoped query, no unescaped render — even at the cost of ergonomics. It frequently wins outright, and when it does not it usually contributes the type that the winner adopts.

Put the vocabulary in every brief so the designs come back in the same words and can be compared at all. Include the project's own domain nouns for the same reason.

Each subagent returns:

1. The interface — entry points, types and parameters, plus invariants, ordering constraints and error modes.
2. A usage example showing a real caller.
3. What the implementation hides behind the seam.
4. The dependency strategy and the adapters, by category.
5. Trade-offs: where leverage is high, where it is thin, and what the design makes harder.

### 3. Compare and recommend

Present the designs one at a time so each can be absorbed, then compare them in prose on **depth** (leverage at the interface), **locality** (where change concentrates when the requirement moves) and **seam placement** (where the interface sits, and whether it lands on a trust boundary).

Then recommend one, in your own voice, with the reason. Propose a hybrid where elements combine well — the minimal design's entry points with the unrepresentable-unsafe-call design's types is the most common good hybrid. Be opinionated. A menu of four designs and no recommendation moves the work back onto the person who asked.
