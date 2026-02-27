# Threat Model (STRIDE)

| Threat | Mitigation |
|--------|------------|
| **Spoofing** | Bearer token (and M3 device tokens) authenticate clients; token in header/query, not logged. |
| **Tampering** | HTTPS when not localhost (recommend reverse proxy); WS over WSS on same; no relay plaintext in M3. |
| **Repudiation** | Append-only logs; session lifecycle events logged without auth material. |
| **Information disclosure** | Bind 127.0.0.1 by default; logs never contain tokens; no keys off host. |
| **Denial of service** | Rate limiting (token bucket); connection/session limits per token; bounded ring buffers; log rotation. |
| **Elevation of privilege** | Host runs engine processes with same user as daemon; no setuid; no execution of arbitrary remote code beyond configured engines. |

## Additional mitigations

- Explicit `--bind=0.0.0.0` plus startup warning when exposing to network.
- Recommendation: use VPN (e.g. Tailscale) for remote access instead of opening ports.
- Revocation: M3 device trust list on host; token rotation via new pairing.
