# Ruleset analysis

## Path enumeration checklist

Two zones are almost never connected by one path. Walk this list explicitly and record each as present, absent, or not assessed.

**Data plane**
- The routed path through the firewall or ACL everyone thinks of.
- A second routed path — a backup circuit, an old link left up after a migration, a route that only appears when the primary fails.
- Asymmetric routes, where traffic leaves via the inspected path and returns via an uninspected one. Stateful firewalls drop this; stateless ACLs happily permit it, which is why the traffic still works and nobody investigates.
- Layer 2 adjacency: a VLAN trunked to both zones, a stretched L2 domain, a mis-trunked port. A shared broadcast domain means the firewall is not in the path at all.

**Management and infrastructure**
- The management VLAN or out-of-band network, which frequently reaches every device in every zone by design.
- Hypervisor management (vCenter, Proxmox, oVirt) and the storage fabric. A VM in the low zone and a VM in the high zone on the same host share more than the network diagram shows.
- Backup and replication networks. Backup servers routinely hold credentials for both zones and a path to each.
- Console servers, IPMI/iDRAC/iLO, PDUs. Rarely segmented, universally powerful.

**Identity and shared services**
- The directory, AAA server, or secrets manager both zones authenticate against. A boundary that both sides trust for identity is a path for anything that compromises it.
- Shared DNS, NTP, PKI, log collectors, monitoring agents. Agent traffic is usually outbound from both zones to one collector, which makes the collector a bridge.
- Certificate authorities and the systems that issue to both zones.

**Automation and human paths**
- CI/CD runners with credentials or network reach into both. The single most common modern bypass — the pipeline is on a flat network by default and deploys to everything.
- Jump hosts, bastions, and admin workstations with routes into both.
- Configuration management servers, orchestrators, and the API endpoints of the devices themselves.
- VPN clients with split tunnelling, and remote-access policies that assign the same address pool regardless of role.

**Cloud and container**
- VPC peering, transit gateway attachments, VPN and Direct Connect / ExpressRoute circuits.
- Shared VPC endpoints, PrivateLink services, and NAT gateways.
- IAM as a network-adjacent path: a role assumable from one account into the other reaches resources no packet could.
- Kubernetes: pods in the same cluster are mutually reachable unless a NetworkPolicy says otherwise; nodes route between namespaces freely; a service mesh may be enforcing policy the network is not, or vice versa. Also check whether the CNI in use actually implements NetworkPolicy — several do not, and the policies apply silently to nothing.

**People and process**
- A shared credential, or a person with access to both zones from one workstation. Not a network path, and it defeats one; note it and move on rather than pretending the network review covers it.

## Per-platform ruleset semantics

Getting these wrong produces an analysis that is confident and incorrect.

| Platform | Match model | Default for unmatched | State | The mistake it invites |
| --- | --- | --- | --- | --- |
| Cisco IOS ACL | First match, top-down | Implicit `deny ip any any`, not shown in the config | Stateless unless reflexive/CBAC | Return traffic is separately permitted, so a "deny inbound" boundary is often permitted by the outbound-return rule. Also: an ACL configured but never applied to an interface enforces nothing. |
| Junos firewall filter / security policy | First match; filters terminate on `then accept/discard` | Filter: implicit accept in some contexts — check. SRX policy: implicit deny | Filters stateless; SRX policies stateful | Zone-based policy means moving an interface between zones silently re-evaluates every rule referring to that zone. |
| PAN-OS | First match, top-down | Implicit intrazone-allow and interzone-deny at the bottom, hidden unless shown | Stateful | Rules are zone-based, and `any` in the source zone field means any *zone*, which is broader than most readers assume. |
| FortiOS | First match, top-down within the interface pair | Implicit deny | Stateful | Policy ordering is per interface pair, so the "top" of the list is not global. |
| AWS Security Group | All rules evaluated; allow-only, no deny | Deny (nothing is permitted unless a rule allows) | Stateful — return traffic is automatic | A group referencing another group permits *every instance in it*, present and future. Membership changes silently expand the rule. |
| AWS Network ACL | Numbered, first match, separate inbound and outbound | Explicit final deny | **Stateless** — return traffic needs its own rule | The stateless/stateful mismatch with security groups is the classic error; ephemeral port ranges must be allowed back. |
| Azure NSG | Priority-ordered, first match | Default rules permit VNet-to-VNet and load-balancer traffic before the final deny | Stateful | The default `AllowVnetInBound` rule permits everything within the VNet at priority 65000; an NSG that "denies" without overriding it does not. |
| GCP firewall | Priority-ordered, allow and deny both expressible | Implied allow-egress, deny-ingress | Stateful | Rules apply by target tag or service account; an instance without the tag is not covered, and nothing warns you. |
| Kubernetes NetworkPolicy | Additive allow; policies union | No policy selecting a pod = **allow all** | Stateful via CNI | Selecting a pod with an empty ingress rule denies all to it; not selecting it at all permits everything. The two look similar and mean opposites. Policies are namespace-scoped, so a policy in the wrong namespace does nothing. |
| Istio / service mesh | Layer 7 authorization policy, additive | Depends on mesh mode; permissive mTLS accepts plaintext | Application layer | Mesh policy governs sidecar-to-sidecar traffic only. Anything bypassing the sidecar — host network pods, direct node access — is unaffected. |
| iptables / nftables | First match within a chain, chains traversed in order | Chain policy (`ACCEPT` or `DROP`) | Stateful via conntrack | An early `ESTABLISHED,RELATED` accept is correct and also means new-connection rules below it never see existing flows. Rule order across chains and tables is where errors hide. |

## The four decay patterns and how to detect them

**Overly broad**
- Grep for `any`, `0.0.0.0/0`, `::/0`, `ALL`, protocol `ip`/`any`, and port ranges wider than the service needs.
- Compare each rule's scope against the service it exists for: a rule permitting a `/16` for one database server is 65,534 addresses of unnecessary reach.
- Check object groups and address sets by expanding them. A rule that reads narrowly can reference a group that grew to hundreds of entries.
- In cloud: a security group referencing another group, or a rule permitting the VPC CIDR, is broad by construction even though it reads specific.

**Orphaned**
- Resolve every source and destination. Does the address still exist, is it still assigned to what the rule name claims, is the DNS name still owned by you?
- **Then check whether it was reissued.** A rule permitting a decommissioned host's address permits whatever DHCP or the cloud handed that address to next. This is the finding that turns a housekeeping item into an exposure.
- Where the platform records hit counts, find rules with zero hits over a long window — strong evidence of orphaned, weak evidence of safe to remove (an annual process may be the only user).
- Where the platform records rule creation or a ticket reference, find rules with no attributable owner. "Nobody knows why this exists" is a reportable state.

**Shadowed**
- Compare each rule against all rules above it: is its match space fully covered by an earlier rule with the same or different action? Most vendors have a built-in checker — use it rather than doing this by hand on a large ruleset.
- A shadowed *permit* under a *deny* means someone believes something is allowed that is not; expect a workaround elsewhere. A shadowed *deny* under a *permit* means a control someone thinks is enforced is not.

**Implicit reliance**
- Establish the default action for unmatched traffic on every enforcement point, from the table above rather than from the config, since most platforms do not print it.
- Ask whether the boundary depends on it. A boundary held up by a final implicit deny is fine; one held up by "we assume nothing routes there" is not a boundary.
- Check for rules *below* the effective deny — they are dead, and their existence means the ruleset has been reordered or migrated without review.

## Evidence, ranked from safest

1. **Config reading.** Always available, never disruptive, and insufficient on its own — it tells you what the device is configured to do, not what the network does.
2. **Platform policy lookup.** Non-intrusive and much stronger than reading: PAN-OS `test security-policy-match`, FortiOS `diagnose firewall iprope lookup`, Junos `show security match-policies`, AWS VPC Reachability Analyzer and IAM/Network Access Analyzer, Azure Network Watcher IP flow verify, GCP connectivity tests. These answer "would this specific packet be permitted" using the device's own evaluation engine, which catches misapplied policies and zone errors that reading misses.
3. **Existing telemetry.** Flow logs, connection logs, firewall session tables, service mesh telemetry. Shows what has *actually* crossed the boundary — which occasionally reveals a path the config analysis did not predict, and is the strongest evidence of all when it does.
4. **Active reachability testing.** Sending packets. The only method that proves the whole path end to end, and the only one that changes anything.

## The authorization gate for active testing

Reading config and querying policy engines is observation. Sending packets across a boundary is action, and it is the point where a review can become an unauthorized test.

- **The user's own infrastructure:** confirm the environment (prod versus a lower one), confirm the user has the authority to authorize it, and agree what you will send. Then proceed. Prefer a lower environment where the ruleset is equivalent, and say so if it is not equivalent.
- **Anything belonging to a client or third party:** check `engagement.authorized_scope` and `engagement.out_of_scope`. Without a recorded scope and an `authorization_ref`, do not send traffic. Say plainly that the finding is config-derived and unverified rather than testing anyway or quietly softening the claim.
- **Either way, keep it proportionate.** Proving a boundary is open needs one connection to one port, logged and reported. It does not need a scan, and a scan across a segmentation boundary tends to page someone.
- **Record what you sent, from where, and when.** If it trips an alert, the responder needs to identify it as yours within a minute — otherwise your review becomes their incident.
