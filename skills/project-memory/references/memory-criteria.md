# The keep-or-drop test

A memory costs context on every session in this project, forever, whether or not it turns out to be relevant. That is the whole economics of the directory: the cost is certain and recurring, the benefit is occasional. A fact has to earn its place against that.

Three questions, all of which must be yes.

## 1. Is it durable?

Will this still be true in three months?

The failure mode is recording the present tense. "We are mid-migration to the new auth service" is true today and misleading in six weeks, at which point it actively misinforms a session that trusts it. If a fact has an expiry, either write the expiry into it — "until the v3 migration lands, both tables are written" — or do not write it.

Task state is never durable. What you are doing right now belongs in the conversation, not on disk.

## 2. Is it derivable?

Could a session find this out by reading the repository?

This is the one that catches most candidates. It is tempting to record a map of the codebase, and it is nearly always waste: the session can read the codebase, and your map goes stale the moment someone moves a file. The same applies to dependency lists, what a function does, the test command, and anything the README already states.

Git history counts as derivable. "This bug was introduced in the caching refactor" is `git log`'s job.

The useful residue is the reasoning that never got committed. A file can show you *that* retries are capped at three. Only a memory can tell you the cap is three because the upstream provider rate-limits at four and the fourth attempt gets the whole IP blocked for an hour.

## 3. Would it cost a future session real time?

Would someone waste an afternoon, or ship something wrong, without it?

The strongest memories describe **silent** failures — things that do not error, just quietly do the wrong thing. A loud failure teaches itself; you hit it, you read the message, you fix it. A silent one can be rediscovered indefinitely.

Weak memories are conveniences: mild time-savers that would have been obvious within minutes. Those are not worth a permanent context cost.

## Worked candidates

| Candidate | Verdict | Why |
| --- | --- | --- |
| "The release workflow refuses a tag that is not an ancestor of main." | **Drop** | Derivable — it is in the workflow, and it fails loudly with that exact message. |
| "Releases are cut from tags because the tag is the distribution channel; merging to main reaches no one." | **Keep** | The reasoning is nowhere in the code, and without it the natural assumption is that merging is enough. |
| "The auth module is in `src/auth/`." | **Drop** | Derivable, and wrong the first time someone moves it. |
| "Hook registration is read at session start, so a hooks.json change needs a restart even though the script is live." | **Keep** | Silent: the change appears to have applied. Cost an afternoon once already. |
| "We use Apache-2.0." | **Drop** | In `LICENSE`. Also a cross-project preference, which belongs in global memory rather than one repository. |
| "Retries are capped at three because the provider blocks the IP on the fourth." | **Keep** | Not derivable, and raising the cap looks harmless right up until it is not. |
| "Currently refactoring the payment handler." | **Drop** | Task state. True for a week. |
| "The staging database is a restored production snapshot, so it contains real customer records." | **Keep the fact, not the detail** | Durable and expensive to learn the hard way. Record the constraint; never record a hostname, credential or customer name. |

## When the set gets large

Past the context budget the hook stops sending bodies and sends descriptions only. That is a working fallback, not a good state — it means every session pays for a listing it mostly does not need, and relevance now rests entirely on how well each description is written.

Treat crossing that threshold as a signal to curate rather than as a feature:

- **Merge** memories that circle the same subject. Three notes on the release process should be one.
- **Delete** anything whose reasoning has expired. A constraint outlives the reason for it and becomes cargo cult; deleting it is the only way that stops.
- **Promote** anything that turned out to be genuinely public — an architectural decision the whole team needs — into the repository's own docs, where it can be reviewed and versioned. Memories are for what stays local.

A directory that only ever grows is not a memory. It is a landfill with a search cost.
