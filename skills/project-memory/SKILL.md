---
name: project-memory
description: Records and recalls what a codebase taught you — the constraints, rejected approaches and gotchas that the code does not state and git history does not imply. Use when someone says "remember this", "make a note of that", "don't forget", "why did we do it this way", "what do you know about this repo", "we tried that before", "write that down", "add a memory", or when a session rediscovers something a previous session already paid for. Memories stay local to the machine; a fact worth publishing belongs in the repository's own docs instead.
---

# Project Memory

A codebase states what it does. It rarely states what it refuses to do, what was tried and abandoned, or which innocuous-looking change broke production last spring. That knowledge lives in whoever was there, and it leaves when the session ends.

This skill writes it down. A memory is one fact about this project, in a file next to the project, loaded automatically at the start of every future session by the `project-memory` hook this pack ships. Sessions are disposable; the memory outlives them.

The discipline that makes this work is refusing to write most of what you could. A directory of forty memories that restate the README is worse than five that carry things no one could have read off the code — it costs context on every single session and buries the ones that matter.

## First-run check

Read `.dragon-buddy/config.json`. This skill uses `project.name` for the header on new memories and nothing else, so a missing config is not fatal — but if it is absent, say so and offer `dragon-dev-buddy:buddy-setup`, since every other skill in this pack needs it.

Memories live in `.dragon-buddy/memories/`. If the directory does not exist, create it when writing the first memory, not before.

## Inputs

Ask for nothing that can be read. Specifically, do not ask for the project name, the language, or what the repository does.

Ask only when you are about to write a memory and cannot tell:
- Whether the fact is durable or an artefact of the current task.
- Whether a constraint is a decision someone made or an accident nobody has fixed.

## Workflow

1. **Check the guardrails before writing anything.** Confirm `.dragon-buddy/memories/` is ignored by git — the ignore rule is what keeps memories local. If it is not, add it to `.gitignore` and say so.

   Then offer the commit guard, which is the backstop for `git add -f` and for a repository whose ignore rule drifted. **Find where git actually looks for hooks first:**

   ```sh
   hooks_dir="$(git config --get core.hooksPath || echo .git/hooks)"
   ```

   `core.hooksPath` overrides `.git/hooks` completely, and it is commonly set globally by dotfiles and by tools like husky. Writing to `.git/hooks/pre-commit` on a machine that has it set installs a guard that is never invoked — it fails open, silently, which is the one way a guard is worse than none. Never assume the default path; read it.

   ```sh
   cp "${CLAUDE_PLUGIN_ROOT}/scripts/pre-commit-memory-guard.sh" "$hooks_dir/pre-commit"
   chmod +x "$hooks_dir/pre-commit"
   ```

   Copy rather than symlink: the plugin lives at a version-keyed path, so a link into it dangles on the next update.

   Two things to say out loud before doing it:
   - If `core.hooksPath` points somewhere global, the guard now applies to **every repository on the machine**. That is usually right for this particular guard — memories are local everywhere — but it is a machine-wide change and the user's call, not yours.
   - If a `pre-commit` hook already exists at that path, do not overwrite it. Add a call to the guard from inside it.

   **Verify it fires**, rather than assuming the copy was enough:

   ```sh
   mkdir -p .dragon-buddy/memories && : > .dragon-buddy/memories/.guard-probe
   git add -f .dragon-buddy/memories/.guard-probe
   git commit -m "guard probe" --dry-run   # must fail
   git reset -q .dragon-buddy/memories/.guard-probe && rm .dragon-buddy/memories/.guard-probe
   ```

   An install that is not observed refusing something has not been tested.

2. **Decide whether this is a memory at all.** Apply the test below. Most candidates fail it, and that is the point.

3. **Look for an existing memory that covers it.** Read the directory first. Update the file that is nearly right rather than adding a second one beside it — two memories on one subject disagree eventually, and the reader cannot tell which is current.

4. **Write one fact per file.** Slug-named, frontmatter first. Convert relative dates to absolute ones: "last week" is meaningless to the session that reads it in March.

5. **Say why.** A memory that records a decision without its reasoning cannot be re-evaluated, only obeyed. When the reasoning expires, the memory should visibly expire with it.

6. **Link related memories** with `[[slug]]`. A link to a memory that does not exist yet is fine — it marks something worth writing.

7. **Delete what turned out to be wrong.** A stale memory is worse than no memory, because it is trusted. When a session discovers a memory is false, remove it in the same breath rather than leaving it for later.

### What is a memory

Write it down when it is **durable**, **not derivable**, and **would cost a future session real time**:

- A constraint with a non-obvious reason. *"Releases are cut from tags because the tag is the distribution channel; a merge to main reaches nobody."*
- An approach that was tried and rejected, and why. Without this, it gets tried again.
- A gotcha with a delayed or silent failure. *"Hook registration is read at session start, so a hooks.json change needs a restart even though the script itself is live."*
- Where something non-obvious lives, when the name does not suggest it.

Do **not** write it down when it is:

- **Derivable from the repository.** Structure, dependencies, what a function does, what the README says. The session can read those.
- **In git history.** Past fixes, who changed what, when a bug was introduced.
- **True only of this conversation.** The task in flight, a temporary workaround, what you are about to do next.
- **A preference that spans projects.** Licence choice, commit style, editor setup. Those belong in the user's own global memory, not in one repository's directory.
- **Secret.** Credentials, tokens, customer names, anything from an engagement under NDA. Memories are gitignored, not encrypted, and they sit in a working tree that gets backed up, shared and screen-shared.

## Output format

One file per memory at `.dragon-buddy/memories/<slug>.md`:

```markdown
---
name: <short-kebab-case-slug>
description: <one line — this is what a future session reads to decide relevance>
metadata:
  type: constraint | decision | gotcha | context
---

<The fact, stated plainly.>

**Why:** <the reasoning, so it can be re-evaluated rather than merely obeyed>

**How to apply:** <what a session should actually do differently>

<Related: [[other-slug]]>
```

The `description` carries the whole routing decision when the set grows past the context budget — at that point the hook sends descriptions only. Write it as a claim, not a topic: "the tag is the distribution channel, not main" beats "notes about releasing".

## Buddy

Before a recall-only run, skip the buddy — reading memories is not work.

Before writing or curating, call `buddy_advise` with what you are about to record.

After writing, updating or deleting memories, call `buddy_observe`:
- `summary`: `"Recorded <n> project memories about <subject>."` or `"Removed a project memory that was no longer true: <subject>."`
- `kind`: `docs`
- `skills_used`: `["dragon-dev-buddy:project-memory"]`

## File output

Writes only inside `.dragon-buddy/memories/`, and `.gitignore` when the ignore rule is missing.

Never modifies source, configuration, or `.dragon-buddy/config.json`. Never commits — the directory is local by design, and `scripts/pre-commit-memory-guard.sh` enforces it.

## Reference library

- `references/memory-criteria.md` — the keep-or-drop test worked through on real candidates, and the failure modes of a directory that grew without one. Load it when deciding whether something is worth recording, or when curating a set that has got large.

## Worked example

`examples/project-memory-run.md` is the quality target: a session that records two memories, rejects four candidates out loud with reasons, and deletes one that had gone stale.

## Quality bar

A good run satisfies all of these:

- Every memory written is durable, not derivable from the repository, and not in git history.
- Every memory states its reasoning, not just its conclusion.
- Relative dates were converted to absolute ones.
- Existing memories were read before new ones were added, and near-duplicates were updated in place rather than appended beside.
- Candidates that were rejected were named, with the reason — a silent rejection is indistinguishable from an oversight.
- Nothing secret, personal, or client-identifying was written.
- `.dragon-buddy/memories/` is ignored by git, and the commit guard was offered.
