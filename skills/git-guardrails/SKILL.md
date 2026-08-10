---
name: git-guardrails
description: Installs a Claude Code PreToolUse hook that refuses the git commands which destroy work — force pushes, pushes to a protected branch, reset --hard, clean -f, branch -D — while leaving ordinary branch pushes alone. Use when someone says "block dangerous git commands", "stop Claude from force pushing", "add git safety hooks", "it reset --hard my changes", "protect main from the agent", "git guardrails", "prevent destructive git", or after an agent threw away work that was never committed. A guardrail that also blocks ordinary work gets turned off within a day, and a guardrail that is off protects nothing.
---

# Git Guardrails

An agent with shell access can run `git reset --hard` as easily as `git status`, and the difference between them is that one of them is not undoable. Uncommitted work has no reflog. A force-pushed branch takes its old commits with it. `git clean -fdx` deletes the `.env` nobody had a copy of.

This skill installs a `PreToolUse` hook on `Bash` that reads each command before it runs and refuses the ones that destroy work which cannot be recovered from this clone. Everything else passes untouched.

The design constraint is what makes it survive: **it does not block `git push`.** Every team whose work reaches a protected branch through pull requests pushes branches constantly, so a blanket push block is uninstalled the first afternoon someone hits it — and an uninstalled guard is worse than none, because people still believe it is there. What is blocked is the push that rewrites a remote, and the push whose destination is protected.

Security angle: this is the control that keeps an agent's mistake from becoming an incident. Rewriting history in a shared repository is also the standard reflex to a leaked secret, and it is the wrong one — rotation is what closes the exposure, and the rewrite only hides the evidence while breaking every clone. The guard buys the pause in which that gets decided by a person; when the trigger really is a committed credential, that pause hands off to `secrets-and-config-audit`, which rotates before it reports.

## First-run check

Read `.dragon-buddy/config.json`. This skill uses `project.name` for the summary and nothing else, so a missing config is not fatal — say so and offer `buddy-setup`, since the rest of the pack needs it.

What the guard actually needs comes from the repository, not the config. Read the default branch rather than guessing at it:

```sh
git symbolic-ref --quiet --short refs/remotes/origin/HEAD || git rev-parse --abbrev-ref HEAD
```

## Inputs

Ask for these three, and nothing else. Everything else is readable.

- **Scope.** This project (`.claude/settings.json`) or all projects (`~/.claude/settings.json`). Say plainly what global means: the hook then runs on every Bash command in every repository on the machine, including ones where `main` is a scratch branch and a force push costs nothing.
- **The protected branches.** Default `main` and `master`. Add `release/*`, `env/*` or whatever this organisation actually treats as untouchable. Globs work.
- **Anything to leave unblocked.** A solo repository with no remote reviewers may genuinely want `push-protected` off. Turning a rule off deliberately, once, is the behaviour this skill is trying to produce; discovering the whole hook removed a week later is the behaviour it is trying to avoid.

## Workflow

1. **Settle scope and the branch list first**, because both are baked into the copy you are about to make. If the user picks global, repeat the consequence in one sentence and get a yes.

2. **Copy the script to where the hook will run it.**

   ```sh
   mkdir -p .claude/hooks
   cp "${CLAUDE_PLUGIN_ROOT}/scripts/block-dangerous-git.sh" .claude/hooks/block-dangerous-git.sh
   chmod +x .claude/hooks/block-dangerous-git.sh
   ```

   For global scope the destination is `~/.claude/hooks/block-dangerous-git.sh`.

   Copy, never symlink. The plugin lives at a version-keyed path, so a link into it dangles on the next update — and it dangles into a hook, where the failure is silent. Copying also means the branch list you are about to edit is not overwritten by an update.

3. **Edit the configuration block at the top of the copy.** It is the only place anything is configurable: `PROTECTED_BRANCHES`, the `RULES` list (comment a line out to turn one off), and `EXTRA_DENY` for non-git patterns. Set the branch list to what step 1 established. Do not scatter edits further down the file.

4. **Register it, merging into what is already there.** Read the settings file, add one entry to `hooks.PreToolUse`, write it back. Never write the file from scratch — it holds permissions, environment and other hooks, and replacing it silently removes them.

   ```json
   {
     "hooks": {
       "PreToolUse": [
         {
           "matcher": "Bash",
           "hooks": [
             {
               "type": "command",
               "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/block-dangerous-git.sh"
             }
           ]
         }
       ]
     }
   }
   ```

   The matcher matches the **tool name**, and only `Bash` carries a shell command. A git operation run through an MCP server arrives as `mcp__<server>__<tool>` and this matcher never sees it — this pack has been bitten by that exact naming before. `references/guardrail-patterns.md` covers what that leaves uncovered.

5. **Verify by observing it refuse, and observing it pass.** Two commands, both run, both reported:

   ```sh
   echo '{"tool_input":{"command":"git push --force origin main"}}' | .claude/hooks/block-dangerous-git.sh; echo "exit=$?"   # must be 2
   echo '{"tool_input":{"command":"git push origin my-branch"}}'    | .claude/hooks/block-dangerous-git.sh; echo "exit=$?"   # must be 0
   ```

   An install that has not been seen refusing something has not been tested. A hook that fails open — wrong path, missing `chmod +x`, a settings file that was written but never re-read — produces no error anywhere; it just stops blocking, and what remains is a belief that destructive git is impossible here. That belief is more dangerous than knowing there is no guard, because it is acted on. The second command matters as much as the first: a guard that refuses everything is discovered and removed just as fast.

   Hook registration is read at session start. The script is live the moment it is copied, but a new `settings.json` entry does not take effect until the session restarts — say so, or the user's own test appears to prove the install failed.

6. **Say what is now blocked, in one list**, and how to get past it once: the user runs the command themselves in their own terminal, or starts the session with `GIT_GUARDRAILS_ALLOW=<rule>`. The exception is deliberately not reachable from the command line — inline environment assignments are stripped before the command is analysed, so the model cannot grant itself one by prefixing a variable.

## Output format

```markdown
## Git guardrails installed — [project]

**Scope:** this project (.claude/settings.json) | all projects (~/.claude/settings.json — every repo on this machine)
**Script:** [path to the copy]
**Protected branches:** [list]
**Rules on:** [list] — **off:** [any turned off, with the reason]

**Verified:**
- `git push --force origin main` → exit 2, refused: [the rule name it cited]
- `git push origin my-branch` → exit 0, allowed

**Takes effect:** [now / after a session restart, because the registration is read at start]
**One-off exception:** run it yourself, or start the session with GIT_GUARDRAILS_ALLOW=<rule>
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Installed the git guardrail hook at <scope>, protecting <branches>, verified refusing a force push and passing a branch push."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:git-guardrails"]`

Relay the reaction verbatim.

## File output

Writes the script copy under `.claude/hooks/` (or the user's own `~/.claude/hooks/`), and merges one entry into `hooks.PreToolUse` in the settings file for the chosen scope.

Touches nothing else. It never runs a git command that changes state, and it never edits the repository's history, tracked files or `.dragon-buddy/config.json`.

## Reference library

Load these for depth when the task calls for it:
- `references/guardrail-patterns.md` — every rule with the reason it destroys work, what is deliberately left unblocked and why, `PreToolUse` exit-code semantics and the matcher pitfall, the command-parsing evasions the script handles and the ones it cannot, and how to grant a one-off exception without disabling the guard.

## Worked example

`examples/git-guardrails-run.md` is the quality target: an install narrowed to the branches that matter, a verification that observes both a refusal and a pass, and a later command the guard blocked that the user genuinely needed — handled without turning anything off.

## Quality bar

A good run satisfies all of these:

- Scope was asked, not assumed, and the global consequence was stated before it was chosen.
- The protected branch list was read from the repository and confirmed, not guessed.
- The script was copied, not symlinked, and made executable.
- The settings file was merged into, and the other hooks and settings that were in it are still in it.
- The guard was **observed refusing** a destructive command with exit 2, and **observed passing** an ordinary branch push with exit 0. Both outputs were shown, not summarised.
- Ordinary work still runs: pushing a feature branch, committing, amending, fetching, and rebasing a feature branch onto the default branch were all left alone.
- The user was told the registration is read at session start, so a test before restarting proves nothing.
- The one-off exception was explained, and it is not one the agent can grant itself.
