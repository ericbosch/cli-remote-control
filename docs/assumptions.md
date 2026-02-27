# Assumptions

Assumptions made for the MVP. Risk and quick validation for each.

| # | Assumption | Risk if wrong | How to validate quickly |
|---|------------|---------------|-------------------------|
| 1 | User has or will install Go (1.21+) on the host | Cannot build/run rc-host | `go version` |
| 2 | Phone and PC share a network (same LAN or VPN) when not using relay | Phone cannot reach host on 127.0.0.1 | Use 0.0.0.0 + token + VPN or Tailscale for phone access |
| 3 | Bearer token is sufficient for MVP auth (no OAuth) | Weak if token leaks; acceptable for local/VPN | Document token storage and revocation (M3 pairing improves this) |
| 4 | One human operator (user) per host | No multi-user RBAC needed for MVP | Out of scope per spec |
| 5 | All AI CLIs can be run in a PTY for fallback | Some CLI may not be PTY-friendly | Test each engine in PTY; document quirks |
| 6 | codex/cursor/claude/gemini may offer JSON/stream-json flags | If not, we use PTY-only for that engine | Check each CLI --help and docs in M2 |
| 7 | Ollama HTTP API is stable and local-only | API change could break adapter | Pin adapter to documented Ollama API; test with local Ollama when available |
| 8 | WebView on Android can render xterm.js well enough for M1 | UX may be imperfect (zoom, keyboard) | Manual test on device; M3 can add native TerminalView |
| 9 | Session “name” is optional and can be auto-generated | No product requirement for named sessions | Implement optional name in POST /api/sessions |
| 10 | Ring buffer replay on attach is “last N KB” (e.g. 64–256 KB) | Very long sessions might lose older output | Configurable buffer size; document in troubleshooting |
