# Host contract (Android client)

This app targets the existing host API/WS contract and must not break the Web UI.

## Base URL + auth

- Base URL example (tailnet): `https://<device>.ts.net:8443`
- REST auth: `Authorization: Bearer <token>`
  - Token comes from the host token file (`host/.dev-token`) but the Android app stores the token locally.
  - Never include the bearer token in any logs or URLs.

## REST endpoints

### List sessions

- `GET /api/sessions`
- Response: JSON array of session objects:
  - `id`, `name`, `engine`, `state`, `exit_code`, `last_seq`, `created`

### Create session

- `POST /api/sessions`
- Body:
  - `engine`: `shell | codex | cursor`
  - `name` (optional)
  - `workspacePath` (optional)
  - `prompt` (optional)
  - `mode` (optional; mainly for cursor)
  - `args` (optional object)

### Issue WS ticket

- `POST /api/ws-ticket`
- Response: `{ "ticket": "<short-lived single-use>", "expires_ms": <epoch_ms> }`

## WebSocket (events + input)

### Attach to session event stream

- `GET /ws/events/{sessionId}?ticket=<ws-ticket>&last_n=256`
- Reconnect: use `from_seq=<last_seq+1>` to continue without gaps.

### Client → server messages

JSON objects:

- Input: `{ "type": "input", "data": "<text>" }`
- Resize: `{ "type": "resize", "cols": <int>, "rows": <int> }` (optional for Android)

### Server → client event schema

Each message is a JSON object:

```
{
  "session_id": "abc",
  "engine": "shell|codex|cursor",
  "ts_ms": 1234567890,
  "seq": 42,
  "kind": "assistant|status|error|thinking_delta|thinking_done|tool_call|tool_output|user",
  "payload": { ... }
}
```

Key `kind` payload shapes (non-exhaustive):

- `assistant`: `{ "data": "<text>", "stream": "stdout" }` (stream may be present)
- `status`: `{ "state": "running|attached|exited", "exit_code": 0 }`
- `error`: `{ "message": "<text>" }`
- `thinking_delta`: `{ "delta": "<text>" }`

## Input semantics (default)

- Terminal output is read-only by default.
- Composer sends full lines/messages on Enter/Send (PTY engines append `\\n`).
- Raw mode is optional and **off** by default.

