# Gateway Implementation Guide
## Core Proxy Logic (Golang)
This service handles incoming user requests and tunnels them to the selected exit node.
### Logic Flow
- **Authenticate User:** Verify API key against the billing database.
- **Extract Targeting:** Parse strings like `customer-us-city-ny`.
- **Fetch Node:** Query the Matchmaker for an active peer.
- **Establish Tunnel:** Use mTLS to securely forward the request.
## Security Checklist
- [ ] **mTLS:** Encrypt all traffic between Gateway and Peer.
- [ ] **IP Binding:** Bind outgoing IPv6 requests to random addresses in the /64 block.
- [ ] **Circuit Breaker:** Automatically drop nodes that return 5xx errors consistently.
