#!/usr/bin/env bash
# Exercise hooks/buddy-observe-gate.mjs against a throwaway HOME.
#
# The gate is a hook, so the only honest way to check it is to feed it the JSON a
# hook is fed and read what it writes -- both to stdout, which is the block
# decision, and to the state directory, which is the memory the next event reads.
# Its whole job is a claim about a turn, and a turn is three or four events; a
# unit test of any one function could not have caught the bug this file was
# written for, where every function was individually correct.
#
# It runs the gate under HOME="$(mktemp -d)", which is what keeps it off the
# developer's own buddy-gate log. Node resolves homedir() from HOME on both
# platforms this pack runs on.
set -u

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
gate="$root/hooks/buddy-observe-gate.mjs"

HOME="$(mktemp -d)"
export HOME
mkdir -p "$HOME/.claude"
trap 'rm -rf "$HOME"' EXIT

session=check-session
mark="$HOME/.claude/buddy-gate/$session.dirty"
log="$HOME/.claude/buddy-gate.log"
fails=0

run() { printf '%s' "$1" | node "$gate"; }
marked() { [ -f "$mark" ] && echo yes || echo no; }
ok() {
	if [ "$2" = "$3" ]; then
		echo "ok   $1"
	else
		echo "FAIL $1: expected [$3], got [$2]"
		fails=$((fails + 1))
	fi
}

edit="{\"hook_event_name\":\"PostToolUse\",\"session_id\":\"$session\",\"tool_name\":\"Edit\"}"
observed="{\"hook_event_name\":\"PostToolUse\",\"session_id\":\"$session\",\"tool_name\":\"mcp__plugin_dragon-dev-buddy_buddy__buddy_observe\",\"tool_response\":{\"ok\":true}}"
observe_failed="{\"hook_event_name\":\"PostToolUse\",\"session_id\":\"$session\",\"tool_name\":\"mcp__buddy__buddy_observe\",\"tool_response\":{\"is_error\":true}}"
prompt="{\"hook_event_name\":\"UserPromptSubmit\",\"session_id\":\"$session\",\"prompt\":\"hi\"}"
stop="{\"hook_event_name\":\"Stop\",\"session_id\":\"$session\",\"stop_hook_active\":false}"
stop_again="{\"hook_event_name\":\"Stop\",\"session_id\":\"$session\",\"stop_hook_active\":true}"
unwired="{\"hook_event_name\":\"SessionStart\",\"session_id\":\"$session\",\"source\":\"resume\"}"

# An edit marks the session, and says nothing while doing it.
out="$(run "$edit")"
ok "an edit marks the session" "$(marked)" "yes"
ok "an edit is silent" "$out" ""

# A turn that changed code and never recorded is blocked exactly once. The mark
# is consumed by the block, so a failure to nag can never wedge the session.
out="$(run "$stop")"
ok "an unrecorded turn is blocked" "$(echo "$out" | grep -c '"decision":"block"')" "1"
ok "the block consumes the mark" "$(marked)" "no"
ok "a turn that changed nothing passes" "$(run "$stop")" ""

# The turn boundary. A mark from a turn that has already ended -- a background
# agent's write, or an interrupted turn -- must not be charged to the next turn.
run "$edit" >/dev/null
out="$(run "$prompt")"
ok "a prompt is silent" "$out" ""
ok "a prompt drops the previous turn's mark" "$(marked)" "no"
ok "the turn after a stale mark passes" "$(run "$stop")" ""

# Recording during the turn is what the gate is asking for.
run "$edit" >/dev/null
run "$observed" >/dev/null
ok "an observation clears the mark" "$(marked)" "no"
ok "a recorded turn passes" "$(run "$stop")" ""

# An observation that errored recorded nothing, so it clears nothing.
run "$edit" >/dev/null
run "$observe_failed" >/dev/null
ok "a failed observation keeps the mark" "$(marked)" "yes"
ok "a turn already blocked once passes" "$(run "$stop_again")" ""
run "$prompt" >/dev/null

# Only Stop may write a decision. Anything else reaching that branch is either a
# wedged session or, on UserPromptSubmit, a block reason read back as context.
ok "an unwired event is silent" "$(run "$unwired")" ""

# The log is the buddy's evidence for how often work gets recorded unasked, and
# it counts a clear carrying had:true as a turn that recorded voluntarily. A
# reset is not one of those.
ok "resets are logged as resets" "$(grep -c '"event":"reset"' "$log")" "2"
ok "one voluntary clear" "$(grep -c '"event":"clear".*"had":true' "$log")" "1"
ok "one block" "$(grep -c '"event":"stop","block":true' "$log")" "1"

# Handling an event is half the claim; the manifest has to send it. The turn
# boundary is one line of hooks.json away from being dead code that passes every
# check above it.
wired="$(node -e '
  const m = require("'"$root"'/hooks/hooks.json").hooks;
  const gate = (e) => (m[e] ?? []).some((g) => (g.hooks ?? []).some((h) => String(h.command).includes("buddy-observe-gate.mjs")));
  console.log(["UserPromptSubmit", "PostToolUse", "Stop"].filter(gate).join(","));
')"
ok "hooks.json wires the gate to every event it handles" "$wired" "UserPromptSubmit,PostToolUse,Stop"

echo
if [ "$fails" -eq 0 ]; then
	echo "the observe gate behaves"
else
	echo "$fails check(s) failed"
fi
exit "$fails"
