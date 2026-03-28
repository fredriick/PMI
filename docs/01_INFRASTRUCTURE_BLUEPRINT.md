# Proxy Mesh Infrastructure Blueprint
## 1. Network Architecture
The system uses a **Hybrid Mesh** to balance cost and stealth.
* **Datacenter Tier:** IPv6 /64 subnets for high-volume, low-priority targets.
* **Residential Tier:** P2P exit nodes (Home Wi-Fi) for high-trust targets.
## 2. The Matchmaker (Go + Redis)
The Matchmaker is the "Brain" that selects the best IP for every request.
### Redis Schema
- `nodes:[country_code]` (Sorted Set): Score = Reputation, Member = NodeID.
- `node_meta:[node_id]` (Hash): Stores ISP, Battery, OS, and Last Seen.
- `cooldown:[target_domain]` (Set): IPs currently blocked by a specific site.
### Selection Logic
1. Receive request with targeting (e.g., -country-us).
2. Query Redis for top candidates in `nodes:us` with high reputation.
3. Filter out nodes present in `cooldown:[target]`.
4. Randomly pick one to ensure even distribution and avoid hotspots.
