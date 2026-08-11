#!/usr/bin/env bash
# Exercise scripts/block-dangerous-git.sh against the commands it claims to
# judge.
#
# The guard is a hook, so the only honest way to check it is to feed it the JSON
# a hook is fed and read the exit code the client acts on: 2 refuses the command,
# anything else runs it. Reading the parser proves nothing -- every rule in it was
# individually correct on the day `git push --tags` was refused, because the bug
# was in a fallback three functions away from the rule that fired.
#
# Both directions are tested and both matter. A guard that stops refusing is the
# obvious failure. A guard that starts refusing ordinary work is the one that
# gets uninstalled, and an uninstalled guard protects nothing -- which is why the
# allow table here is longer than the block table.
#
# Several rules depend on the branch the caller is standing on, so the cases run
# inside a throwaway repository whose HEAD this script controls. Running them in
# the pack's own checkout would make the result depend on whoever last ran
# `git checkout`.
set -u

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
guard="$root/scripts/block-dangerous-git.sh"

if [ ! -x "$guard" ]; then
	echo "FAIL: $guard is not executable; the hook would fail open" >&2
	exit 1
fi

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

git init -q "$scratch"
cd "$scratch"
git config user.email guard@example.invalid
git config user.name "Guard Check"
git commit -q --allow-empty -m "root"
git branch -M main
git branch feature

failures=0
checks=0

# check <expected-exit> <branch-to-stand-on> <command>
check() {
	local want=$1 branch=$2 cmd=$3 got
	git checkout -q "$branch"
	printf '{"tool_input":{"command":%s}}' "$(printf '%s' "$cmd" | sed 's/\\/\\\\/g; s/"/\\"/g; s/^/"/; s/$/"/')" |
		"$guard" >/dev/null 2>&1
	got=$?
	checks=$((checks + 1))
	if [ "$got" -ne "$want" ]; then
		failures=$((failures + 1))
		printf 'FAIL  on %-8s want exit %s, got %s: %s\n' "$branch" "$want" "$got" "$cmd" >&2
	fi
}

blocked() { check 2 "$1" "$2"; }
allowed() { check 0 "$1" "$2"; }

# --- Refused: destroys work that no clone can recover -----------------------
blocked feature "git push --force origin main"
blocked feature "git push -f origin main"
blocked feature "git push --force-with-lease origin feature"
blocked feature "git push origin +main"
blocked feature "git push --mirror origin"
blocked feature "git push origin main"
blocked main "git push"
blocked feature "git reset --hard HEAD~1"
blocked feature "git clean -fd"
blocked feature "git clean -fdx"
blocked feature "git branch -D feature"
blocked feature "git checkout ."
blocked feature "git restore ."
blocked feature "git stash drop"
blocked feature "git stash clear"
blocked feature "git filter-branch --all"
blocked feature "git reflog expire --expire=now --all"
blocked feature "rm -rf .git"
blocked main "git rebase origin/main"

# Evasions: the destructive command is not the first word.
blocked feature "echo hi && git push --force origin main"
blocked feature "ls; git reset --hard"
blocked feature "sudo git clean -fdx"
blocked feature "GIT_GUARDRAILS_ALLOW=reset-hard git reset --hard"

# --- Allowed: ordinary work, and the release path ---------------------------
allowed feature "git push origin feature"
allowed feature "git push -u origin feature"
allowed feature "git status"
allowed feature "git commit -s -m wip"
allowed feature "git commit --amend"
allowed feature "git fetch origin --prune"
allowed feature "git pull --ff-only"
allowed feature "git checkout main"
allowed feature "git switch -c another"
allowed feature "git rebase main"
allowed feature "git rebase origin/main"
allowed feature "git rebase -i HEAD~3"
allowed feature "git stash"
allowed feature "git stash pop"
allowed feature "git branch -d merged"
allowed feature "git tag -a v1.5.0 -m release"
allowed feature "npm run build"
allowed feature "rm -rf .github/workflows/tmp"

# The release path runs from main, which is what made the tag cases regress:
# with no refspec the parser fell back to the current branch, so `git push
# --tags` standing on main was read as a push *of* main. --tags pushes tags and
# no branch at all, so there is nothing protected about it.
allowed main "git push --tags"
allowed main "git push origin --tags"
allowed main "git push origin v1.4.0"
allowed main "git checkout main"
allowed main "git pull --ff-only"

# The counterpart that keeps the fix narrow: --follow-tags *does* push the
# current branch alongside the tags, so from main it is still a push to main.
blocked main "git push --follow-tags"
# And an explicit refspec is still read, whatever else is on the line.
blocked main "git push origin main --tags"

if [ "$failures" -ne 0 ]; then
	echo "FAILED: $failures of $checks guard cases" >&2
	exit 1
fi
echo "ok: $checks guard cases behave"
