# Worked example: installing the guard after a force push ate someone's commit

A session yesterday rebased a shared branch and force-pushed it, taking a colleague's commit with it. The user says: *"put something in place so that can't happen again — but I push branches fifty times a day, don't break that."*

---

**Scope and branches, before touching anything.**

> Two decisions first, because both get baked into the copy I install.
>
> **Scope.** Project (`.claude/settings.json`) or all projects (`~/.claude/settings.json`)? Global means this runs on every Bash command in every repository on this machine — including throwaway ones where `main` is a scratch branch and a force push costs nothing. Project scope is the safer default and it is what I would pick unless you want this everywhere.

**User:** This project.

> **Protected branches.** Reading what this repository actually treats as default rather than assuming:

```sh
$ git symbolic-ref --quiet --short refs/remotes/origin/HEAD
origin/main
$ git branch -r | sed -n 's|  origin/||p' | grep -E '^(release|env)/' | head
release/2026.7
release/2026.8
```

> `main` plus a `release/*` line that is clearly live. Proposing `PROTECTED_BRANCHES=(main master release/*)` — `master` costs nothing to leave in and covers repositories you clone later if you ever widen the scope.

**User:** Yes, and we don't have anything else protected.

---

**Install.**

```sh
$ mkdir -p .claude/hooks
$ cp "${CLAUDE_PLUGIN_ROOT}/scripts/block-dangerous-git.sh" .claude/hooks/block-dangerous-git.sh
$ chmod +x .claude/hooks/block-dangerous-git.sh
```

> Copied rather than symlinked: the plugin sits at a version-keyed path, so a link into it dangles on the next update — and it would dangle inside a hook, where nothing announces the breakage. The copy also means the branch list I am about to edit survives updates.

Edited the one configuration block at the top of the copy:

```bash
PROTECTED_BRANCHES=(main master release/*)
```

`RULES` left as shipped. `EXTRA_DENY` left with its single default entry.

**Registration.** `.claude/settings.json` already had a `permissions` block and a `PostToolUse` hook, so this is a merge, not a write:

```json
{
  "permissions": { "allow": ["Bash(npm test:*)"] },
  "hooks": {
    "PostToolUse": [ { "matcher": "Edit|Write", "hooks": [ { "type": "command", "command": "npm run -s fmt:staged" } ] } ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/block-dangerous-git.sh" }
        ]
      }
    ]
  }
}
```

> The existing formatter hook and the permissions block are still there. Writing this file from scratch would have removed both, and nothing would have complained.

---

**Verify — observed refusing, and observed passing.**

```sh
$ echo '{"tool_input":{"command":"git push --force origin main"}}' | .claude/hooks/block-dangerous-git.sh; echo "exit=$?"
BLOCKED by git-guardrails [push-force]

  git push --force origin main

A force push replaces history on the remote. Commits that only existed there are
gone, and every clone that had them is now wrong.

Do this instead: rebase or amend locally and push a new branch, or open a pull request.
exit=2
```

```sh
$ echo '{"tool_input":{"command":"git push origin fix/login-timeout"}}' | .claude/hooks/block-dangerous-git.sh; echo "exit=$?"
exit=0
```

> Both halves matter. The first proves the guard is live; without it the install is a belief, and a hook that fails open — wrong path, missing `chmod +x`, a typo that exits 1 instead of 2 — looks identical to a working one from the outside. Exit codes other than 2 are treated as a hook *error* and the command still runs, so "no error appeared" is not evidence.
>
> The second proves it did not cost you your day. Fifty branch pushes a day still go through untouched.

Spot-checked the rest of the working day while I was here:

```sh
$ for c in 'git commit --amend --no-edit' 'git rebase main' 'git stash pop' 'git clean -nd' 'git checkout src/app.ts'; do
    printf '{"tool_input":{"command":"%s"}}' "$c" | .claude/hooks/block-dangerous-git.sh >/dev/null 2>&1
    echo "$? $c"
  done
0 git commit --amend --no-edit
0 git rebase main
0 git stash pop
0 git clean -nd
0 git checkout src/app.ts
```

> `git rebase main` from a feature branch stays allowed on purpose — it rewrites your branch, not `main`. What is blocked is a rebase that rewrites a protected branch, which is a different command that happens to share a verb.

---

**Result**

- **Scope:** this project, `.claude/settings.json`
- **Protected:** `main`, `master`, `release/*`
- **Blocked:** force pushes anywhere, pushes to a protected branch, `--mirror`/`--prune`, `reset --hard`, `clean -f`, `branch -D`, `checkout .` / `restore .` / `checkout -f`, `stash drop`/`clear`, a rebase of a protected branch, `filter-branch`, reflog expiry and `gc --prune=now`.
- **Takes effect:** the script is live now; the registration is read at session start, so restart before concluding it did not work.
- **One-off exception:** run the command yourself, or start the session with `GIT_GUARDRAILS_ALLOW=<rule>`. The model cannot grant itself one — inline environment assignments are stripped before the command is analysed.

---

## Two days later: a block that was correct and still in the way

**User:** it's refusing `git branch -D fix/login-timeout` and that branch is already merged.

```sh
$ printf '{"tool_input":{"command":"git branch -D fix/login-timeout"}}' | .claude/hooks/block-dangerous-git.sh
BLOCKED by git-guardrails [branch-delete-force]

  git branch -D fix/login-timeout

-D deletes a branch whether or not it is merged. Commits reachable only from it
survive in the reflog for a while and then get collected.

Do this instead: use git branch -d, which refuses unmerged work, and merge or
push the branch first if it matters.
```

> The block is right and the branch really is merged — both things are true. This repository squash-merges, so the branch's commits are not ancestors of `main` and `-d` will keep refusing on ancestry grounds even though the *content* landed. That is exactly the case where reaching for `-D` becomes a habit, and the habit is what eventually deletes something that had not landed.
>
> Checking the content is actually in before doing anything:

```sh
$ git fetch -q origin && git cherry -v origin/main fix/login-timeout | grep '^+' | wc -l
       0
```

> Zero commits on the branch that are not equivalent to something on `main`. It is safe to delete. Three ways to finish, none of which involve turning the guard off:
>
> 1. You run `git branch -D fix/login-timeout` yourself — my preference, it takes four seconds and the destructive act has a person attached.
> 2. `git push origin --delete fix/login-timeout` for the remote copy, which is not blocked; the deletion that needed forcing is only the local ref.
> 3. If you are clearing out thirty of these, start the session with `GIT_GUARDRAILS_ALLOW=branch-delete-force` and every other rule stays live for the duration.
>
> What I am not doing is commenting the rule out of `RULES` for one branch. That is the edit nobody reverses.

**User:** I'll just run it.

---

## What this run got right

- Asked scope and stated what global would mean before the user chose, rather than after.
- Read the protected branches out of the repository, and found a `release/*` line the user had not mentioned.
- Copied rather than symlinked, and said why the difference matters in a hook specifically.
- Merged into a settings file that already had two other things in it, and named what would have been lost by writing it fresh.
- **Observed the guard refusing and observed it passing**, and explained why the passing half is not a formality — a guard that blocks branch pushes gets uninstalled, and a guard that fails open leaves a belief behind.
- Flagged that hook registration is read at session start, so the user's own test would otherwise appear to prove a failed install.
- Handled a correct-but-inconvenient block by verifying the underlying fact (`git cherry`) and routing around it with an exception the user grants, instead of editing the rule list under pressure.
