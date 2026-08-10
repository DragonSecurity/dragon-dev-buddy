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
