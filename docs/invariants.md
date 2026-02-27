# System Invariants (Non-Negotiables)

These must hold for all deployments. Code and config must enforce them.

| Invariant | Enforcement |
|-----------|-------------|
| **Default bind MUST be 127.0.0.1** | Server listens on 127.0.0.1 unless user sets explicit flag (e.g. `--bind=0.0.0.0`). |
| **All endpoints MUST require auth** | Every REST and WebSocket endpoint checks Bearer token (or M3 device token). No anonymous access. |
| **Sessions MUST survive client disconnect** | Session process and PTY keep running; reattach returns same session. |
| **Logs MUST be local-only and append-only** | Logs written only to host filesystem; no log shipping by default; append-only files, rotation by size/time. |
| **Public bind MUST require explicit flag and loud warning** | If bind address is 0.0.0.0, startup log and optionally stderr must print a clear security warning. |
| **Never store provider API keys outside the host machine** | No API keys in Android app, relay, or web client; CLIs use host env/config only. |
| **Relay (if added) MUST NOT see plaintext** | M3 relay only routes encrypted blobs; no session content or tokens in relay. |
| **No reverse engineering of closed products** | Use only official CLIs and official APIs (e.g. Ollama HTTP API). |
| **CORS locked down** | Web served by host = same origin; otherwise allowlist only; no wildcard for credentialed requests. |
