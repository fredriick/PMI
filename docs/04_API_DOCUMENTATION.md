# Proxy Mesh API v1.0
## Connection Format
`http://[user]-[targets]:[pass]@gateway.io:8000`
## Target Modifiers
- `-country-[code]`: e.g., `-country-us`
- `-city-[name]`: e.g., `-city-london`
- `-session-[id]`: Keep the same IP for subsequent requests.
## Success Headers
- **X-Request-ID:** Correlation ID generated from the configured prefix and format, or echoed from the incoming request.
- **X-Proxy-Node-ID:** ID of the exit peer for troubleshooting.
- **X-Proxy-Latency:** Time taken for the exit node to respond.

## Request ID Configuration
- `GET /api/admin/request-id` returns the active prefix and format.
- `POST /api/admin/request-id` accepts `{"prefix":"pm","format":"unix|timestamp|uuid"}`.
