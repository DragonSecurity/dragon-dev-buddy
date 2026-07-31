# Secret patterns and rotation

## High-confidence prefixes

Format alone identifies the provider, and often the scope. Read the prefix; never authenticate.

| Prefix | Provider | Note |
| --- | --- | --- |
| `AKIA…` | AWS access key, long-lived | The dangerous kind. `ASIA…` is a temporary STS key and expires. |
| `ghp_`, `gho_`, `ghs_`, `github_pat_` | GitHub | `ghs_` is a short-lived Actions token; the others are not. |
| `glpat-` | GitLab PAT | |
| `sk_live_`, `rk_live_` | Stripe | `_test_` variants are harmless. `rk_` is restricted-scope; ask which scopes. |
| `xoxb-`, `xoxp-`, `xapp-` | Slack | Bot, user, app-level. |
| `SG.` | SendGrid | |
| `AIza…` | Google API key | Often unrestricted by default. |
| `ya29.` | Google OAuth access token | Short-lived. |
| `sk-`, `sk-ant-` | OpenAI, Anthropic | |
| `npm_` | npm automation token | Publish rights. Treat as supply-chain critical. |
| `dckr_pat_` | Docker Hub | |
| `-----BEGIN … PRIVATE KEY-----` | Any | Check for a passphrase; an unencrypted key is worse. |
| `eyJ…` | JWT | Decode the payload for `iss`, `aud`, `exp`. An expired token is not a finding. |

## Custom and generic patterns

Scanners find the table above. They miss the things your company invented. Grep for these:

```
(?i)(api[_-]?key|secret|password|passwd|token|credential|private[_-]?key)\s*[:=]\s*["'][^"']{8,}
(?i)(postgres|mysql|mongodb|redis|amqp)://[^\s"']*:[^\s"'@]+@
(?i)authorization\s*[:=]\s*["']?(bearer|basic)\s+[A-Za-z0-9+/=._-]{16,}
```

Then look at the specific files where secrets accumulate:

```
.env, .env.*, !.env.example      the obvious one, and the .example exception
docker-compose*.yml              environment: blocks
*.tfvars, terraform.tfstate      state files contain resolved secret values
k8s secrets manifests            base64 is encoding, not encryption
CI: .github/workflows, .gitlab-ci.yml, Jenkinsfile
config/*.json, appsettings*.json, application*.properties
notebooks (.ipynb)               outputs cells retain printed values
test fixtures                    "it's just a test key" is often not
README.md, docs/                 curl examples with real tokens
```

## Assessing liveness without using the credential

Permitted:
- Prefix and format analysis, as above.
- Is the file still in the working tree, or only in history?
- `git log --diff-filter=A -- <path>` for when it arrived; `git log -1 -- <path>` for when it last changed.
- Was the repository public during the exposure window? Ask; check the platform's audit log if available.
- Ask the user to check the provider dashboard: last-used timestamp, source IPs, active sessions.

Not permitted, ever:
- Sending the credential to the provider to see if it authenticates.
- Decrypting, cracking, or brute-forcing anything.
- Using it "read-only, just to check the scope."

If the user asks you to test it, explain the distinction and offer the dashboard route instead. If they insist and it is their own credential in their own account, that is their call to make — but it should be an explicit decision, not a default.

## Rotation, by provider

Order matters. Issue the new credential before revoking the old one wherever the platform allows it, or you take an outage on top of an incident.

| Provider | Steps |
| --- | --- |
| **AWS** | Create a second access key for the user → deploy it → verify → delete the first key. Then check CloudTrail for the old key's `AccessKeyId` over the whole exposure window. If it is a root key, delete it and never make another one. |
| **GitHub PAT** | Settings → Developer settings → revoke. Check Security log for actions by that token. If it had `write:packages` or repo write, review recent pushes and releases for changes nobody remembers. |
| **Stripe** | Roll the key in the dashboard (Stripe supports a grace period on roll). Review Events and Logs for the exposure window, filtered by API key. Restricted keys: check exactly which scopes, do not assume. |
| **Database password** | Create a new user with identical grants → migrate connection strings → drop the old user. Do not just change the password on a shared account; you will not know what broke. Then check connection logs for source addresses. |
| **Private key (SSH/TLS)** | Generate new, deploy, remove old from `authorized_keys` or the certificate store. For TLS, revoke the certificate; rotation without revocation leaves the old one valid. |
| **JWT signing secret** | Rotate with an overlap window: accept both old and new for the token lifetime, then drop the old. Rotating instantly logs out every user, which is sometimes correct and should be a decision. |
| **npm / package registry token** | Revoke, then check the package's publish history for versions nobody recognizes. A compromised publish token is a supply-chain incident affecting your users, not just you. |

## After rotation

Purging git history (`git filter-repo`, BFG) is optional and often not worth it:

- It rewrites every commit hash, breaking every open PR, every local clone, and every reference to a commit in your ticket system.
- It does not remove the secret from forks, from anyone's existing clone, from GitHub's cached views of old commits, or from the scrapers that already have it.

**Rotation is what makes the secret worthless. History rewriting is cosmetic.** Do it when the repository is about to be made public and the history is short. Otherwise rotate, document, and move on.

Always tell the user this explicitly, because the instinct is to reach for the history rewrite first and feel finished.
