# Security

## Defaults

- The host **binds to 127.0.0.1 only**. It is not reachable from other machines unless you change the bind address.
- **All API and WebSocket endpoints require authentication** (Bearer token). No anonymous access.
- **Logs are written only on the host** and are append-only; they do not contain auth tokens.

## Token

- Use a **strong, random token** (e.g. the one generated with `--generate-dev-token`).
- **Do not commit** `host/.dev-token` or any file containing the token (it is gitignored).
- Do not share the token or expose it in screenshots or logs.

## Exposing the host on the network

If you bind to **0.0.0.0** (e.g. to use the app from your phone on the same LAN):

- **Warning:** The service is then reachable by anyone on that network who has the token.
- Prefer a **VPN** (e.g. [Tailscale](https://tailscale.com)) so that only your devices can reach the host, and keep binding to 127.0.0.1 or the VPN interface.
- Use **HTTPS** in front of the host (e.g. reverse proxy with TLS) if you ever expose it beyond a trusted LAN.

## CORS

- The host allows same-origin and localhost origins for the web client. It does not use a wildcard for credentialed requests.

## Provider API keys

- **Never** store provider API keys (e.g. for Claude, Gemini) in the Android app or in any relay. CLIs use the host’s environment and config only.
