#!/usr/bin/env python3
"""Stop hook: block once if a turn changed code but never told the buddy.

Reads the Stop hook payload on stdin, walks the transcript back to the last
real user message, and looks at what happened since. If the turn edited files
and no buddy_observe went out, it blocks with a reason so the model finishes
the job. Fails open on anything unexpected -- a broken gate must never wedge
a session.
"""
import json
import os
import sys
import time

EDIT_TOOLS = {"Edit", "Write", "NotebookEdit", "MultiEdit"}
LOG = os.path.expanduser("~/.claude/buddy-gate.log")
# A buddy_observe this recently is good enough. Guards against a false nag when
# turn-boundary detection picks the wrong start (seen once, not reproducible).
RECENT_TAIL = 40
REASON = (
    "This turn changed code but never recorded it with the buddy. "
    "Call mcp__buddy__buddy_observe with a one-sentence summary of what you did, "
    "passing skills_used if you invoked any skills, then relay the reaction to the "
    "user verbatim. If the tool is not in your tool list it is deferred -- run "
    'ToolSearch with query "select:mcp__buddy__buddy_observe" first.'
)


def is_user_turn(entry):
    """A real user prompt, not a tool result or an injected reminder."""
    if entry.get("type") != "user" or entry.get("isMeta"):
        return False
    content = (entry.get("message") or {}).get("content")
    if isinstance(content, str):
        return True
    if isinstance(content, list):
        return not any(
            isinstance(b, dict) and b.get("type") == "tool_result" for b in content
        )
    return False


def tool_names(entry):
    content = (entry.get("message") or {}).get("content")
    if not isinstance(content, list):
        return
    for block in content:
        if isinstance(block, dict) and block.get("type") == "tool_use":
            yield block.get("name") or ""


def main():
    payload = json.load(sys.stdin)

    # Already blocked once this turn -- let it stop, or we loop forever.
    if payload.get("stop_hook_active"):
        return

    path = payload.get("transcript_path")
    if not path:
        return

    with open(path, errors="ignore") as fh:
        entries = []
        for line in fh:
            try:
                entries.append(json.loads(line))
            except ValueError:
                continue

    # Walk back to the last real user prompt; everything after it is this turn.
    start = 0
    for i in range(len(entries) - 1, -1, -1):
        if is_user_turn(entries[i]):
            start = i
            break

    edited = observed = False
    for entry in entries[start:]:
        for name in tool_names(entry):
            if name in EDIT_TOOLS:
                edited = True
            elif name.endswith("buddy_observe"):
                observed = True

    recent = any(
        name.endswith("buddy_observe")
        for entry in entries[-RECENT_TAIL:]
        for name in tool_names(entry)
    )
    block = edited and not observed and not recent

    try:
        with open(LOG, "a") as fh:
            fh.write(
                json.dumps(
                    {
                        "at": time.strftime("%Y-%m-%dT%H:%M:%S"),
                        "entries": len(entries),
                        "start": start,
                        "edited": edited,
                        "observed": observed,
                        "recent": recent,
                        "block": block,
                        "tail": [
                            n
                            for e in entries[-12:]
                            for n in tool_names(e)
                        ],
                    }
                )
                + "\n"
            )
    except Exception:
        pass

    if block:
        json.dump({"decision": "block", "reason": REASON}, sys.stdout)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass  # fail open
