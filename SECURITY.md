# Security policy

## Reporting a vulnerability

Report privately through
[GitHub Security Advisories](https://github.com/DragonSecurity/dragon-dev-buddy/security/advisories/new).
Do not open a public issue for anything exploitable.

Expect an acknowledgement within three working days and an assessment within ten.

## What counts as a vulnerability here

This pack ships no runtime — it is markdown that instructs a model. The
interesting attack surface is therefore not memory safety, it is what the
instructions cause a model to do:

- **Prompt injection through skill content.** A skill that reads
  attacker-controlled input (a diff, a dependency advisory, a vulnerability
  report, a PR description) and treats it as instruction rather than data.
- **Instructions that leak.** A skill that would cause credentials, findings, or
  a client name to be written somewhere unintended — a public report path, a
  commit, an outbound request.
- **Missing authorization gates.** The skills that can produce exploit code check
  `engagement.authorized_scope` first. A path around that check is a
  vulnerability in this pack.
- **Unsafe defaults in the build or CI.** A workflow that runs untrusted code
  with repository secrets, an unpinned action, a bundle that ships something it
  should not.

## What does not

- A skill giving security advice you disagree with. Open an issue.
- Findings produced by these skills about *your* codebase — those are yours.
- The example configurations under `examples/`, which contain deliberately
  vulnerable code as illustration and are never executed.

## Scope of the pack itself

`config.example.json` documents an `engagement` block that can name a client and
an authorization reference. That block is the reason `.dragon-buddy/` is in
`.gitignore`. If you commit your own config, check it first.
