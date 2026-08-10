#!/usr/bin/env bash
# Refuse destructive git commands before Claude Code runs them.
#
# Installed as a Claude Code PreToolUse hook on Bash. It reads the hook payload
# on stdin, looks at tool_input.command, and exits 2 with an explanation on
# stderr to refuse the call, or 0 to let it through. Exit 2 is the only code the
# client treats as a block; anything else runs the command.
#
# The rule for what belongs here is narrow on purpose: an operation is blocked
# when it destroys work that cannot be recovered from this clone. A guardrail
# that also blocks ordinary work gets removed within a day, and a removed
# guardrail protects nothing -- which is why `git push` is not blocked outright.
# Teams whose work reaches a protected branch through pull requests push
# branches all day long. What is blocked is a push that rewrites history, and a
# push whose destination is a protected branch.
#
# Everything you are likely to want to change -- the protected branch list, the
# rules that are on, extra patterns -- is in the EDIT HERE block below, and
# nowhere else.
set -u

# ---------------------------------------------------------------------------
# EDIT HERE -- this block is the whole configuration surface.
# ---------------------------------------------------------------------------

# Branches whose history is not yours to move. Shell globs are allowed, so
# 'release/*' and 'env/*' work. Anything reaching one of these goes via a pull
# request.
PROTECTED_BRANCHES=(main master)

# Rules that are on. Comment a line out to turn that rule off; the guard keeps
# running for everything else. Each name is explained in the skill's
# references/guardrail-patterns.md.
RULES=(
	push-force          # --force, --force-with-lease, -f, a +refspec: rewrites a remote
	push-protected      # any push whose destination is a protected branch
	push-mirror         # --mirror / --prune: deletes remote refs you do not have
	reset-hard          # git reset --hard: discards the working tree and the index
	clean-force         # git clean -f / -fd / -fdx: deletes untracked files outright
	branch-delete-force # git branch -D (and -M onto a protected name)
	checkout-discard    # git checkout . / -- . / -f: discards uncommitted work
	restore-discard     # git restore . : same, newer spelling
	stash-destroy       # git stash drop / clear: the stash has no reflog
	rebase-protected    # a rebase that rewrites a protected branch
	history-rewrite     # filter-branch / filter-repo: rewrites every commit
	reflog-destroy      # reflog expire/delete, gc --prune=now: removes the undo path
)

# Extra refusals, matched as extended regexes against the whole command line.
# Use these for things that are not git subcommands. The default entry catches
# `rm -rf` aimed at a .git directory -- the repository, reflog and all -- and is
# written not to fire on .github.
EXTRA_DENY=(
	'(^|[[:space:]])rm[[:space:]]+(-[A-Za-z]+[[:space:]]+)*-[A-Za-z]*[rR][A-Za-z]*[[:space:]]+[^[:space:]]*\.git([[:space:]]|/|$)'
)

# ---------------------------------------------------------------------------
# END EDIT -- below here is the parser.
# ---------------------------------------------------------------------------

# A one-off exception is granted from the environment the client was started in,
# never from the command line: inline assignments are stripped before analysis,
# so a `GIT_GUARDRAILS_ALLOW=... git push --force` typed by the model changes
# nothing. Comma-separated rule names, e.g. GIT_GUARDRAILS_ALLOW=push-protected.
ALLOW=",${GIT_GUARDRAILS_ALLOW:-},"

CURRENT_BRANCH=""
BRANCH_RESOLVED=0

rule_on() {
	local want=$1 r
	for r in ${RULES[@]+"${RULES[@]}"}; do
		[ "$r" = "$want" ] && return 0
	done
	return 1
}

# A rule fires only if it is in RULES and has not been excepted for this session.
active() {
	rule_on "$1" || return 1
	case "$ALLOW" in
	*",$1,"*) return 1 ;;
	esac
	return 0
}

current_branch() {
	if [ "$BRANCH_RESOLVED" -eq 0 ]; then
		BRANCH_RESOLVED=1
		CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)
		# Detached HEAD reports "HEAD"; there is no branch to be protective of.
		[ "$CURRENT_BRANCH" = "HEAD" ] && CURRENT_BRANCH=""
	fi
	printf '%s' "$CURRENT_BRANCH"
}

is_protected() {
	local ref=${1:-} p
	ref=${ref#+}
	ref=${ref#refs/heads/}
	[ -n "$ref" ] || return 1
	for p in ${PROTECTED_BRANCHES[@]+"${PROTECTED_BRANCHES[@]}"}; do
		# shellcheck disable=SC2053 -- the right side is a pattern on purpose.
		[[ $ref == $p ]] && return 0
	done
	return 1
}

refuse() {
	local rule=$1 why=$2 instead=$3
	{
		echo "BLOCKED by git-guardrails [$rule]"
		echo
		echo "  $COMMAND"
		echo
		echo "$why"
		echo
		echo "Do this instead: $instead"
		echo
		echo "You do not have authority to override this. If the command is genuinely"
		echo "needed, say so and let the user run it themselves, or ask them to start"
		echo "the session with GIT_GUARDRAILS_ALLOW=$rule."
	} >&2
	exit 2
}

# --- reading the payload ---------------------------------------------------
#
# Fails closed. No JSON parser, or a payload that will not parse, means the
# guard cannot see what it is being asked to allow, and a guard that waves
# commands through in that state is worse than no guard: it leaves a belief
# where a control used to be. Refusing is loud, and the message says how to fix
# it in one line.

read_command() {
	if command -v jq >/dev/null 2>&1; then
		jq -r '.tool_input.command // ""' 2>/dev/null
	elif command -v python3 >/dev/null 2>&1; then
		python3 -c 'import json,sys
try:
    d = json.load(sys.stdin)
except Exception:
    sys.exit(3)
print(((d.get("tool_input") or {}).get("command") or ""))' 2>/dev/null
	else
		return 127
	fi
}

PAYLOAD=$(cat)
if COMMAND=$(printf '%s' "$PAYLOAD" | read_command); then
	:
else
	{
		echo "BLOCKED by git-guardrails [install]"
		echo
		echo "The hook could not read its input: neither jq nor python3 is on PATH,"
		echo "or the payload was not the JSON a PreToolUse hook is given."
		echo
		echo "It refuses rather than passes, because a guard that cannot read the"
		echo "command it is guarding has stopped being a guard. Install jq, or"
		echo "remove the hook from settings.json if you no longer want it."
	} >&2
	exit 2
fi

[ -n "$COMMAND" ] || exit 0

# --- extra patterns --------------------------------------------------------

case "$ALLOW" in
*",extra-deny,"*) ;;
*)
	for pat in ${EXTRA_DENY[@]+"${EXTRA_DENY[@]}"}; do
		if printf '%s' "$COMMAND" | grep -qE "$pat"; then
			refuse "extra-deny" \
				"The command matches a deny pattern configured in the guard, and it destroys a repository rather than a change inside one." \
				"leave it to the user, or edit EXTRA_DENY in the hook script."
		fi
	done
	;;
esac

# --- splitting a command line into candidate git invocations ---------------
#
# A prefix match on the raw string misses `a && git push --force`, `sudo git
# ...`, `FOO=1 git ...` and `bash -c 'git ...'`. Every shell separator becomes a
# newline, then each fragment is stripped of the wrappers that can sit in front
# of a command, and only fragments that actually resolve to git are analysed.

SEPARATORS=$';|&(){}`\n'
NEWLINES=$'\n\n\n\n\n\n\n\n\n'

strip_quotes() {
	local s=${1:-}
	s=${s#[\"\']}
	s=${s%[\"\']}
	s=${s#\\}
	printf '%s' "$s"
}

analyze_push() {
	local force=0 mirror=0 skipnext=0 a d spec
	local -a pos=()
	for a in ${ARGS[@]+"${ARGS[@]}"}; do
		if [ "$skipnext" -eq 1 ]; then
			skipnext=0
			continue
		fi
		case "$a" in
		--force | --force-with-lease* | --force-if-includes) force=1 ;;
		--mirror | --prune) mirror=1 ;;
		-o | --push-option | --receive-pack | --exec | --repo) skipnext=1 ;;
		--*) ;;
		-[A-Za-z]*)
			case "$a" in
			*f*) force=1 ;;
			esac
			;;
		*) pos[${#pos[@]}]=$a ;;
		esac
	done

	if [ "$force" -eq 1 ] && active push-force; then
		refuse "push-force" \
			"A force push replaces history on the remote. Commits that only existed there are gone, and every clone that had them is now wrong." \
			"rebase or amend locally and push a new branch, or open a pull request."
	fi
	if [ "$mirror" -eq 1 ] && active push-mirror; then
		refuse "push-mirror" \
			"--mirror and --prune delete remote refs that are absent locally. Branches nobody has fetched disappear." \
			"push the specific refs you mean by name."
	fi

	# pos[0] is the remote; everything after it is a refspec.
	local i=1
	local -a specs=()
	while [ "$i" -lt "${#pos[@]}" ]; do
		specs[${#specs[@]}]=${pos[$i]}
		i=$((i + 1))
	done

	if [ "${#specs[@]}" -eq 0 ]; then
		specs=("$(current_branch)")
	fi

	for spec in ${specs[@]+"${specs[@]}"}; do
		case "$spec" in
		+*)
			if active push-force; then
				refuse "push-force" \
					"A leading + on a refspec is a force push in another spelling. It rewrites the remote ref." \
					"push without the +, or open a pull request."
			fi
			;;
		esac
		d=${spec##*:}
		d=$(strip_quotes "$d")
		[ "$d" = "HEAD" ] && d=$(current_branch)
		if is_protected "$d" && active push-protected; then
			refuse "push-protected" \
				"'$d' is a protected branch. Nothing reaches it by push -- it takes a pull request, which is where review and the required checks happen." \
				"push a branch and open a pull request against $d."
		fi
	done
}

analyze_rebase() {
	local a skipnext=0
	local -a pos=()
	for a in ${ARGS[@]+"${ARGS[@]}"}; do
		if [ "$skipnext" -eq 1 ]; then
			skipnext=0
			continue
		fi
		case "$a" in
		--abort | --continue | --skip | --quit | --edit-todo | --show-current-patch)
			return 0
			;;
		--onto | -x | --exec | -s | --strategy | -X | --strategy-option) skipnext=1 ;;
		-*) ;;
		*) pos[${#pos[@]}]=$a ;;
		esac
	done

	# `git rebase <upstream>` rewrites the current branch; `git rebase <upstream>
	# <branch>` rewrites <branch>. Rebasing a feature branch onto main is normal
	# work and stays allowed -- what is blocked is a rebase that rewrites a
	# protected branch.
	local target
	if [ "${#pos[@]}" -ge 2 ]; then
		target=${pos[1]}
	else
		target=$(current_branch)
	fi

	if is_protected "$target" && active rebase-protected; then
		refuse "rebase-protected" \
			"This rewrites '$target', a protected branch. Every commit gets a new hash and everyone else's clone diverges from it." \
			"rebase your own branch, or merge $target into it."
	fi
}

analyze_git() {
	local sub=$1 a
	case "$sub" in
	push) analyze_push ;;
	rebase) analyze_rebase ;;
	reset)
		for a in ${ARGS[@]+"${ARGS[@]}"}; do
			if [ "$a" = "--hard" ] && active reset-hard; then
				refuse "reset-hard" \
					"--hard discards every uncommitted change in the working tree and the index. Nothing that was never committed is recoverable." \
					"git stash, or commit on a scratch branch first."
			fi
		done
		;;
	clean)
		local forced=0 dry=0
		for a in ${ARGS[@]+"${ARGS[@]}"}; do
			case "$a" in
			--force) forced=1 ;;
			--dry-run) dry=1 ;;
			-[A-Za-z]*)
				case "$a" in
				*f*) forced=1 ;;
				esac
				case "$a" in
				*n*) dry=1 ;;
				esac
				;;
			esac
		done
		if [ "$forced" -eq 1 ] && [ "$dry" -eq 0 ] && active clean-force; then
			refuse "clean-force" \
				"git clean deletes untracked files from disk. They were never in git, so there is no reflog and no undo -- and with -x that includes .env files and local config." \
				"run git clean -nd first and delete what it lists, deliberately."
		fi
		;;
	branch)
		local hard=0 delete=0 force=0 forcemove=0 hitsprotected=0
		for a in ${ARGS[@]+"${ARGS[@]}"}; do
			case "$a" in
			--delete) delete=1 ;;
			--force) force=1 ;;
			--move) ;;
			--*) ;;
			-[A-Za-z]*)
				case "$a" in
				*D*) hard=1 ;;
				esac
				case "$a" in
				*M*) forcemove=1 ;;
				esac
				case "$a" in
				*f*) force=1 ;;
				esac
				case "$a" in
				*d*) delete=1 ;;
				esac
				;;
			*)
				is_protected "$a" && hitsprotected=1
				;;
			esac
		done
		if { [ "$hard" -eq 1 ] || { [ "$delete" -eq 1 ] && [ "$force" -eq 1 ]; }; } && active branch-delete-force; then
			refuse "branch-delete-force" \
				"-D deletes a branch whether or not it is merged. Commits reachable only from it survive in the reflog for a while and then get collected." \
				"use git branch -d, which refuses unmerged work, and merge or push the branch first if it matters."
		fi
		if [ "$forcemove" -eq 1 ] && [ "$hitsprotected" -eq 1 ] && active branch-delete-force; then
			refuse "branch-delete-force" \
				"A forced branch move onto a protected name overwrites whatever that name pointed at." \
				"pick a name that is not already in use."
		fi
		;;
	checkout)
		for a in ${ARGS[@]+"${ARGS[@]}"}; do
			case "$a" in
			--force)
				if active checkout-discard; then
					refuse "checkout-discard" \
						"checkout --force throws away uncommitted changes in the working tree to make the switch succeed." \
						"commit or stash first, then switch."
				fi
				;;
			--*) ;;
			-[A-Za-z]*)
				case "$a" in
				*f*)
					if active checkout-discard; then
						refuse "checkout-discard" \
							"checkout -f throws away uncommitted changes in the working tree to make the switch succeed." \
							"commit or stash first, then switch."
					fi
					;;
				esac
				;;
			. | ./ | '*' | ':/')
				if active checkout-discard; then
					refuse "checkout-discard" \
						"Checking out '.' restores every tracked file from the index, discarding uncommitted edits across the whole tree." \
						"name the specific files, after reading the diff."
				fi
				;;
			esac
		done
		;;
	restore)
		for a in ${ARGS[@]+"${ARGS[@]}"}; do
			case "$a" in
			. | ./ | '*' | ':/')
				if active restore-discard; then
					refuse "restore-discard" \
						"Restoring '.' overwrites every modified tracked file with the version in the index or a commit. Uncommitted work is gone." \
						"name the specific files, after reading the diff."
				fi
				;;
			esac
		done
		;;
	stash)
		local first=${ARGS[0]:-}
		case "$first" in
		drop | clear)
			if active stash-destroy; then
				refuse "stash-destroy" \
					"The stash has no reflog you can rely on. A dropped or cleared entry is not listed anywhere afterwards." \
					"git stash list and inspect first, or git stash pop, which only drops after a successful apply."
			fi
			;;
		esac
		;;
	filter-branch | filter-repo)
		if active history-rewrite; then
			refuse "history-rewrite" \
				"This rewrites every commit in the repository. Every existing clone, tag, open pull request and CI reference is invalidated." \
				"if this is a leaked-secret cleanup, rotate the credential first -- that is the part that actually matters -- and plan the rewrite with the whole team."
		fi
		;;
	reflog)
		local first=${ARGS[0]:-}
		case "$first" in
		expire | delete)
			if active reflog-destroy; then
				refuse "reflog-destroy" \
					"The reflog is the only way back from a bad reset, rebase or branch delete. Expiring it removes the undo path for everything above." \
					"leave it alone; it expires on its own schedule."
			fi
			;;
		esac
		;;
	gc)
		for a in ${ARGS[@]+"${ARGS[@]}"}; do
			case "$a" in
			--prune=now | --prune=all)
				if active reflog-destroy; then
					refuse "reflog-destroy" \
						"Pruning now collects the unreachable objects that a reset or a rebase left behind -- the ones recovery depends on." \
						"let gc run with its default two-week grace period."
				fi
				;;
			esac
		done
		;;
	esac
	return 0
}

analyze_segment() {
	local seg=$1
	local -a toks=()
	read -r -a toks <<<"$seg"
	local n=${#toks[@]}
	[ "$n" -eq 0 ] && return 0

	local i=0 t base found=0
	while [ "$i" -lt "$n" ]; do
		t=$(strip_quotes "${toks[$i]}")
		if [ -z "$t" ]; then
			i=$((i + 1))
			continue
		fi
		# A leading environment assignment belongs to the wrapper, not the command.
		case "$t" in
		*=*)
			if printf '%s' "${t%%=*}" | grep -qE '^[A-Za-z_][A-Za-z0-9_]*$'; then
				i=$((i + 1))
				continue
			fi
			;;
		esac
		base=${t##*/}
		case "$base" in
		git | git.exe)
			found=1
			break
			;;
		sudo | doas | env | nice | ionice | time | command | builtin | exec | xargs | nohup | stdbuf | sh | bash | zsh | dash | ksh | if | then | else | do | while | until | '!')
			i=$((i + 1))
			continue
			;;
		-*)
			i=$((i + 1))
			continue
			;;
		*) return 0 ;;
		esac
	done
	[ "$found" -eq 1 ] || return 0

	# git's own options sit between `git` and the subcommand.
	i=$((i + 1))
	while [ "$i" -lt "$n" ]; do
		t=$(strip_quotes "${toks[$i]}")
		case "$t" in
		-C | -c | --git-dir | --work-tree | --namespace | --exec-path | --super-prefix)
			i=$((i + 2))
			;;
		--git-dir=* | --work-tree=* | --namespace=* | --exec-path=* | -C* | -c*)
			i=$((i + 1))
			;;
		-P | --no-pager | --paginate | --bare | --no-replace-objects | --literal-pathspecs | --no-optional-locks)
			i=$((i + 1))
			;;
		*) break ;;
		esac
	done
	[ "$i" -lt "$n" ] || return 0

	local sub
	sub=$(strip_quotes "${toks[$i]}")
	ARGS=()
	i=$((i + 1))
	while [ "$i" -lt "$n" ]; do
		ARGS[${#ARGS[@]}]=$(strip_quotes "${toks[$i]}")
		i=$((i + 1))
	done

	analyze_git "$sub"
}

while IFS= read -r fragment; do
	[ -n "$fragment" ] || continue
	analyze_segment "$fragment"
done <<EOF
$(printf '%s' "$COMMAND" | tr "$SEPARATORS" "$NEWLINES")
EOF

exit 0
