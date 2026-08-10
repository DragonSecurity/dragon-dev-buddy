# The wizard template, and the patterns it encodes

The canonical copy of this library ships as `${CLAUDE_PLUGIN_ROOT}/scripts/wizard-template.sh`. Copy it; do not retype it from here. This document is the same file with the reasoning attached, and if the two ever disagree, the script is right.

## The rule that comes before the template

A wizard exists for the steps a human must take. Every step an agent can take, the agent takes — now, not in the script. A wizard that opens a browser so a human can click a button that has an API is a worse version of a shell command, and it trains people to hand-drive things that should have been automated.

Before authoring a single stage, split the procedure in two and do the agent half. What survives is the wizard: a console that has no API, a credential the agent must never see, an approval only a named human can give, a step whose failure needs a human judgement call.

## The library

Everything above the `STAGES` marker is identical in every wizard this skill generates. That consistency is the point — a human who has run one of these knows how the next one behaves, including how to stop it. Author below the marker only.

```bash
#!/usr/bin/env bash
#
# Wizard — walks a human, step by step, through a procedure only a human can do.
# Generated from the dragon-dev-buddy `runbook-wizard` skill.
#
# Everything above the STAGES marker is the wizard library. It is identical in
# every wizard this pack generates: do not hand-edit it. Author the stages below
# the marker, and nothing else.
#
#   ./wizard.sh              plan  — prints the whole procedure, changes nothing
#   ./wizard.sh --apply      apply — prompts, writes, and performs the actions
#   ./wizard.sh --forget ID  drop one recorded stage so it runs again
#
# Never run this under `set -x`, and never tee its output to a file or a
# terminal recorder. Both capture what the secret handling below is built to
# keep off the screen and off the disk.

set -euo pipefail
umask 077   # every file this script creates is owner-only until told otherwise

# bash 3.2 is the floor, because it is still the system bash on macOS and a
# cutover gets run on whatever laptop is in the room. Nothing below uses
# associative arrays, mapfile, ${var,,}, or a bare ${arr[@]} on an empty array —
# the last of those is a hard error under `set -u` on 3.2, which is how a wizard
# fails on the one machine it was needed on.

# ──────────────────────────────────────────────────────────────────────────────
# Arguments
# ──────────────────────────────────────────────────────────────────────────────

MODE=plan
FORGET=""

usage() {
  cat <<'USAGE'
usage: wizard.sh [--apply] [--forget STAGE_ID] [--help]

  (no flags)         plan mode: print every stage and every value it will ask
                     for. Nothing is prompted, written, opened or sent.
  --apply            do it for real.
  --forget STAGE_ID  remove STAGE_ID from the state file so it runs again.
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --apply)          MODE=apply ;;
    --plan|--dry-run) MODE=plan ;;
    --forget)
      FORGET="${2:-}"
      [ -n "$FORGET" ] || { printf 'expected: --forget STAGE_ID\n' >&2; exit 2; }
      shift ;;
    -h|--help)        usage; exit 0 ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

# ──────────────────────────────────────────────────────────────────────────────
# Wizard library
# ──────────────────────────────────────────────────────────────────────────────

if [ -t 1 ] && command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
  BOLD=$(tput bold); DIM=$(tput dim); RESET=$(tput sgr0)
  BLUE=$(tput setaf 4); GREEN=$(tput setaf 2); YELLOW=$(tput setaf 3); RED=$(tput setaf 1)
else
  BOLD=""; DIM=""; RESET=""; BLUE=""; GREEN=""; YELLOW=""; RED=""
fi

# Fallbacks only. The author sets the real paths at the top of the STAGES
# section, reading WIZARD_ENV_FILE / WIZARD_STATE_FILE so the person running it
# can still point them elsewhere. The library reads these at call time, so the
# later assignment wins.
ENV_FILE="${WIZARD_ENV_FILE:-.env}"
STATE_FILE="${WIZARD_STATE_FILE:-.wizard-state}"

TOTAL_STAGES=0
_STAGE_INDEX=0
_WROTE_ENV=""       # newline-separated KEYs written to ENV_FILE this run
_WROTE_SECRET=""    # newline-separated CI secret names set this run
_TODO=""            # newline-separated things the human still has to do
_n_env=0
_n_secret=0
_n_todo=0
_IGNORE_CHECKED=""

say()  { printf '  %s\n' "$1"; }
step() { printf '  %s•%s %s\n' "$BLUE" "$RESET" "$1"; }
note() { printf '  %s%s%s\n' "$DIM" "$1" "$RESET"; }
warn() { printf '  %s⚠ %s%s\n' "$YELLOW" "$1" "$RESET"; }
fail() { printf '\n  %s✗ %s%s\n\n' "$RED" "$1" "$RESET" >&2; exit 1; }

todo() {
  _TODO="${_TODO}${1}"$'\n'
  _n_todo=$((_n_todo + 1))
}

# irreversible "what cannot be undone" — say it before the human does it. A
# stage that is not idempotent is allowed; a stage that is not idempotent and
# does not say so is how a resumed run makes things worse.
irreversible() {
  printf '  %s⚠ NOT IDEMPOTENT — %s%s\n' "$YELLOW" "$1" "$RESET"
  note "Re-running this stage is not a no-op. It is recorded the moment it"
  note "completes, so a resumed run skips it rather than repeating it."
}

# ── Resumability ──────────────────────────────────────────────────────────────
#
# One line per completed stage: "<id> <UTC timestamp>". A half-finished cutover
# that cannot be resumed is worse than one that never started, so the state file
# is written as each stage lands, not at the end.

_state_has()  { [ -f "$STATE_FILE" ] && grep -q "^$1 " "$STATE_FILE"; }
_state_when() { grep "^$1 " "$STATE_FILE" | tail -n1 | cut -d' ' -f2-; }

_apply_forget() {
  [ -n "$FORGET" ] || return 0
  if [ -f "$STATE_FILE" ]; then
    local tmp
    tmp="$(mktemp "$STATE_FILE.XXXXXX")"
    grep -v "^$FORGET " "$STATE_FILE" > "$tmp" || true
    mv "$tmp" "$STATE_FILE"
    chmod 600 "$STATE_FILE"
  fi
  note "forgot stage: $FORGET"
}

stage_done() {
  [ "$MODE" = apply ] || return 0
  mkdir -p "$(dirname "$STATE_FILE")"
  printf '%s %s\n' "$1" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$STATE_FILE"
  chmod 600 "$STATE_FILE"
  printf '  %s✓ recorded%s %s\n' "$GREEN" "$RESET" "$1"
}

# stage_needed ID "Name" — announce a stage; return non-zero if a previous run
# already completed it. Wrap the whole stage body in it:
#
#   if stage_needed "rotate-token" "Rotate the deploy token"; then
#     ...
#     stage_done "rotate-token"
#   fi
stage_needed() {
  local id="$1" name="$2"
  _STAGE_INDEX=$((_STAGE_INDEX + 1))
  printf '\n%s%s▸ Stage %s/%s · %s%s\n' \
    "$BOLD" "$BLUE" "$_STAGE_INDEX" "$TOTAL_STAGES" "$name" "$RESET"
  note "id: $id"
  if [ "$MODE" = apply ] && _state_has "$id"; then
    printf '  %s✓ completed %s — skipping%s\n' "$GREEN" "$(_state_when "$id")" "$RESET"
    note "run it again with: $0 --forget $id --apply"
    return 1
  fi
  return 0
}

# ── Git safety ────────────────────────────────────────────────────────────────

# assert_ignored FILE — refuse to write a credential anywhere git is willing to
# commit. Checked once per file, before the first write to it.
assert_ignored() {
  local file="$1"
  case "$_IGNORE_CHECKED" in *"[$file]"*) return 0 ;; esac
  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    warn "not inside a git repository — cannot confirm $file is ignored; check by hand"
    _IGNORE_CHECKED="$_IGNORE_CHECKED[$file]"
    return 0
  fi
  if git ls-files --error-unmatch -- "$file" >/dev/null 2>&1; then
    fail "$file is tracked by git. A credential written into it is one commit from being published. Run: git rm --cached -- $file"
  fi
  if ! git check-ignore -q -- "$file"; then
    fail "$file is not covered by .gitignore. Add it and re-run — this wizard does not write credentials to a file git will commit."
  fi
  _IGNORE_CHECKED="$_IGNORE_CHECKED[$file]"
}

# preflight FILE... — assert every file that will receive a credential is
# ignored, before the first stage asks for anything. write_env checks again as a
# backstop, but a refusal that arrives after the human has already pasted a
# token is a poor experience even though it is a safe one. Call this once,
# straight after banner.
preflight() {
  local f
  if [ "$MODE" != apply ]; then
    for f in "$@"; do note "will check $f is gitignored and untracked"; done
    return 0
  fi
  for f in "$@"; do assert_ignored "$f"; done
}

# ── Talking to the human ──────────────────────────────────────────────────────

banner() {
  _apply_forget
  printf '\n%s%s  %s%s\n' "$BOLD" "$BLUE" "$1" "$RESET"
  printf '%s  %s stages · mode: %s%s\n\n' "$DIM" "$TOTAL_STAGES" "$MODE" "$RESET"
  if [ "$MODE" != apply ]; then
    say "Plan mode. Nothing is asked, written, opened or sent."
    say "Read the stages below, then re-run with --apply to do it for real."
    printf '\n'
    return 0
  fi
  assert_ignored "$STATE_FILE"
  say "You drive the browser; this wizard says exactly what to do and captures"
  say "what you copy back. Ctrl-C is safe at any point — completed stages are"
  say "recorded in $STATE_FILE and skipped when you re-run."
  printf '\n'
  pause "Ready?"
}

pause() {
  [ "$MODE" = apply ] || return 0
  printf '  %s%s%s ' "$DIM" "${1:-Press Enter to continue}" "$RESET"
  IFS= read -r _ || true
}

# confirm "question" — y/N gate. In plan mode it answers yes without asking, so
# the plan walks the whole procedure.
confirm() {
  [ "$MODE" = apply ] || return 0
  local reply=""
  printf '  %s? %s [y/N] %s' "$YELLOW" "$1" "$RESET"
  IFS= read -r reply || true
  case "$reply" in [Yy]*) return 0 ;; *) return 1 ;; esac
}

# open_url URL — open the human's browser, including under WSL.
open_url() {
  local url="$1"
  if [ "$MODE" != apply ]; then note "will open: $url"; return 0; fi
  printf '  %s↗ opening%s %s\n' "$GREEN" "$RESET" "$url"
  { if   command -v wslview      >/dev/null 2>&1; then wslview "$url"
    elif command -v explorer.exe >/dev/null 2>&1; then explorer.exe "$url"
    elif command -v xdg-open     >/dev/null 2>&1; then xdg-open "$url"
    elif command -v open         >/dev/null 2>&1; then open "$url"
    else warn "no browser opener found — visit it yourself: $url"; fi
  } >/dev/null 2>&1 || warn "could not open a browser — visit it yourself: $url"
}

# ── Capturing values ──────────────────────────────────────────────────────────
#
# ask        KEY "Prompt" [REGEX] [hint]   visible input
# ask_secret KEY "Prompt" [REGEX] [hint]   hidden input, never echoed
#
# REGEX is a bash regex matched in-process — the value never reaches an argv or
# a pipe on its way to being validated. On a re-run, Enter keeps whatever is
# already in ENV_FILE.
ask()        { _read_value 0 "$@"; }
ask_secret() { _read_value 1 "$@"; }

_read_value() {
  local hidden="$1" key="$2" prompt="$3"
  local pattern="${4:-.}" hint="${5:-that does not look like a valid value}"
  local current input attempt=0

  if [ "$MODE" != apply ]; then
    if [ "$hidden" = 1 ]; then note "will ask (hidden): $prompt  →  \$$key"
    else                       note "will ask: $prompt  →  \$$key"; fi
    # Bind the name to an empty value so the rest of the stage can be walked.
    # Without this, `set -u` aborts the plan at the first write_env that
    # references a value plan mode never asked for.
    printf -v "$key" '%s' ""
    return 0
  fi

  current="$(_existing "$key" || true)"
  while [ "$attempt" -lt 5 ]; do
    attempt=$((attempt + 1))
    if [ -n "$current" ]; then
      printf '  %s%s%s %s[Enter keeps the value already in %s]%s ' \
        "$BOLD" "$prompt" "$RESET" "$DIM" "$ENV_FILE" "$RESET"
    else
      printf '  %s%s%s ' "$BOLD" "$prompt" "$RESET"
    fi
    if [ "$hidden" = 1 ]; then
      IFS= read -rs input || true
      printf '\n'
    else
      IFS= read -r input || true
    fi
    if [ -z "$input" ] && [ -n "$current" ]; then input="$current"; fi
    if [[ "$input" =~ $pattern ]]; then
      printf -v "$key" '%s' "$input"
      return 0
    fi
    warn "$hint"
  done
  fail "no valid value for $key after 5 attempts"
}

# _existing KEY — the value already in ENV_FILE, if any. Never printed.
_existing() {
  [ -f "$ENV_FILE" ] || return 1
  local line
  line="$(grep -E "^$1=" "$ENV_FILE" | tail -n1)" || return 1
  line="${line#*=}"
  line="${line#\'}"
  line="${line%\'}"
  printf '%s' "$line"
}

# ── Writing values ────────────────────────────────────────────────────────────

# write_env KEY VALUE — upsert KEY into ENV_FILE, owner-only, atomically.
write_env() {
  local key="$1" value="$2" tmp
  if [ "$MODE" != apply ]; then note "will write $key → $ENV_FILE"; return 0; fi
  case "$value" in
    "") warn "$key is empty — not written"; todo "$key → $ENV_FILE"; return 0 ;;
    *"'"*)
      warn "$key not written: the value contains a single quote and this wizard will not guess at escaping it"
      todo "$key → $ENV_FILE (set by hand; the value contains a single quote)"
      return 0 ;;
  esac

  assert_ignored "$ENV_FILE"
  # The temp file is created beside ENV_FILE, not in /tmp: mktemp gives it mode
  # 600, the rename is atomic because it stays on one filesystem, and the value
  # never sits in a world-traversable directory.
  tmp="$(mktemp "$ENV_FILE.XXXXXX")"
  chmod 600 "$tmp"
  if [ -f "$ENV_FILE" ]; then grep -vE "^$key=" "$ENV_FILE" > "$tmp" || true; fi
  printf "%s='%s'\n" "$key" "$value" >> "$tmp"
  mv "$tmp" "$ENV_FILE"
  chmod 600 "$ENV_FILE"

  _WROTE_ENV="${_WROTE_ENV}${key}"$'\n'
  _n_env=$((_n_env + 1))
  printf '  %s✓ wrote%s %s → %s (mode 600)\n' "$GREEN" "$RESET" "$key" "$ENV_FILE"
}

# set_secret NAME VALUE — set a GitHub Actions repository secret.
#
# The value goes to gh over stdin. `gh secret set NAME --body "$value"` puts the
# credential in the process table, where every other user on the machine can
# read it with ps, and in this script's own shell history if anyone lifts the
# line out to run it by hand. There is no version of that which is acceptable.
set_secret() {
  local name="$1" value="$2"
  if [ "$MODE" != apply ]; then note "will set CI secret $name (value never shown)"; return 0; fi
  if [ -z "$value" ]; then
    warn "no value captured for $name — recorded as still to do"
    todo "CI secret $name"
    return 0
  fi
  if ! command -v gh >/dev/null 2>&1 || ! gh auth status >/dev/null 2>&1; then
    warn "gh is not authenticated — $name recorded as still to do"
    todo "CI secret $name — run 'gh secret set $name' and paste at the prompt (never --body)"
    return 0
  fi
  if ! confirm "Set GitHub Actions secret $name on this repository?"; then
    todo "CI secret $name"
    return 0
  fi
  if printf '%s' "$value" | gh secret set "$name" >/dev/null 2>&1; then
    _WROTE_SECRET="${_WROTE_SECRET}${name}"$'\n'
    _n_secret=$((_n_secret + 1))
    printf '  %s✓ set%s CI secret %s\n' "$GREEN" "$RESET" "$name"
  else
    warn "gh refused to set $name"
    todo "CI secret $name — 'gh secret set $name' failed; check repo permissions"
  fi
}

# set_var NAME VALUE — a non-secret CI variable. This one does pass the value in
# argv, which is exactly why it must never be used for a credential.
set_var() {
  local name="$1" value="$2"
  if [ "$MODE" != apply ]; then
    if [ -n "$value" ]; then note "will set CI variable $name = $value"
    else                     note "will set CI variable $name (public value, captured above)"; fi
    return 0
  fi
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1 \
     && gh variable set "$name" --body "$value" >/dev/null 2>&1; then
    printf '  %s✓ set%s CI variable %s\n' "$GREEN" "$RESET" "$name"
    return 0
  fi
  warn "could not set CI variable $name"
  todo "CI variable $name = $value"
}

# ── Doing things ──────────────────────────────────────────────────────────────

# act "what this does" -- cmd args...
#
# Prints the description and the exact argv before running anything, gates it
# behind a confirm, and does nothing at all in plan mode. Because the argv is
# printed, a secret can never be one of the arguments.
act() {
  local desc="$1"; shift
  if [ "${1:-}" = "--" ]; then shift; fi
  printf '  %s→ %s%s\n' "$BLUE" "$desc" "$RESET"
  note "  $*"
  if [ "$MODE" != apply ]; then note "  (plan — not run)"; return 0; fi
  if confirm "Run it now?"; then
    "$@"
  else
    warn "skipped — recorded as still to do"
    todo "$desc  ($*)"
  fi
}

# ── Closing ───────────────────────────────────────────────────────────────────

finish() {
  printf '\n%s%s  Wizard finished · mode: %s%s\n' "$BOLD" "$GREEN" "$MODE" "$RESET"
  if [ "$MODE" != apply ]; then
    note "Nothing was asked, written or sent. Re-run with --apply to do it for real."
    printf '\n'
    return 0
  fi
  if [ "$_n_env" -gt 0 ]; then
    note "wrote $_n_env value(s) to $ENV_FILE, mode 600, gitignored:"
    printf '%s' "$_WROTE_ENV" | while IFS= read -r k; do [ -n "$k" ] && note "  - $k"; done
  fi
  if [ "$_n_secret" -gt 0 ]; then
    note "set $_n_secret CI secret(s):"
    printf '%s' "$_WROTE_SECRET" | while IFS= read -r k; do [ -n "$k" ] && note "  - $k"; done
  fi
  if [ "$_n_todo" -gt 0 ]; then
    printf '\n'
    warn "still to do by hand:"
    printf '%s' "$_TODO" | while IFS= read -r t; do [ -n "$t" ] && note "  - $t"; done
  fi
  printf '\n'
  note "Nothing was committed. This wizard never runs git add, git commit or git push."
  printf '\n'
}

_on_exit() {
  local code="$1"
  if [ "$code" -ne 0 ] && [ "$MODE" = apply ]; then
    printf '\n  %s⚠ stopped during stage %s of %s. Completed stages are recorded in %s; re-run with --apply to resume.%s\n\n' \
      "$YELLOW" "$_STAGE_INDEX" "$TOTAL_STAGES" "$STATE_FILE" "$RESET" >&2
  fi
}
trap '_on_exit $?' EXIT

# ──────────────────────────────────────────────────────────────────────────────
# STAGES — author below this marker only. One stage per step the human takes.
# Set TOTAL_STAGES to the number of stages you write.
# ──────────────────────────────────────────────────────────────────────────────

TOTAL_STAGES=2
ENV_FILE="${WIZARD_ENV_FILE:-.env.local}"
STATE_FILE="${WIZARD_STATE_FILE:-.dragon-buddy/wizard-example.state}"

banner "Example — provider API token"
preflight "$ENV_FILE"

# ── Replace everything below with the real stages ─────────────────────────────

if stage_needed "issue-token" "Provider — issue a scoped API token"; then
  say "You issue the token; this wizard stores it. It is never printed."
  open_url "https://provider.example.com/account/api-tokens"
  step "Create token → template 'Read analytics' → scope it to this project only."
  step "Set an expiry. A token with no expiry outlives the person who made it."
  step "Create, then copy the value — the page shows it once."
  ask_secret PROVIDER_TOKEN "Paste the token:" '^[A-Za-z0-9_-]{32,}$' \
    "expected 32+ characters of [A-Za-z0-9_-]; that is not a token"
  ask PROVIDER_ACCOUNT_ID "Paste the account ID shown in the sidebar:" '^[0-9a-f]{32}$' \
    "expected 32 hex characters"
  write_env PROVIDER_TOKEN "$PROVIDER_TOKEN"
  write_env PROVIDER_ACCOUNT_ID "$PROVIDER_ACCOUNT_ID"
  set_secret PROVIDER_TOKEN "$PROVIDER_TOKEN"   # CI needs this one
  set_var PROVIDER_ACCOUNT_ID "$PROVIDER_ACCOUNT_ID"
  stage_done "issue-token"
fi

if stage_needed "revoke-old" "Provider — revoke the token being replaced"; then
  irreversible "the old token stops working the moment you revoke it"
  say "Do this only once the new token is in CI — stage 1 above."
  open_url "https://provider.example.com/account/api-tokens"
  step "Find the token named 'ci-deploy (old)'."
  step "Revoke it. Do not delete the audit entry."
  pause "Revoked?"
  act "re-run the deploy workflow with the new token" -- gh workflow run deploy.yml
  stage_done "revoke-old"
fi

# ──────────────────────────────────────────────────────────────────────────────

finish
```

## Secret handling — the rules the library exists to enforce

These are not style preferences. Each one names a place a credential ends up when it is broken.

**Never echoed.** `ask_secret` reads with `read -rs`. The value is never printed back, never included in a confirmation line, never shown in the closing summary. The summary names keys, not values. A wizard run over a shared screen, a recorded call, or a terminal with scrollback is the normal case, not the exception.

**Never in an argument.** `gh secret set NAME --body "$token"` puts the token in the process table, where any other user on the machine reads it with `ps`, and in the shell history of anyone who lifts the line out to run by hand. The library pipes to stdin instead: `printf '%s' "$value" | gh secret set "$name"`. The same rule governs `act`, which prints its own argv precisely so that a secret cannot be smuggled into one unnoticed. `set_var` does pass its value in argv — which is exactly why it is only ever used for a public value.

**Never validated through a subprocess.** `_read_value` matches with bash's `[[ =~ ]]`. Piping the value to `grep` would work, but it moves the secret through a second process for no reason; the pattern belongs in argv, the value does not.

**Never world-readable.** `umask 077` at the top of the script, `mktemp` beside the destination rather than in `/tmp`, `chmod 600` on both the temp file and the final file. The temp file is created next to `ENV_FILE` for three reasons at once: `mktemp` gives it mode 600, the rename is atomic because it never crosses a filesystem, and the value never sits in a directory every user on the box can traverse.

**Never under `set -x`, never teed.** Both defeat everything above, and both are things a frustrated operator does at 2am to see what is going wrong. The script says so in its own header.

**Never committed.** The generated wizard runs no `git add`, no `git commit`, no `git push` — ever. Before it writes a credential anywhere it calls `assert_ignored`, which fails if the destination is tracked (`git ls-files --error-unmatch`) or not covered by `.gitignore` (`git check-ignore`). Both checks matter: a file that is listed in `.gitignore` but was committed before the rule existed is still tracked, and the ignore rule does nothing for it.

Call `preflight` for every credential destination immediately after `banner`. The check inside `write_env` is a backstop; a refusal that arrives after the human has already pasted a token into a prompt is safe but unpleasant, and unpleasant wizards get run with the guard commented out.

## Prompt-and-validate loops

`ask` and `ask_secret` take an optional bash regex and a hint:

```bash
ask_secret CF_API_TOKEN "Paste the token:" '^[A-Za-z0-9_-]{40}$' \
  "a Cloudflare token is 40 characters of [A-Za-z0-9_-] — that is not one"
ask CF_ACCOUNT_ID "Paste the Account ID from the sidebar:" '^[0-9a-f]{32}$' \
  "expected 32 hex characters"
```

Validate every value, and validate it against the shape the provider actually issues. The failure this prevents is specific and expensive: a human copies the label instead of the value, or copies a truncated token from a wrapped terminal, the wizard writes it, CI fails four minutes later with an authentication error, and the person now debugging it is not the person who ran the wizard. Validation at the prompt is the only place the mistake is one keystroke from being fixed.

Five attempts, then `fail`. An unbounded loop with a wrong value in the clipboard is a trap.

Where a value has no checkable shape, still constrain what you can — non-empty, a length floor, a prefix (`sk_live_`, `ghp_`, `AKIA`). A prefix check also catches the far worse mistake of pasting a *production* credential into a stage that asked for a test one.

## Opening a URL portably

```bash
if   command -v wslview      >/dev/null 2>&1; then wslview "$url"
elif command -v explorer.exe >/dev/null 2>&1; then explorer.exe "$url"
elif command -v xdg-open     >/dev/null 2>&1; then xdg-open "$url"
elif command -v open         >/dev/null 2>&1; then open "$url"
else warn "no browser opener found — visit it yourself: $url"; fi
```

WSL first, because a WSL shell has `xdg-open` on the path and it opens nothing a human can see. The whole block is `|| warn`: a wizard that dies because the browser would not launch has failed at the one thing it could have shrugged off, and the URL is printed either way.

Open the URL *before* asking for the value it produces. The human should never be looking at a prompt for a page they have not been sent to.

## Where a value belongs

| Destination | Use it for | How |
| --- | --- | --- |
| `.env` / `.env.local` | anything local development needs | `write_env KEY "$value"` — gitignored, mode 600 |
| CI secret | values a workflow needs and humans must not read back | `set_secret NAME "$value"` — stdin, never `--body` |
| CI variable | non-sensitive config a workflow needs | `set_var NAME "$value"` — argv, so never a credential |
| nowhere | a value that only had to exist | capture nothing; the stage is a pure action |

Two rules decide the row. A credential goes to CI **as a secret**, never as a variable, because a variable is readable in the Actions UI and printable in a log. And a value goes to `.env` only if something on this machine reads it — a deploy token that only CI uses should not have a local copy at all, because a local copy is one more place it leaks from.

Every `set_secret NAME` must match a `secrets.NAME` reference in `.github/workflows/`. Grep for them while scoping; a secret set under a name nothing reads is a wizard that appears to work and a pipeline that stays broken.

## Resumability via a state file

One line per completed stage — `<id> <UTC timestamp>` — appended the moment the stage lands, not at the end of the run. The state file is gitignored and mode 600.

```bash
if stage_needed "revoke-old-token" "Revoke the leaked token"; then
  ...
  stage_done "revoke-old-token"
fi
```

The failure this exists for: a cutover that gets half-way, the laptop sleeps or the human has to leave, and the second run repeats a step that was not idempotent. A half-finished cutover that cannot be resumed is worse than one that never started, because the system is now in a state neither the old runbook nor the new one describes.

Stage ids are stable slugs and they are printed on screen, so a human reading a transcript can say exactly where it stopped. `--forget ID` drops one stage so it runs again — that is the supported way to redo a step, rather than deleting the whole state file and re-running a token rotation from the top.

Order the stages so that the irreversible one is as late as possible, and so that everything it depends on is already recorded as done. In a credential rotation this means: issue the new credential, install it everywhere, verify it works, and only then revoke the old one. Revoking first turns a rotation into an outage.

Mark any stage that is not idempotent with `irreversible "what cannot be undone"`, on screen, before the human acts.

## Plan mode is the default

Running the wizard with no flags walks every stage and prints exactly what it will open, ask for, write and send — and does none of it. `--apply` does it for real.

This is not a courtesy. A wizard is generated code performing outward-facing actions against a live account under time pressure, frequently written by an agent that has never seen the dashboard in question. The plan run is where a human catches the stage that names the wrong repository, the token that would go to the wrong environment, or the revocation that is sequenced before the replacement.

The mode gate lives inside every helper, so a stage body is written once and reads the same in both modes. The consequence is that stages must call helpers and not bare commands: anything else runs in plan mode, which is the one thing plan mode promises will not happen. Use `act "description" -- cmd args...` for everything the library does not already cover.

`confirm` answers yes in plan mode without asking, so the plan walks the whole procedure rather than stopping at the first gate. Phrase every confirmation so that yes is the path that continues — "Set the CI secret?", not "Skip this stage?" — or the plan describes a run nobody would have.

`_read_value` binds its key to an empty string in plan mode. Without that, `set -u` aborts the plan at the first `write_env` referencing a value plan mode deliberately did not ask for — the plan fails while the apply run would have worked, which reads as a broken wizard.

## Portability

`bash 3.2` is the floor, because that is still the system bash on macOS and a cutover gets run on whichever laptop is in the room. Three things follow: no associative arrays, no `mapfile`, no `${var,,}` — and no bare `${arr[@]}` on an empty array, which is a hard error under `set -u` on 3.2. That last one is why the library accumulates its summaries in newline-delimited strings with integer counters rather than in arrays. The naive version passes every test on the author's machine and dies on the operator's.

`date -u +%Y-%m-%dT%H:%M:%SZ` rather than `date -Is`, which BSD `date` does not have.

## Verifying before handing it over

```sh
bash -n wizard.sh                    # syntax
shellcheck wizard.sh                 # if available
./wizard.sh                          # plan mode: prints, changes nothing
```

Run plan mode. It is safe by construction and it is the only end-to-end exercise available — never run `--apply` yourself, because it blocks on human input and performs real outward-facing actions.

Then check the two halves of the file separately, because they are guaranteed by different things. The library is guaranteed by being unmodified:

```bash
diff <(sed '/^# STAGES/,$d' wizard.sh) \
     <(sed '/^# STAGES/,$d' "${CLAUDE_PLUGIN_ROOT}/scripts/wizard-template.sh")
```

The stages are guaranteed by reading them:

```bash
sed -n '/^# STAGES/,$p' wizard.sh |
  grep -nE -- '--body|set -x|git (add|commit|push)|echo .*(TOKEN|SECRET|KEY|PASSWORD)'
```

Every hit in the second is either a bug or needs a comment saying why it is not one: `--body` on a secret, `set -x` anywhere, any git write, any echo of a captured credential.

Grep the stages and not the whole file. Run that pattern over an untouched template and it returns five hits — the header warning about `set -x`, two comments explaining why `--body` is banned, `set_var`'s one legitimate `--body`, and `finish`'s line promising nothing was committed. All five are correct, none of them will ever go away, and a check that fires on every wizard ever generated is a check the next person waves through. The diff above is what covers the library; the grep only has to cover what you wrote.

Finally, trace statically: every value from the scoping step is captured by exactly one `ask`/`ask_secret`, lands where scoping said it lands, and every `set_secret` name matches a `secrets.*` reference in CI.
