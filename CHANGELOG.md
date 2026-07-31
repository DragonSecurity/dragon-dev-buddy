# Changelog

Notable changes to the pack. Versions track `.claude-plugin/plugin.json`.

## Unreleased

### Added

- `secure-code-review`: **PR mode** — reviewing a pull request rather than a bare
  diff. Covers fetching `base...head` rather than the branch tip, reading removed
  lines as deliberately as added ones, treating the PR description as a
  hypothesis to check the diff against, and handling a fork PR as untrusted code
  that is never executed locally.
- `secure-code-review`: **batch review** of several PRs in one pass — triage by
  what each diff touches, published before the reviews start; per-PR trust
  positions; and a cross-PR pass for the defects a per-PR review structurally
  cannot see (add-then-loosen across two PRs, merge-order constraints, one
  pattern repeated across many PRs).
- `secure-code-review`: `references/pr-review.md` with the `gh`/`git` mechanics,
  the batch triage depth table, and how to post a review back without approving.
- `internal/skillpack`: a Go test suite that validates the pack's structure and
  conventions. See [CONTRIBUTING.md](CONTRIBUTING.md).
- CI running `gofmt`, `go vet` and `go test` on every push and pull request, and
  building the installable bundle as an artifact.
- `scripts/build-plugin.sh` to produce `dist/dragon-dev-buddy.plugin`.
- `CONTRIBUTING.md`, `SECURITY.md`, this changelog.

### Fixed

- `ship-it` had a worked example that no `SKILL.md` section pointed at, so the
  model never loaded it. Found by the new test suite.
- The README omitted `security-audit-orchestrator` entirely and undercounted the
  pack at fourteen skills. Also found by the test suite, which now pins the count.
- A local absolute path (`/Users/...`) in a `buddy-companion` example.

## 1.0.0

Initial pack: 16 skills, `buddy-setup` onboarding, and the buddy MCP reporting
contract.
