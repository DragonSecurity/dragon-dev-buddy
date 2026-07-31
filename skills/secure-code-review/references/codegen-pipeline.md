# Reviewing spec-driven codegen codebases

Some codebases do not hand-write their API clients. An API defines an OpenAPI spec, and the SDKs and infrastructure clients are *generated* from it. The reference shape for this pack: a **Huma v2** (on go-chi) Go API emits `openapi.json`; a **Go SDK** (openapi-generator), a **TypeScript SDK** (`@hey-api/openapi-ts` with a react-query plugin), and a **Terraform provider** (openapi-generator) are all generated from that spec. The `waggle` repo at `~/projects/glueops/waggle` is the canonical example.

This changes where the review effort goes. Reviewing generated code line by line is mostly wasted — it is regenerated on the next spec change and your comments vanish. Review the **three things that are actually authored**: the spec, the handler behind it, and the generator configuration. Everything else is a function of those.

## What to review, and what to skip

| Artifact | Authored or generated? | Review it? |
| --- | --- | --- |
| The Huma handler + service + repo | authored | **Yes** — this is where the real logic and the real bugs live |
| The OpenAPI spec (`docs/openapi.json`) | generated *from* the Go types/handlers | **Yes, as an output** — it is the contract; review what it exposes |
| `sdk/go`, `sdk/ts`, `ui/src/sdk` | generated | **No line-by-line** — review the *config* that shapes them |
| `openapi-ts.config.ts`, `.openapi-generator*`, `openapitools.json` | authored | **Yes** — small files, high leverage |
| The Terraform provider under `terraform-provider-*` | generated | **No line-by-line** — review the spec surface it exposes and its auth handling |

If you find yourself commenting on a `model_*.go` or a generated `sdk/` file, stop. Either the finding belongs in the spec (fix the Huma type, regenerate) or in the generator config. A finding written against generated code is erased by the next `go generate`.

## The spec is a security surface

With Huma, the spec is derived from your Go request/response structs and their tags. So the spec exposes exactly what those structs expose — which is easy to get wrong, because a struct written for internal use becomes a public contract the moment it is returned from a handler.

- **Over-exposure in output bodies.** A `*_output_body` struct that embeds the full domain model leaks whatever the model carries — internal ids, tenant references, password hashes, soft-delete flags, audit columns. Review every output body for fields the caller has no business seeing. This is the single highest-yield check in a Huma codebase, and it propagates: an over-exposed field is now in the Go SDK, the TS SDK, and the Terraform provider's state.
- **Under-constrained input bodies.** Huma validates from struct tags (`minimum`, `maxLength`, `pattern`, `enum`, `required`). A missing tag is a missing validation *for every generated client*, because the clients trust the spec. Check that input bodies constrain what the handler assumes — a field the service treats as bounded must be bounded in the tag, or the bound exists nowhere.
- **Mass assignment via input bodies.** If an input body includes a field the caller should not set (role, tenant, owner, price), the generated clients will happily send it and the handler will happily bind it. Server-controlled fields must not appear in input bodies at all.
- **Auth advertised correctly.** Huma security schemes drive what the spec says is protected. An operation missing its security requirement is advertised as public to every client and every consumer of the spec. Confirm the security scheme is on every operation that needs it, not just enforced in middleware — the two can disagree, and the spec is what the SDKs believe.

## Auth handling in generated clients

The generated SDKs and the Terraform provider inherit whatever auth pattern the generator config sets. Review the config, not the output:

- **How does the TS SDK carry the token?** `@hey-api/client-fetch` is configured once; check that the token comes from a runtime source, not baked into the generated code or a committed config. A token in `openapi-ts.config.ts` or in a generated file is a leak that regeneration reintroduces every time.
- **Does the Terraform provider store credentials in state?** openapi-generator-derived providers commonly stash the configured API token, and **Terraform state is plaintext by default**. A provider that writes the token into a resource or data-source attribute puts it in every `terraform.tfstate`, which lands in the state backend and often in CI logs. This is a recurring critical finding in generated providers — check the provider schema for any credential attribute that is stored rather than write-only/sensitive.
- **TLS verification.** Generated clients sometimes expose an "insecure/skip-verify" knob. Confirm it is not defaulted on, and not set in the config for convenience against a staging host and then shipped.

## Drift between spec and generated code

Generated code is only safe if it matches the current spec. When they drift, the client enforces yesterday's contract:

- If `docs/openapi.json` is newer than the vendored `ui/src/sdk` or `sdk/go`, the generated client may be missing a validation or an auth requirement that the spec now defines. Flag stale generated output as a finding — the fix is `regenerate`, but the *risk* is that a control added to the spec is not yet in the clients.
- Prefer a codebase that regenerates in CI and fails on a diff, so drift cannot ship. If it does not, that absence is itself worth noting (route it to `hardening-playbook`).

## Where the bugs actually are

Spend the review budget here, in order:

1. **Output body over-exposure** — highest yield, propagates to every client.
2. **The handler/service/repo logic** — authorization, tenancy, the money path. Same as any Go review; `references/review-patterns.md` applies unchanged.
3. **Input body validation tags and mass assignment** — a missing tag is a missing control everywhere.
4. **Generator config** — token handling, TLS, provider state credentials.
5. **Spec/client drift** — a control that exists in the spec but not yet in the shipped clients.

Do not spend it reading generated `model_*` files. They are a mirror of the spec; review the spec.
