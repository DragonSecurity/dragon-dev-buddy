# The frontier method

## The tree

A design decision is rarely independent. "Users can share a document" implies a decision about who a share is addressed to, which implies a decision about whether an invited person must already have an account, which implies a decision about what an unregistered invitee can see before they sign up. Each of those hangs off the one above it: you cannot sensibly answer the last until the middle one is settled, because "nothing, they must sign up first" erases the question entirely.

Write the tree down. Not for the user — for you. A tree held vaguely in mind produces a frontier that drifts, and a drifting frontier is how a round ends up containing two questions that answer each other.

A workable node has four parts: the decision, its prerequisites, the real options, and what distinguishes them *in this system*. That last part is what turns a generic question into one worth a user's attention. "Sessions in a cookie or in Redis" is a textbook question; "sessions in a cookie or in Redis, and you told me support must be able to kick a compromised account inside a minute" is a decision.

## Computing the frontier

The frontier is the set of decisions whose prerequisites are all settled. Compute it mechanically:

1. Mark every node **settled** (the user has answered it, or a fact establishes it), **open**, or **pruned** (its parent's answer removed it).
2. A node is on the frontier if it is open and every one of its prerequisites is settled.
3. A node whose prerequisite is a fact currently being looked up is **not** on the frontier. The lookup is an unsettled prerequisite like any other.

Two nodes on the frontier at the same time must be independent of each other. If answering Q3 would change what you recommend for Q5, Q5 is not on the frontier — it hangs off Q3 and belongs to the next round. This is the check that fails most often in practice, and it fails invisibly: the user answers both, the answers conflict, and you resolve the conflict by picking one, which is the guess the whole method exists to prevent.

The frontier is usually three to eight questions. One question means you have modelled the tree as a chain, and you are about to drip. Twenty means you have flattened it and half of them depend on the other half.

## Batching a round

Everything on the frontier goes out in one message. Number them, continuing the count from the previous round rather than restarting at one. Give each a title so the user can navigate a round of seven without re-reading it, and give each a recommended answer.

The recommendation is not decoration. It is what makes a seven-question round answerable in one pass: the user reads, agrees with four, overrides three, and replies "1 yes, 2 yes, 3 use Postgres not Redis, 4 yes, 5 no — we never store the file itself, 6 yes, 7 30 days". That reply settles seven decisions. The same seven asked one at a time cost seven turns and, worse, each answer is given without sight of the decisions it constrains.

Say what a round settles in its heading, and open with what the previous round closed. The user cannot see your tree; the only evidence they have that the interview is converging is that you keep naming decisions as done and branches as pruned.

Then stop. Do not answer your own questions, do not proceed on the recommendations, do not ask a follow-up before the round comes back.

## Facts versus decisions

**A fact is anything true about the world independent of what the user wants.** It is your job, always. Look it up or dispatch a subagent for it.

- What the schema actually stores, and whether that column is nullable.
- Whether that endpoint currently requires authentication.
- What version is deployed, what the dependency tree contains, what the CI pipeline runs.
- Whether the thing the user thinks exists actually exists.
- What the framework does by default when the header is absent.

**A decision is a preference, a tradeoff or a risk acceptance.** It is the user's, always, and it is the only thing you may ask for.

- What the system should do when the check fails.
- How long data is kept and who can read it.
- Whether an invited person may see a filename before they have an account.
- Which risk is acceptable and which is not.

The boundary is sharper than it looks under pressure. "Do you use Postgres?" is a fact and asking it wastes a turn and signals you have not read the repository. "Should sessions live in Postgres?" is a decision. When you catch yourself about to ask a question whose answer is written down somewhere, go and read it instead.

Dispatch subagents in parallel with the round going out, never before it. A subagent takes a minute; a round takes the user five. Blocking the round on the subagent serialises the two, and the interview stalls for no reason.

Facts also prune. A subagent that reports "there is no tenant column on this table at all" can close three decisions that were premised on multi-tenancy already existing. Fold the report in when it lands, say what it settled, and carry on.

## The four failure modes

**Asking the user for facts.** Every one of these spends a turn of the user's attention to obtain something you could have read, and it trains the user to stop trusting that your questions are worth answering. The tell: you are asking about the present tense of the system rather than the future tense of the design.

**The drip.** One question, wait, one question, wait. It feels responsive and it is the worst way to run an interview. The user answers each question without seeing the decisions it constrains, you steer with your own follow-ups, and the design converges on whatever you guessed first. If you have asked exactly one question and are waiting, you have got the tree wrong — either the frontier really is one node, in which case say so, or you have modelled a tree as a chain.

**Asking downstream of an open question.** Two questions in one round where the answer to one changes the right answer to the other. The user answers both in good faith, the answers are incompatible, and the resolution is yours — a guess, wearing the user's authority. Check every pair in the round before sending it: if this answer changed, would that recommendation change?

**Declaring the frontier empty when it is not.** The most damaging one, because it ends the interview. It happens when the tree stopped growing: you enumerated the decisions you thought of at the start and never asked what the answers implied. Before closing, walk each settled decision once more and ask what it now makes possible, what it makes ambiguous, and what it makes dangerous. The security-shaped decisions are the ones that most often surface only on this pass — the abuse case, the retention period, the failure mode, the boundary crossing that only exists because of a choice made in round two.

## Closing into a deliverable

The interview's product is a settled tree with reasoning attached. Convert it, do not merely transcribe it.

**Into a spec** (`secure-feature-build`): decisions become requirements; ruled-out branches become non-goals, which is the part that stops them being rebuilt; assumptions still standing become the spec's stated preconditions. Every abuse case the interview surfaced goes in with its decided response, before any implementation begins.

**Into a threat model** (`threat-model`): the trust boundaries the interview named are the boundaries, and they arrive with a reason rather than being inferred from code. Decisions about what fails open versus closed become the preconditions on threats. What was ruled out constrains scope. Feed settled boundaries back into `security.trust_boundaries` so the next skill does not re-derive them.

**Into a change plan** (`change-window`): decisions become the ordered steps, the rollback, and the go/no-go criteria. "Open by choice" is the list of things to verify during the window rather than before it.

**Into config** (`buddy-setup`): the answers about exposure, data sensitivity, auth model and boundaries are exactly the keys that skill writes. An interview run before setup means the config records decided values instead of defaults nobody looked at.

Whatever the target, carry the *why*. A decision recorded without its reasoning cannot be re-evaluated when the world changes, only obeyed — and the next session will re-litigate it from scratch, arriving somewhere else.
