# Guardrail patterns: what is blocked, what is not, and what the hook cannot see

## The test for admission

An operation belongs in the block list when it **destroys work that cannot be recovered from this clone**. Two consequences follow, and both are load-bearing:

- Anything recoverable from the reflog, from a remote, or from the index is not blocked, however alarming it looks. `git commit --amend` is reversible for ninety days. `git reset --hard` is not reversible at all for anything that was never committed.
- Anything that is part of the normal working day is not blocked, even when a careless use of it would hurt. Pushing a branch is how work reaches review. If the guard interferes with that, it is removed, and then it protects nothing at all. Every entry below was checked against "would a normal week hit this?"

## The rules

Each name is a line in the `RULES` array at the top of the script. Comment the line out to turn the rule off. The one exception is `extra-deny`, which is the last entry below: it has no line in `RULES` because it is not a git rule at all — it is driven by the `EXTRA_DENY` array, and it is turned off by emptying that array rather than by commenting anything out.

### `push-force` — `--force`, `-f`, `--force-with-lease`, `--force-if-includes`, a `+refspec`

The remote ref is replaced rather than advanced. Commits that existed only on the remote — someone else's push in the last ten minutes, a CI-generated commit — are unreachable afterwards, and no local reflog knows about them because they were never local. `--force-with-lease` is safer than `--force` and still destroys anything fetched since your last fetch. `git push origin +main` is `--force` in another spelling, which is why the parser reads refspecs and not just flags.

This one is blocked for **every** branch, not only protected ones, because that is the operation the skill exists for: a history rewrite pushed anywhere is the class of mistake that cannot be undone locally.

### `push-protected` — any push whose destination is a protected branch

Not because the push is destructive — a fast-forward to `main` destroys nothing — but because it routes around the place the checks live. In a repository where `main` is reached by pull request, a direct push skips review, skips required status checks, and produces a commit nobody has seen. Where the branch is protected server-side the push is refused anyway, after the fact, in a way an agent will happily retry with `--force`.

Destination is parsed properly, not prefix-matched: `git push origin feature:main`, `git push origin HEAD:refs/heads/main`, `git push origin :main` (a deletion), and a bare `git push` while `main` is checked out all resolve to the same protected destination.

The branch a bare push resolves to is the one you are standing on, which makes `--tags` the exception worth stating: it pushes tags and no branch, so there is no destination to protect and it is allowed even from `main`. `--follow-tags` is a different flag — it pushes the current branch alongside the tags — and is judged as the branch push it is. This is the distinction 1.4.0 got wrong, refusing `git push --tags` from `main` and so refusing the step that publishes a release; `scripts/check-git-guardrails.sh` now holds both cases so the fix cannot quietly reverse.

### `push-mirror` — `--mirror`, `--prune`

Both delete remote refs that are absent locally. A branch a colleague pushed this morning and you have not fetched is gone, along with its commits. This is the most surprising entry on the list precisely because neither flag has "force" in its name.

### `reset-hard` — `git reset --hard`

Discards the working tree and the index. Committed work survives in the reflog; uncommitted work has never been in an object database and is gone the instant this returns. It is also the most common agent reflex when a build breaks, which is what makes it worth a hook rather than a habit.

`git reset --soft` and `git reset` (mixed) are not blocked. They move a ref and leave the tree.

### `clean-force` — `git clean -f`, `-fd`, `-fdx`, `--force`

Deletes untracked files from disk. Untracked means git never had a copy, so there is nothing to recover from — and `-x` adds the ignored files, which is where `.env`, local certificates and a database file usually live. `git clean -nd` (dry run) is allowed, and a `-n` anywhere in the flags makes the command a dry run, so `git clean -fdn` passes too.

### `branch-delete-force` — `git branch -D`, `--delete --force`, and `-M` onto a protected name

`-d` refuses to delete a branch with unmerged commits. `-D` is exactly the instruction to ignore that refusal. The commits linger in the reflog until they are collected, which is a recovery path that expires — and the person who needs it is usually finding out three weeks later. `git branch -M main` is a forced rename over an existing name, which is a delete wearing a different hat, so it is blocked when the target is protected.

### `checkout-discard` — `git checkout .`, `git checkout -- .`, `git checkout -f`

Restores every tracked file from the index across the whole tree. It reads as navigation and behaves as deletion. `-f` is the same thing in service of a branch switch: it throws away conflicting local modifications so the switch can succeed.

Checking out a **named path** is not blocked. `git checkout src/app.ts` discards one file the user can see in the diff; `.` discards everything they had forgotten about.

### `restore-discard` — `git restore .`

The newer spelling of the same operation, blocked on the same argument. Named paths pass.

### `stash-destroy` — `git stash drop`, `git stash clear`

The stash is a ref with a reflog you cannot browse in the normal way, and `clear` empties it in one line. Recovering a dropped stash means going through dangling commits by hash. `git stash pop` is not blocked: it drops only after a successful apply, and it fails safe on conflict.

### `rebase-protected` — a rebase that rewrites a protected branch

This is the entry most worth reading carefully, because the obvious rule is the wrong one. **`git rebase main` from a feature branch is not blocked.** It rewrites your branch, not `main`, and it is how most teams stay current. Blocking it would be the same mistake as blocking `git push`.

What is blocked is a rebase whose *rewritten* branch is protected: `git rebase -i HEAD~3` while `main` is checked out, or `git rebase upstream/main main`. The script decides by parsing positionals — `git rebase <upstream> <branch>` rewrites `<branch>`, everything else rewrites the current branch. `--abort`, `--continue`, `--skip` and `--quit` always pass, because refusing to let someone out of a half-finished rebase is its own hazard.

### `history-rewrite` — `git filter-branch`, `git filter-repo`

Every commit gets a new hash. Every clone, every open pull request, every tag, every CI reference and every deployment pinned to a SHA is invalidated at once, and the coordination cost lands on people who were not in the room.

The usual trigger is a committed secret. Rotate the credential first — that is what actually closes the exposure. The rewrite only removes the copy in this repository's history, and never the copies in forks, in the platform's dangling objects, or in whatever already scraped it.

### `reflog-destroy` — `git reflog expire`, `git reflog delete`, `git gc --prune=now|=all`

The reflog is the undo path for everything above it. `gc --prune=now` collects the unreachable objects that a reset or a rebase just orphaned — the exact objects a recovery needs. Blocking these is what makes the rest of the list survivable when a rule is turned off or a command slips past.

### `extra-deny` — regexes against the whole command line

For destruction that is not a git subcommand. The one shipped default matches `rm -rf` against a path ending in `.git`, which removes the repository itself, reflog and all. It is written to not fire on `.github`, and it does fire on a `.git` reached by a relative path (`rm -rf ../old-checkout/.git`).

Configured in `EXTRA_DENY`, not `RULES`. It still answers to `GIT_GUARDRAILS_ALLOW=extra-deny` like every other rule, so a session that genuinely needs it can be excepted without editing the script.

## Deliberately not blocked

| Command | Why it stays allowed |
| --- | --- |
| `git push origin <branch>` | How work reaches review. Blocking it is what gets the whole hook uninstalled. |
| `git push origin --delete <branch>` | Deleting a merged feature branch is routine cleanup; the commits are in the target branch. A delete aimed at a protected branch is still caught by `push-protected`. |
| `git commit --amend` | Local, and the pre-amend commit sits in the reflog. Only becomes dangerous when force-pushed, which is separately blocked. |
| `git rebase <upstream>` from a feature branch | Rewrites your own branch. Normal, frequent, recoverable from the reflog. |
| `git stash pop` | Drops only after a successful apply. |
| `git reset --soft`, `git reset` | Move a ref; the tree is untouched. |
| `git checkout <path>`, `git restore <path>` | Scoped to a file the user is looking at. `.` is the dangerous argument, not the verb. |
| `git clean -n`, `-nd` | The dry run is the thing you want people reaching for. |
| `git gc` without `--prune=now` | Respects the two-week grace period, so recovery is still possible. |
| `git worktree remove --force`, `git submodule deinit -f` | Rare enough that a rule for them costs more attention than it returns. Add them to `RULES` if your repository lives on them. |

## PreToolUse semantics

The hook is fed a JSON object on stdin containing `tool_name`, `tool_input` and session metadata. Only the exit code decides the outcome:

| Exit | Effect |
| --- | --- |
| `0` | The command runs. stdout is not shown to the model in the normal flow. |
| `2` | The command is **blocked** and stderr is fed back to the model as the reason. |
| anything else | Treated as a hook error, surfaced as a warning — **and the command runs**. |

That last row is the one to internalise. A syntax error, a missing interpreter, a `chmod` that was forgotten, an unbound variable under `set -u` — all of them exit non-zero and non-2, and every one of them lets the command through while looking, from the outside, exactly like a working install. This is why the script fails **closed** (exit 2, with an explanation) when it cannot parse its input or find `jq` or `python3`, and why the install step is not finished until a refusal has been seen with its own eyes.

### The matcher pitfall

`matcher` matches the **tool name**, and it is a regular expression, not a glob. `"Bash"` is right for shell commands. What it does not cover:

- **MCP tools**, which are addressed as `mcp__<server>__<tool>` — for this pack's own buddy, `mcp__plugin_dragon-dev-buddy_buddy__buddy_observe` when the plugin declares the server and `mcp__buddy__buddy_observe` when it is registered by hand. The server segment changes with the installation, which is why a matcher written against one spelling silently matches nothing on the other half of installs. If a git-capable MCP server is in play, it needs its own matcher, and the payload shape will not be `tool_input.command`.
- **Other tools that write**, `Edit` and `Write`, which can change `.git/config` or a hook script directly.
- **A user typing the command themselves.** This is a guardrail for the agent's shell, not an access control on the repository. Branch protection on the server is the control; this is what stops the accident.

## Command-parsing evasions

A prefix match on the command string is defeated by the first compound command it meets. The script splits on `;`, `|`, `&`, `&&`, `||`, newlines, subshells and command substitution, then strips what can legitimately sit in front of a command — leading environment assignments, `sudo`, `env`, `nice`, `time`, `xargs`, `exec`, and a `bash -c` wrapper — before deciding whether the fragment is git at all. It also handles `git -C <dir>` and `git -c k=v`, an absolute path to the binary, and `\git`.

What it cannot see, and what to say out loud rather than imply:

- **A script file.** `./deploy.sh` may force-push on line 40. The hook sees `./deploy.sh`.
- **An alias or a git alias.** `git ship` expanding to a force push is invisible; the subcommand is `ship`.
- **Anything reconstructed at runtime** — a variable holding the flag, a base64 decode piped to a shell.
- **Text that merely looks like a command.** `git commit -m "wip; git push --force"` is split at the `;` and refused. This is a false positive, and it is the right direction to err in: the fix is to reword the message, which costs seconds.

None of these are worth chasing. This is a guardrail against a slip, not a sandbox against an adversary — an agent that wanted to get around it could edit `settings.json`. Presenting it as more than that is how it ends up trusted for something it does not do.

## Granting a one-off exception

Three ways, in order of preference:

1. **The user runs the command themselves**, in their own terminal. Nothing to configure, nothing to remember to put back, and the destructive act has a human attached to it.
2. **Start the session with the rule excepted:** `GIT_GUARDRAILS_ALLOW=push-protected claude`. Comma-separated for several. Every other rule stays live, and the exception dies with the session.
3. **Comment the rule out of `RULES`** in the installed copy — or, for `extra-deny`, remove the pattern from `EXTRA_DENY`. This is the honest way to make a permanent decision — visible in one place, reversible in one line — and it is much better than the alternative it replaces, which is deleting the hook from `settings.json` at 6pm and never restoring it.

The exception is read from the environment the client was started in, and inline assignments are stripped from the command before analysis. `GIT_GUARDRAILS_ALLOW=push-force git push --force` typed by the model changes nothing: the assignment never reaches the hook's own environment. An exception is a thing the user grants, which is the only way an exception means anything.
