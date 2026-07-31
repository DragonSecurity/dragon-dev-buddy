# Insecure defaults by platform

Every entry: what to look for, and what an attacker gets from it. A finding without the second half is a style note.

## Docker

| Look for | Consequence |
| --- | --- |
| No `USER` directive | Container runs as root. A container escape or a write primitive becomes host-level. |
| `ADD` with a URL | Fetches and extracts at build time, unverified. Supply chain, silently. |
| Secrets in `ENV` or `ARG` | Baked into an image layer forever. `docker history` reveals them; so does anyone who pulls the image. |
| `latest` base image tag | Unreproducible builds. You cannot tell what shipped, and you cannot tell what changed. |
| Build context includes `.git` | The whole history, including anything ever committed, ships inside the image. |
| No `HEALTHCHECK` | Orchestrator keeps routing to a broken container. Availability, not confidentiality. |
| `--privileged` in compose or run | Effectively no isolation. Container escape is a formality. |
| Docker socket mounted into a container | That container is root on the host. Full stop. |

## Kubernetes

| Look for | Consequence |
| --- | --- |
| No `securityContext`, or `runAsRoot` | Same as Docker, at fleet scale. |
| `allowPrivilegeEscalation: true` (default) | setuid binaries inside the pod become an escalation path. |
| Secrets as env vars rather than mounted files | Leak into crash dumps, `/proc`, child processes, and observability tooling. |
| No `NetworkPolicy` | Every pod can reach every other pod. One compromised pod reaches the database directly. |
| `ServiceAccount` with cluster-admin, or automounted by default | A compromised pod owns the cluster. |
| No resource limits | One pod starves the node. Denial of service by accident or on purpose. |
| Secret manifests committed to git | base64 is encoding. Treat as plaintext, always. |

## Terraform and AWS

| Look for | Consequence |
| --- | --- |
| S3 bucket without `block_public_acls` / `block_public_policy` | The classic public bucket. One misapplied ACL and it is indexed. |
| Security group with `0.0.0.0/0` on anything but 80/443 | SSH, RDP, or a database exposed to the internet. Scanned within hours. |
| IAM policy with `"Action": "*"` or `"Resource": "*"` | Any compromise becomes total. Absence of least privilege is the finding. |
| RDS `publicly_accessible = true` | Database reachable from the internet. Now only a password stands between. |
| Unencrypted EBS, RDS, or S3 | Fails most compliance regimes outright, independent of exploitability. |
| No versioning or MFA-delete on state buckets | Ransomware deletes your infrastructure state and you cannot roll back. |
| Secrets in `.tfvars` or committed state | State files contain **resolved** values, including generated passwords. |
| No CloudTrail, or CloudTrail not multi-region | After an incident you cannot answer what happened. Every IR runs blind. |

## CI/CD

| Look for | Consequence |
| --- | --- |
| Actions pinned to a tag, not a commit SHA | Tags are mutable. Moving one changes what runs in a pipeline that holds deploy credentials. |
| `pull_request_target` with a checkout of the PR head | Untrusted code runs with repository secrets. This is the single most exploited CI pattern. |
| Secrets available to fork PR workflows | Any stranger who opens a PR can exfiltrate them. |
| No branch protection on the deploy branch | One force-push deploys anything. |
| Deploy credentials as long-lived static keys | Use OIDC federation instead; nothing to leak. |
| `echo` or `set -x` around secret-bearing commands | Secrets in build logs, which are usually more widely readable than the repo. |

## Web framework and server

| Look for | Consequence |
| --- | --- |
| Debug mode on in production | Stack traces, config dumps, and in Flask/Django an interactive console. |
| `DEBUG`, `/metrics`, `/actuator`, `/debug/pprof` reachable unauthenticated | Internal state, sometimes credentials, sometimes remote execution. |
| Source maps deployed | Your original source, comments included. |
| `Access-Control-Allow-Origin: *` **with** credentials | Any site can make authenticated requests as your users. |
| Session cookie missing `HttpOnly`, `Secure`, `SameSite` | XSS becomes session theft; CSRF becomes possible. |
| No `Content-Security-Policy` | Nothing constrains an injected script. Defence in depth, but the cheapest available. |
| Default admin credentials, or a default JWT secret from a tutorial | Search engines index the tutorial. So do attackers. |
| Verbose server headers | Version disclosure. Low, but free to fix. |

## Databases

| Look for | Consequence |
| --- | --- |
| Application connects as owner or superuser | SQL injection escalates from data theft to schema destruction and, on some engines, code execution. |
| No TLS on the connection | Credentials and data readable by anything on the path. |
| No row-level security in a multi-tenant schema | Tenancy depends on every query being written correctly, forever. |
| Backups unencrypted, or restorable by anyone | The backup is a full copy with weaker access controls than production. |
| No connection limit per user | One runaway client exhausts the pool for everyone. |

## How to report a config finding

Name the file and line, the current value, the target value, and the consequence *for this system*:

> **C3 — Container runs as root** — Medium
> `Dockerfile:1-14` — no `USER` directive, so the process runs as uid 0.
> **Consequence:** ledger-api parses untrusted JSON and writes uploaded files to disk. A write-primitive bug that would otherwise be contained becomes host-level on the Fly machine, which also runs the export worker with S3 credentials in its environment.
> **Fix:** add `RUN adduser -D app` and `USER app` before `CMD`. Verify the app can still bind its port — 8080 is above 1024, so it can.

The generic version ("containers should not run as root") is true everywhere and acted on nowhere.
