# Architecture (Host event stream contract)

This doc describes the **canonical event stream** contract used for reconnect/replay across engines.

## Canonical events

The host defines a stable JSON event type in `host/internal/events`:

- `SessionEvent` fields: `session_id`, `engine`, `ts_ms`, `seq`, `kind`, `payload`
- `seq` is monotonic per session.
- Events are persisted as JSONL under `host/.run/sessions/<session_id>.jsonl` (local-only).

## WebSocket (v2 canonical stream)

Endpoint:

- `GET /ws/events/{session_id}`

Auth:

- `/ws/*` requires the Bearer token.
- Browsers should **not** put the Bearer token in the WebSocket URL. Instead:
  - `POST /api/ws-ticket` (Authorization: `Bearer <token>`) returns `{ticket, expires_ms}`
  - Connect WS using `?ticket=...` (short-lived, single-use)

Replay controls (query params):

- `from_seq=<uint>`: replay events with `seq > from_seq`, then live tail
- `last_n=<int>`: replay last N events, then live tail
- If neither is provided, the server replays a default tail (currently 256) then live tail.

Wire format:

- Server → client: each message is a single `SessionEvent` JSON object.
- Client → server: legacy control messages are accepted for input/resize:
  - `{"type":"input","data":"..."}`
  - `{"type":"resize","cols":<int>,"rows":<int>}`
