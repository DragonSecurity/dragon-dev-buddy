# First-pass repo survey

The goal is a complete draft profile before the first question. Spend the tool calls; they are cheaper than the user's attention.

## Universal first look

```
README.md                  what it claims to be
.gitignore                 what they already know not to commit
.env.example               the shape of the secrets, without the secrets
Dockerfile, compose.yaml   runtime, base image, exposed ports, USER directive
.github/workflows/         CI provider, what runs on PR vs main, deploy trigger
infra/ terraform/ k8s/     IaC surface, which cloud
```

## By ecosystem

| Ecosystem | Manifest | Lockfile | Test command usually | Notes |
| --- | --- | --- | --- | --- |
| Node | `package.json` | `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock` | `scripts.test` | read `scripts` wholesale; it is the practice block |
| Python | `pyproject.toml`, `requirements.txt` | `poetry.lock`, `uv.lock` | `pytest` | check for `[tool.ruff]`, `[tool.bandit]` |
| Go | `go.mod` | `go.sum` | `go test ./...` | `go.mod` go directive gives the runtime version |
| Rust | `Cargo.toml` | `Cargo.lock` | `cargo test` | check for `cargo-deny` config |
| Java | `pom.xml`, `build.gradle` | — | `mvn test`, `gradle test` | dependency tree is deep; SCA matters more here |
| Ruby | `Gemfile` | `Gemfile.lock` | `bundle exec rspec` | check for `brakeman` |
| PHP | `composer.json` | `composer.lock` | `phpunit` | |

No lockfile in a deployed application is itself a finding. Note it and move on; `dependency-audit` will pick it up.

## Detecting a spec-driven codegen stack

Some codebases generate their API clients from an OpenAPI spec rather than hand-writing them, and recognizing this shapes the whole profile — it means the review and audit skills should target the spec and the generator config, not the generated output. The signature (this is the `waggle` reference architecture):

```
go.mod: github.com/danielgtaylor/huma/v2   Go API that emits an OpenAPI spec (usually on go-chi)
docs/openapi.json  or  openapi.{json,yaml}  the generated spec — the source of truth
openapi-ts.config.ts, openapitools.json     TS SDK generation via @hey-api/openapi-ts
.openapi-generator/, .openapi-generator-ignore   openapi-generator output (Go SDK and/or TF provider)
sdk/go, sdk/ts, ui/src/sdk                   vendored generated SDKs
terraform-provider-*/                        a generated Terraform provider
mprocs.yaml                                  multi-process dev runner often paired with this stack
```

When you see this, record it in the profile and set the stack accordingly: Huma+chi Go backend, generated Go/TS SDKs, generated Terraform provider, React+shadcn+react-query UI consuming the vendored TS SDK. Note it in `project.stack` and mention it in the summary, because it changes the top-skill routing — `secure-code-review` and `dependency-audit` both have a `codegen-pipeline.md` reference that only matters for these repos, and a review pointed at generated `model_*` files is wasted effort.

Half-signatures still count: a Huma backend with no generated clients yet is still a spec-first codebase; a repo with `.openapi-generator/` but no Huma is generating clients for someone else's API. Record what you actually see.

**Persistence and migrations, in the same idiom.** The reference stack is GORM database-first with generated migrations:

```
go.mod: gorm.io/gorm, gorm.io/driver/postgres     GORM models are the schema source of truth
go.mod: ariga.io/atlas-provider-gorm              atlas generates migrations by diffing the GORM models
atlas.hcl                                          the atlas config that points at the GORM models
go.mod: github.com/pressly/goose/v3               goose applies/maintains the generated migrations
internal/migrations/{control,tenant}/             migrations split per schema/surface
```

The point worth recording: schema changes flow **models → atlas generates → goose applies**, not GORM `AutoMigrate`. A review or hardening pass on such a repo should treat the GORM models as the schema authority and the migration files as generated output — flag hand-edited migrations that no longer match the models, the same way you would flag drift in a generated SDK.

**Dev environment.** A `docker-compose.yml` (or `compose.yml`) providing the dev database is the norm — the reference stack runs Postgres + **pgbouncer** (pooler) + **mailpit** (SMTP catcher) + **pgweb** (DB browser), env-driven. This is dev-only infrastructure: note it in the profile, and note for `secrets-and-config-audit` that dev compose files are a common home for default credentials that must never be reused in any deployed environment.

## Detecting existing security tooling

Presence of any of these means the answer to "do you have tooling" is yes, whatever the user says:

```
.semgrep.yml, .semgrepignore          SAST
.github/workflows/codeql*.yml         SAST
.gitleaks.toml, .trufflehog*          secret scanning
.snyk, renovate.json, dependabot.yml  SCA / dependency automation
trivy.yaml, .grype.yaml               container and SCA
.pre-commit-config.yaml               read it; often contains all of the above
```

## Inferring exposure from the code

These are signals, not proof. Confirm with the user, but lead with what you found.

- A `Dockerfile` with `EXPOSE 80` or `443`, or an ingress in k8s manifests, points at public or authenticated.
- Auth middleware applied globally versus per-route tells you whether the default is closed or open. A per-route default-open pattern is worth flagging during setup itself.
- A `cors` config with `origin: "*"` means somebody expected browser traffic from anywhere.
- No auth dependency in the manifest at all, plus an HTTP server, usually means internal or unfinished. Ask which.

## Inferring data sensitivity from the schema

Look at migrations, ORM models, or the `CREATE TABLE` statements. Column names are honest:

- `email`, `phone`, `address`, `dob`, `ip_address` → `pii`
- `card_`, `iban`, `account_number`, `amount`, `balance` → `financial`
- `diagnosis`, `patient`, `mrn`, `insurance` → `phi`
- `token`, `secret`, `api_key`, `refresh_token`, `password_hash` → `credentials`

`credentials` outranks the others when several apply, because compromise there cascades into other systems.

## What not to do during setup

- Do not run the test suite, the linter, or any scanner. Setup is read-only and fast. Those belong to the skills that were built for them.
- Do not open every source file. Manifests, config and schema answer the setup questions; source code answers the audit questions.
- Do not report findings. If you spot something alarming during the survey, note it into `security.known_risk_areas` and tell the user which skill will handle it properly. Setup is not an audit.
