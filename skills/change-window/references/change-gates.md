# Change gates

## Thresholds

| Gate | Pass | Warn | Fail (no-push) |
| --- | --- | --- | --- |
| Intent vs running | candidate diffed against running; only expected deltas | drift present, explained, and the change preserves it | unexplained drift on the target; running config could not be read |
| Self-severance | change touches nothing it travels over | touches a severance path, timer armed, out-of-band tested | touches a severance path with no timer, or out-of-band untested/absent |
| Rollback | automatic timer, or a manual revert reachable out-of-band | manual revert, in-band only, low blast radius | no mechanism named; revert depends on the change having worked |
| Blast radius | one device, or a class with a healthy standby | many devices, understood, staged | both halves of a redundant pair in one window; peer unhealthy or unverified |
| Ordering | additive before subtractive; no broken intermediate | mismatch window named and short | old path removed before the new one is up; symmetric change one end only |
| Stateful teardown | nothing stateful crosses the change, or the drop is accepted | sessions drop, users warned, timed for low traffic | mid-day teardown of established sessions nobody has been told about |
| Observability | a check that does not traverse the change | monitoring degraded during the window, understood | the only reachability check runs through the interface being changed |

Warns accumulate. Three warns on an unscheduled edge change with one engineer awake is a no-push even though no single gate failed. Say when you are escalating on accumulated risk rather than a single failure.

## The self-severance catalogue

The question is always the same — *does this change touch anything the change itself, or its rollback, depends on?* What changes is where that dependency hides.

**Any managed device**
- The management interface, its address, or its VRF.
- The ACL, policy or `permit` that lets your source address reach the management plane.
- The route back to your workstation, jump host or orchestrator — including a default route you are about to replace.
- The AAA / TACACS+ / RADIUS server, its shared secret, or the authorization policy for the next command. A working session that can no longer authorize a command is a soft lockout, and it looks exactly like a hard one at 02:00.
- SSH or API service config: host keys, ciphers, listeners, rate limits, source restrictions.
- The uplink or trunk carrying management traffic, and anything that flaps it — speed, duplex, VLAN membership, spanning-tree role, LACP.

**Routers and switches**
- Routing policy or filters that could withdraw the prefix your management address lives in.
- Redistribution or a route-map applied in the direction of the management path.
- Spanning-tree parameters that can trigger a topology change and a brief blackhole.
- VLAN pruning, native VLAN changes, or an allowed-list on a trunk that carries management.

**Firewalls**
- The policy allowing your administrative session — most acutely when the ruleset is being reordered, replaced wholesale, or migrated between policy models.
- Zone or interface reassignment, which silently re-evaluates every rule that referenced them.
- NAT changes affecting the management path.
- The implicit default action, if a rewrite changes where the deny lands.

**Load balancers and reverse proxies**
- The listener or virtual server the management UI is published behind.
- Certificate or TLS profile changes that affect the admin interface.
- Health-check config that can mark the pool down and take out the path you administer through.

**Cloud and orchestrated infrastructure**
- Security-group or NACL rules covering the bastion, the VPN endpoint, or the CI runner performing the change.
- Route tables, transit-gateway attachments, peering, and endpoint policies on the management path.
- The IAM role the automation itself assumes. Revoking your own permission mid-apply is the cloud form of the same mistake.
- For Kubernetes: the NetworkPolicy or admission control governing the API server path, and the CNI itself.

**DNS**
- The record the orchestrator, the AAA server, the syslog target or the certificate validation resolves through. DNS severance is delayed by the TTL, which makes it worse: the change looks fine for an hour and then everything fails at once.

## Rollback mechanisms and their semantics

Automatic beats manual, because automatic does not require you to still have reachability.

| Platform | Mechanism | Semantics worth knowing |
| --- | --- | --- |
| Junos | `commit confirmed <minutes>` | Reverts unless a second `commit` lands inside the window. The safest form on this list — arm it, then confirm. `rollback 1` recovers the previous config if you still have a session. |
| Cisco IOS / IOS-XE | `reload in <minutes>`, or `configure replace` from a saved file | `reload in` is coarse — it reboots rather than reverts, so it costs an outage on top of the failure. `archive` plus `configure replace` is the cleaner path but needs the archive set up *before* the change. |
| Cisco IOS-XR / NX-OS | commit with checkpoint / rollback to checkpoint | Take the checkpoint explicitly first; do not assume one exists. |
| Arista EOS | `configure session` with `commit timer` | Candidate config is staged, applied on commit, reverted if the timer expires unconfirmed. Same shape as Junos. |
| PAN-OS | candidate commit; `revert to running` | The candidate is uncommitted, so pre-commit revert is free. After commit, roll back to a saved named config version. |
| FortiOS | `execute backup`, revert by restore; some builds support a config-revert timer | Restore is a reboot-class operation on many models — treat it as coarse. |
| Linux packet filter | `iptables-apply` / a scheduled `nft -f` of the previous ruleset | The timer pattern implemented by hand: schedule the restore, cancel it on success. Cheap and effective; use it. |
| Terraform | previous state plus a targeted revert apply | Not a rollback in the atomic sense. A destroyed resource does not come back with the same identity, and some providers reorder on re-apply. Check the plan for `destroy` before treating revert as symmetric. |
| Kubernetes | `kubectl rollout undo` | Clean for workloads. Does not undo CRD, webhook, or CNI-level changes, which are exactly the ones that sever the API path. |
| Ansible / orchestration | re-run against the previous committed template | Only a rollback if the previous state is genuinely captured and the run is idempotent. If the play was not idempotent going forward, it is not idempotent going back. |

**What no rollback undoes:** a route already withdrawn from a peer you do not control, a DNS record cached downstream for its TTL, sessions and translations torn down, a certificate already presented, an alert already paged, a neighbor that reset and re-converged, keys already rotated on one side.

## Stateful teardown catalogue

Name which of these the change disturbs, and how long recovery takes:

- **Firewall sessions and NAT translations** — cleared on policy replacement, zone reassignment, or failover. Long-lived connections (database pools, SSH, streaming, VPN) die; short HTTP requests mostly do not notice.
- **Routing adjacencies** — BGP, OSPF, IS-IS resets and their convergence time. Graceful restart and BFD change the numbers substantially in both directions; know which is on.
- **Layer-2 state** — MAC tables, ARP and neighbor caches, spanning-tree reconvergence. Usually seconds, occasionally a blackhole if a topology change coincides.
- **Tunnel state** — IPsec security associations, GRE keepalives, WireGuard handshakes. Rekey time is not zero.
- **Load balancer state** — established connections, session affinity, health-check state machine. Draining is the difference between a graceful change and a visible one.
- **DHCP leases** — a scope or relay change can strand clients until lease renewal, which may be hours.
- **Multicast** — IGMP/PIM state rebuilds are slow and visible to exactly the applications least tolerant of it.

## Ordering rules

- **Additive before subtractive.** Add the new permit, prove traffic uses it, then remove the old. Applies to routes, rules, NAT entries, peers, pool members and certificates alike.
- **Symmetric settings have a mismatch window.** MTU, encapsulation, authentication keys, timers, LACP mode, VLAN tagging — one end is changed before the other, and between the two the link may be down or, worse, up and silently dropping. Name the window and keep it short; where the platform allows accepting both old and new simultaneously, use that instead.
- **Never both halves of a pair.** Change the standby, verify, fail over, verify, then change the former active. A window that touches both is not staged, it is scheduled downtime with extra steps.
- **Soak the first device.** For a fleet, apply to one device of the least critical class, wait long enough for the failure mode you are worried about to appear — including the slow ones, like a leak or a table that fills — then proceed. "Applied to all forty in ninety seconds" is a rollout, not a staged rollout.
- **Order by criticality, ascending.** Access before distribution, distribution before core, leaf before spine, non-revenue path before revenue path.
- **Independent variables, separate windows.** A firmware upgrade and a config change in one window means a failure tells you nothing about which caused it.

## Change shapes that warrant extra caution

Not blockers, but each turns nearby warns into no-pushes faster:

- **Ruleset or policy replacement rather than an edit.** Wholesale replacement re-evaluates everything, including the rules nobody remembers relying on.
- **A change delivered in-band with no tested out-of-band path.** The single most common way a routine change becomes a site visit.
- **First change after a long freeze.** The running config has drifted further than anyone thinks, and the muscle memory is stale.
- **A change to the thing that would tell you it broke** — monitoring, syslog, the collector, the metrics path.
- **Automation applying to a class rather than a device.** The blast radius is whatever the inventory query returned, and nobody has read that query lately. Print the resolved target list before applying, always.
- **A change nobody else is awake for.** Not a fail on its own; it raises the bar on rollback, because there is nobody to call.
- **Anything on the path that carries the rollback for a *different* in-flight change.** Two overlapping windows is how a recoverable problem becomes an unrecoverable one.
