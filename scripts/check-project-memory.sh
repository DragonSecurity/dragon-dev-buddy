#!/usr/bin/env bash
# Exercise hooks/project-memory.mjs against a throwaway project.
#
# The hook's promise is about cost: whatever a repo accumulates, a session start
# pays at most the budget for it. That is a claim about the output's size across
# three regimes -- small enough to quote, too big to quote, too big even to list
# -- so each is built and measured rather than reasoned about.
set -u

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
hook="$root/hooks/project-memory.mjs"

project="$(mktemp -d)"
trap 'rm -rf "$project"' EXIT
mem="$project/.dragon-buddy/memories"
mkdir -p "$mem"
unset DRAGON_BUDDY_HOOKS DRAGON_BUDDY_DISABLED_HOOKS DRAGON_BUDDY_MEMORY_MAX_CHARS
fails=0

run() { printf '{"hook_event_name":"SessionStart","cwd":"%s"}' "$project" | node "$hook"; }
context() { node -e 'let s="";process.stdin.on("data",c=>s+=c).on("end",()=>{console.log(s.trim()?JSON.parse(s).hookSpecificOutput.additionalContext:"")})'; }
ok() {
	if [ "$2" = "$3" ]; then
		echo "ok   $1"
	else
		echo "FAIL $1: expected [$3], got [$2]"
		fails=$((fails + 1))
	fi
}
memory() { # name body [age-in-minutes]
	printf -- '---\nname: %s\ndescription: What %s taught us about this codebase, in one line.\n---\n\n%s\n' "$1" "$1" "$2" >"$mem/$1.md"
	touch -t "$(date -v-"${3:-0}"M +%Y%m%d%H%M.%S 2>/dev/null || date -d "-${3:-0} min" +%Y%m%d%H%M.%S)" "$mem/$1.md"
}

ok "no memories, no context" "$(run)" ""

# Small: quoted in full.
memory alpha "The alpha body."
ok "a small set is quoted in full" "$(run | context | grep -c 'The alpha body.')" "1"

# Past the budget: listed, not quoted.
memory beta "$(printf 'long %.0s' $(seq 1 1500))"
out="$(run | context)"
ok "past the budget, bodies are dropped" "$(echo "$out" | grep -c 'The alpha body.')" "0"
ok "past the budget, every memory is still listed" "$(echo "$out" | grep -c '^- \*\*')" "2"

# Past the budget even as a list: the newest are named, the rest counted.
for i in $(seq 1 60); do memory "note-$i" "x" "$i"; done
out="$(run | context)"
ok "the listing itself stays inside the budget" "$([ "${#out}" -le 6600 ] && echo yes || echo "no (${#out} chars)")" "yes"
ok "the newest memory is named" "$(echo "$out" | grep -c 'note-1\*\*')" "1"
ok "the oldest memory is counted, not named" "$(echo "$out" | grep -c 'note-60\*\*')" "0"
ok "the cut says how many it left out" "$(echo "$out" | grep -cE 'and [0-9]+ older memories not listed')" "1"

# The budget is configurable, and nonsense in it is ignored rather than read as 0.
small="$(DRAGON_BUDDY_MEMORY_MAX_CHARS=800 run | context)"
ok "DRAGON_BUDDY_MEMORY_MAX_CHARS lowers the budget" "$([ "${#small}" -lt "${#out}" ] && echo yes || echo no)" "yes"
ok "a nonsense budget falls back to the default" "$(DRAGON_BUDDY_MEMORY_MAX_CHARS=lots run | context)" "$out"
ok "a zero budget falls back to the default" "$(DRAGON_BUDDY_MEMORY_MAX_CHARS=0 run | context)" "$out"

# The kill switches.
ok "DRAGON_BUDDY_DISABLED_HOOKS turns it off" "$(DRAGON_BUDDY_DISABLED_HOOKS=project-memory run)" ""
ok "DRAGON_BUDDY_HOOKS=off turns it off" "$(DRAGON_BUDDY_HOOKS=off run)" ""

echo
if [ "$fails" -eq 0 ]; then
	echo "project memory behaves"
else
	echo "$fails check(s) failed"
fi
exit "$fails"
