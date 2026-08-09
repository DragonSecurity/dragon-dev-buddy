#!/usr/bin/env sh
# Refuse to commit project memories.
#
# Memories are notes to yourself about a codebase: why a constraint exists, what
# was tried and rejected, which gotcha cost an afternoon. They are written to be
# useful, not to be read by strangers, and they routinely pick up paths, host
# names, ticket numbers and the occasional client. `.gitignore` is what keeps
# them local; this is the backstop for the two ways past it — someone reaching
# for `git add -f`, and a repository whose ignore rule was never added or has
# since drifted.
#
# Deliberately not a secret scan. The rule here is not "no credentials in this
# file", which is a judgement call with false negatives; it is "this directory
# does not get committed", which is a fact. A memory worth publishing is a
# memory that belongs somewhere else in the tree — move it, do not exempt it.
#
# Install:
#   ln -s ../../scripts/pre-commit-memory-guard.sh .git/hooks/pre-commit
# or, if you already have a pre-commit hook, call this from it.
set -eu

guarded='.dragon-buddy/memories/'

# `git diff --cached` needs something to diff against, and on the very first
# commit in a repository HEAD does not exist yet — it fails, and a `|| true`
# swallowing that turns the guard into a no-op for exactly the commit most
# likely to be made in a hurry. Against the empty tree instead, so a root commit
# is compared with "nothing" rather than with a ref that is not there.
if git rev-parse --verify -q HEAD >/dev/null 2>&1; then
	against=HEAD
else
	against="$(git hash-object -t tree /dev/null)"
fi

# Only what is staged for this commit. --diff-filter excludes deletions, since
# removing a memory that was committed by mistake is exactly what someone
# cleaning up needs to be able to do.
staged="$(git diff --cached --name-only --diff-filter=ACMR "$against" -- "$guarded")"

if [ -z "$staged" ]; then
	exit 0
fi

echo "pre-commit: refusing to commit project memories." >&2
echo >&2
echo "$staged" | sed 's/^/  /' >&2
echo >&2
echo "Memories under $guarded are local to your machine by design." >&2
echo "They are working notes, and they collect paths, hosts and names that" >&2
echo "read very differently in a public repository than in your editor." >&2
echo >&2
echo "If this repository is missing the ignore rule, add it:" >&2
echo "  echo '$guarded' >> .gitignore" >&2
echo >&2
echo "If you genuinely want this content in the repository, move it out of" >&2
echo "that directory first, so the decision is visible in the diff." >&2

exit 1
