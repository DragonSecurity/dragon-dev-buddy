# Worked example: recording what a debugging session cost

A session has just spent three hours on a release that appeared to succeed and reached nobody. The user says: *"write that down so we don't do it again"*.

---

**Guardrails first.**

```sh
$ git check-ignore -q .dragon-buddy/memories/ && echo ignored || echo NOT IGNORED
NOT IGNORED
```

> `.dragon-buddy/memories/` is not ignored here, so anything I write would be committable. Adding the rule before writing a word:

```sh
$ printf '\n# Project memories, local to this machine by design.\n.dragon-buddy/memories/\n' >> .gitignore
$ git config --get core.hooksPath || echo .git/hooks
~/.config/git/hooks
```

> Not the default. `core.hooksPath` is set globally on this machine, so `.git/hooks/` is never consulted — installing there would have produced a guard that never runs. Installing where git actually looks:

```sh
$ cp "${CLAUDE_PLUGIN_ROOT}/scripts/pre-commit-memory-guard.sh" ~/.config/git/hooks/pre-commit
$ chmod +x ~/.config/git/hooks/pre-commit
```

> Flagging that this path is global: the guard now applies to every repository on this machine. For this particular guard that is what you want — memories are local everywhere — but it is a machine-wide change, so it is your call. There was no existing `pre-commit` there to overwrite; if there had been, I would have added a call from inside it instead.

**Verify it actually fires**, rather than trusting the copy:

```sh
$ : > .dragon-buddy/memories/.guard-probe && git add -f .dragon-buddy/memories/.guard-probe
$ ~/.config/git/hooks/pre-commit; echo "exit=$?"
pre-commit: refusing to commit project memories.

  .dragon-buddy/memories/.guard-probe
exit=1
$ git reset -q .dragon-buddy/memories/.guard-probe && rm .dragon-buddy/memories/.guard-probe
```

> Running the hook directly, not `git commit --dry-run` — that does not execute pre-commit hooks at all and exits 0 against a guard that would have refused, so it would report a working install while proving nothing. Observed refusing something. An install that has not been seen to block is not an install that works.

---

**Read what is already there.**

```sh
$ ls .dragon-buddy/memories/
release-flow.md   node-version-floor.md
```

> Two existing memories. `release-flow.md` is close to what I am about to write, so I will update it rather than add a second file about releasing.

---

**Candidates from this session, and the verdict on each.**

| Candidate | Verdict |
| --- | --- |
| The release workflow refuses a tag that is not an ancestor of `main` | **Drop** — derivable, and it fails loudly with that exact message |
| Merging to `main` publishes nothing; the tag is the distribution channel | **Keep** — this is the assumption that cost three hours |
| Hook registration is read at session start; the script is live but the registration is not | **Keep** — silent, and it looks like the change applied |
| The bug was introduced when hooks moved into the plugin | **Drop** — `git log` |
| `hooks/` contains three `.mjs` files | **Drop** — derivable, and stale the moment one is added |
| We are mid-way through moving memories into the pack | **Drop** — task state, true for a week |

> Four of six dropped. Two written.

---

**The update, to `release-flow.md`:**

```markdown
---
name: release-flow
description: A merge to main publishes nothing here — the git tag is the distribution channel
metadata:
  type: constraint
---

Merging to `main` makes no release. The tag is what consumers resolve: the
marketplace compares the manifest version it finds on a tag, and downstream
installs pin a semver range that matches tags and nothing else.

**Why:** distribution was deliberately decoupled from the default branch so that
landing work and publishing it are two decisions. The cost is that a merged pull
request looks finished and has reached nobody.

**How to apply:** finish every change by tagging `main` and pushing the tag.
If two repositories move together, tag the dependency first — a range that
matches no tag fails at install time, not at merge time. See [[hook-registration-timing]].
```

**The new one, `hook-registration-timing.md`:**

```markdown
---
name: hook-registration-timing
description: Hook scripts are live immediately, but hook registration is only read at session start
metadata:
  type: gotcha
---

Editing a hook *script* takes effect immediately. Editing which hooks exist, or
what they match, does not — that is read once when the session starts.

**Why:** the client loads hook configuration at startup and holds it for the
session. A symlinked or in-place checkout makes the script live and gives the
misleading impression that the whole thing is.

**How to apply:** after changing `hooks.json`, restart before concluding the fix
did not work. On 2026-08-09 this cost three hours across two wrong hypotheses,
because the gate kept firing and the corrected matcher was already on disk.
```

> Both state their reasoning, so either can be re-evaluated if the constraint changes. The date is absolute — "today" would be meaningless on the next read.

---

**Curation pass.**

```sh
$ head -6 .dragon-buddy/memories/node-version-floor.md
```

> This one says the floor is Node 20 because of a `node:sqlite` flag. `package.json` now says `>=24` and the flag is long gone, so the memory is false and, worse, would be trusted. Deleting it rather than editing it — the fact it recorded no longer exists.

```sh
$ rm .dragon-buddy/memories/node-version-floor.md
```

---

**Result**

- Recorded: 2 (1 updated in place, 1 new)
- Rejected: 4, each with a reason
- Deleted: 1 that was no longer true
- Guardrails: ignore rule added, commit guard installed

Reported to the buddy as `docs`, `skills_used: ["dragon-dev-buddy:project-memory"]`.
