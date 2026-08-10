# Worked example: cutting a bloated draft skill down until it fires

---

**User:** I drafted a new skill for the pack — `detection-review`, it checks whether an app logs enough to notice an attack. It's in skills/detection-review/. Can you get it into shape?

**Assistant:** Reading the draft and one existing skill beside it, since the house structure is the contract here.

*(reads `skills/detection-review/SKILL.md` — 340 lines — and `skills/refactor-safely/SKILL.md`)*

Here is what the draft is, before I touch it: a good idea, 340 lines long, with a description that will never fire it, four of the ten headings, a 70-line platform table inline, and a shell script sitting in `references/` where nothing can reach it.

Taking those in the order that matters. The description first — a skill nobody reaches is worth zero however good the body is.

**Before:**

```
description: A thorough and comprehensive skill for reviewing application
  logging and detection capabilities, helping teams ensure they have the
  observability they need to detect security incidents using industry standard
  practices.
```

Nothing in that is a condition. It says what the skill *is*, three times, and never says when to reach it. There is no `Use when`, which also fails the pack's own test. Every phrase in it — "thorough", "comprehensive", "industry standard practices" — routes nothing.

**After:**

```
description: Establishes whether an application would actually notice an attack:
  which security-relevant events are logged, which of those reach somewhere a
  human or an alert would see, and what an attacker could do that leaves no
  trace. Use when someone says "do we log enough", "would we notice", "detection
  review", "what would this attack look like in our logs", "audit our logging",
  "we had an incident and had nothing to go on", or after adding any
  authentication, authorization or admin path.
```

Four things changed. The branches are enumerated (logged / reaches somewhere / leaves no trace) instead of summarised. Every trigger is a phrase someone types, including the one people actually say after a bad week — "we had an incident and had nothing to go on". The last clause is a condition with no phrase in it, so the skill fires on a new admin route even when nobody asked. And "thorough and comprehensive" is gone, because it was doing no work in a string that is loaded on every turn of every session.

**The structure.** The draft has `## Overview`, `## Steps`, `## Notes`, `## Examples`. The pack's contract is ten headings in a fixed order, so `## Steps` becomes `## Workflow`, `## Notes` splits into `## Output format` and `## File output`, and the four missing ones get written: `## First-run check`, `## Inputs`, `## Buddy (optional, when the MCP server is connected)`, `## Reference library`, `## Quality bar`.

The First-run check is where the config keys go. This skill needs `project.stack` to know what log platform to expect and `security.exposure` to know what an attacker is likely to reach, and nothing else — so it reads those two and says so, rather than demanding the whole file.

**One section goes behind a pointer.** Lines 96–166 are a table of where each platform writes its audit trail and what fields it carries — CloudTrail, Kubernetes audit, nginx, Postgres, the app's own framework. It is good material and every run pays for it today.

The test is mechanical: does every branch need it? No. A review of an app's own handlers never opens the platform table, and a review of cloud audit posture never reads the framework rows. So it moves to `references/platform-audit-trails.md`, and the body keeps one conditional line naming it. Seventy lines leave the top of the file; the branch that needs them still gets them.

The security-control list at lines 40–52 stays inline, by the same test. Every run needs it, and putting the most important step of the skill behind a pointer would make it conditional on the pointer firing.

**One passage collapses into a leading word.** The draft says this, in three different places, slightly differently each time:

> ...an event that is written to a log file that nobody reads, that no alert is
> configured against, and whose retention window expires before anyone would
> look at it.

That is one concept spelled out at three sites. Named once, it becomes a **blind spot**, and then the skill can say "enumerate the blind spots", "rank blind spots by what an attacker reaches through them", and "a blind spot with a control behind it is a finding, not a note" — three usable phrases where there were three restatements. The word is doing the work the paragraph was doing, and the agent has something to look *for* rather than a description to re-read.

**The no-op pass.** Sentence by sentence, asking whether each line changes what the model does versus its default. Nine sentences deleted. Three of them:

- "Be thorough and use your best judgment when reviewing the logging code." — Deleted whole. The model's default is already this, and "best judgment" is not a criterion anything can be checked against. Where the draft wanted resistance to a shallow pass, the fix is in the completion criterion, not an exhortation.
- "Remember that security is important." — Deleted whole.
- "This project uses npm, so run `npm test` to check your work." — Deleted. That is one `package.json` read away, it is a cache with no lookup cost behind it, and it is wrong the day someone switches package manager. What is worth caching is what no file confesses, so the line that replaced it, in the First-run check, is `practice.test_command` from config with a note that the integration suite needs the compose stack up first and fails with a bare connection error if it is not.

**One prohibition turned around.** The draft says "don't just list every log line you find, and don't produce a huge dump of findings with no priority". Both bans put the unwanted behavior into context, where it becomes more available rather than less. Rewritten positive: "Rank findings by what an attacker reaches through the blind spot, and report the top five with the path each one hides."

**Two completion criteria sharpened.** "Review the logging thoroughly" became "Every security-relevant event in the target has one of three verdicts: logged and alerted, logged and dark, or absent." That is checkable, it is exhaustive, and it forces the legwork the word "thoroughly" was standing in for.

**The shell script.** `references/collect-log-config.sh` cannot work where it is: only `.md` files under `references/` can be named in prose, so a script there is a file no SKILL.md can ever point at, and the build fails on it. It moves to the repo-root `scripts/` directory and the skill invokes it through `${CLAUDE_PLUGIN_ROOT}`, the way `project-memory` installs its commit guard.

**Registration.** Row added to the README's Audit group, README prose count moved up by one, `CHANGELOG.md` entry written under `## Unreleased`. No version bump: the three version files move together when a release is cut, not when a skill lands.

**Result:** 340 lines to 118, plus a 70-line reference file and a worked example. `go test ./...` green.

**Debt named, not paid:** this skill wants the same `**Advise first.**` paragraph every other skill carries, and copying it in makes one more identical copy of a paragraph that already sits in every skill but two. The single-source fix is one canonical statement in `buddy-companion` with a pointer from each skill, and it is a pack-wide edit in its own commit — not something a new skill gets to smuggle in. So the draft carries the copy, and the debt is written down here.

---

## What this run got right

- Fixed the description before anything else, because a skill that does not fire is worth nothing however good the body is — and replaced self-description with the phrases a user actually types.
- Enumerated the branches in the description instead of summarising them, and added a condition with no trigger phrase at all so the skill fires on a change nobody thought to ask about.
- Used one mechanical test for disclosure — does every branch need this? — and applied it in both directions, moving the platform table out and deliberately keeping the control list in.
- Found a restated concept at three sites and collapsed it into one pretrained word, so the agent has something to look for rather than a paragraph to re-read.
- Deleted whole sentences rather than trimming them, and named the deletions with the reason instead of quietly shrinking the file.
- Replaced a cache that had no lookup cost with the gotcha the environment cannot answer.
- Turned two prohibitions into one positive instruction, so the unwanted behavior is never named.
- Moved the shell script out of `references/`, where it was unreachable by construction.
- Registered the skill in the README table, the README count and the changelog in the same change, and left the version files alone.
- Named the pack's duplication debt with its fix and its reason, and did not edit every other skill in the pack to pay it.
