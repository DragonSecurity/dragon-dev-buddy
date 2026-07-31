# Reachability and supply-chain reference

## Determining reachability

The question is never "is this package installed." It is "does an attacker-controlled value reach the vulnerable function."

**Method, any ecosystem:**

1. Read the advisory and identify the specific affected function or feature. Advisories are usually narrower than their titles: "prototype pollution in lodash" is really "`_.merge` and `_.set` with attacker-controlled keys."
2. Grep for the import. No import anywhere means it is transitive; find which direct dependency pulls it in and check whether *that* package uses the affected path.
3. Grep for the call. Imported but never called is unreachable.
4. Trace backwards from the call to an entry point. If nothing untrusted reaches it, it is unreachable in practice — say so, and say what would change that.

**Say what you did.** "Grepped for `_.merge`, three call sites, all on internal config objects assembled at boot, none from a request" is a verdict a reviewer can check. "Probably not reachable" is not.

**Default to reachable when you cannot tell.** Dynamic dispatch, reflection, plugin loading, and heavy dependency injection all make static tracing unreliable. Being wrong toward caution costs an upgrade; being wrong the other way costs an incident.

**Ecosystem tools that do this properly:**

| Ecosystem | Tool | Notes |
| --- | --- | --- |
| Go | `govulncheck` | Genuine call-graph reachability. Trust it. |
| Rust | `cargo audit` + `cargo-deny` | Advisory-level; no reachability. |
| Node | `npm audit`, `osv-scanner`, `socket.dev` | No reachability. Very noisy. `--omit=dev` first. |
| Python | `pip-audit` | Advisory-level. |
| Java | OWASP Dependency-Check | High false-positive rate on version matching. |
| Containers | `trivy`, `grype` | Scans OS packages too, which is usually where the volume comes from. |

## Reading an advisory without over-trusting it

- **CVSS is generic by construction.** It is scored for a hypothetical worst-case deployment. A 9.8 on a function you call at boot with a hardcoded argument is not a 9.8 for you.
- **Check the affected version range carefully.** Scanners routinely flag versions that were never affected, especially on Java and on packages that backport fixes.
- **Check whether there is an exploit.** "Proof of concept published" and "theoretical" are different urgencies, even at the same score.
- **`GHSA` and `CVE` for the same bug are one advisory.** Deduplicate; scanners will report both.
- **Denial-of-service advisories in a dev dependency are almost never real.** The build machine crashing is a CI failure, not a security event — unless CI runs untrusted pull requests, in which case reclassify everything dev-only as reachable.

## Supply-chain checks no scanner runs

Work this list manually. These are the risks that produce incidents rather than tickets.

- [ ] **Install scripts.** `npm ls --json` or read manifests for `postinstall`/`preinstall`. Any install script in a dependency runs arbitrary code on every developer machine and every CI runner. List them and justify each.
- [ ] **Maintenance status.** Last release date, open issue count, whether the repository is archived. Two years without a release on a package that parses untrusted input is a finding regardless of current advisories.
- [ ] **Maintainer concentration.** A single-maintainer package with millions of downloads is a takeover target. Note it; you probably will not remove it.
- [ ] **Recent ownership transfer.** The `event-stream` pattern: a new maintainer takes over a dormant popular package, then ships a malicious minor version.
- [ ] **Typosquats.** Compare every direct dependency name against the popular package it resembles. `crossenv`, `python-dateutil` versus `dateutil`, `reqeusts`.
- [ ] **Recently added dependencies.** `git log -p -- package.json` for the last 90 days. Ask who approved each and why. New dependencies get less scrutiny than new code, which is backwards.
- [ ] **Dependency count trend.** Direct count going up faster than features is worth naming, even without a specific finding.
- [ ] **Duplicate versions of the same package.** Often means one path is patched and another is not.
- [ ] **Packages fetched from a non-default registry**, a git URL, or a tarball URL. Each is a supply chain you are not monitoring.

## Lockfiles and pinning

**No lockfile, or an uncommitted one**, in a deployed application: this is the finding that outranks the advisories. It means the code in production was never the code that was tested, and a compromised upstream release is installed automatically.

**Ranges in the manifest are fine when the lockfile is committed.** The lockfile is what resolves. Ranges without a lockfile mean every install is a fresh roll of the dice.

**Pin exact versions for anything that runs at build time with credentials present** — CI actions especially. Pin GitHub Actions to a commit SHA, not a tag; tags are mutable and moving one is a supported way to change what runs in your pipeline.

**Automated update PRs (Dependabot, Renovate) are a control only if someone merges them.** Ninety open update PRs is worse than none: it is a queue that trains the team to ignore the signal. If that is the state, the recommendation is to configure grouping and auto-merge for patch updates, not to open more PRs.

## Compensating controls when there is no patch

| Situation | Control |
| --- | --- |
| Vulnerable parse function, no fix | Validate and constrain input before it reaches the library: size caps, type checks, allowlists. |
| Vulnerable package, no alternative | Isolate: separate process, restricted runtime, no credentials in that process's environment. |
| Transitive, parent will not move | Ecosystem override (`overrides`, `resolutions`, `replace`). Test carefully; you are running an untested combination. |
| Abandoned package | Vendor it. A copy you maintain is better than a dependency nobody does. Budget the maintenance honestly. |
| Advisory that is dev-only but CI runs untrusted PRs | Fix the CI trust model instead: no secrets on fork PRs, no auto-run on untrusted branches. |
