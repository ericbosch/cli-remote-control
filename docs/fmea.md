# Failure Modes and Effects Analysis (FMEA)

| Failure mode | Detection | Mitigation |
|--------------|-----------|------------|
| **WebSocket drop** | Client misses pong; server misses ping | Heartbeat (ping/pong); client reconnect with exponential backoff; reattach to same session. |
| **Duplicated output on reconnect** | User sees same output twice | Replay sent once with clear boundary (e.g. `replay` message type); client renders replay then switches to live stream; optional sequence IDs. |
| **PTY deadlock** | Process blocked on read/write, no output | Bounded buffers; kill session on timeout or explicit terminate; avoid unbounded writes to PTY. |
| **Zombie processes** | Child processes outlive session | Process group kill on terminate; reaper for orphaned engines; document cleanup in ops. |
| **Log growth unbounded** | Disk full | Log rotation (size + count); configurable max size; alert in docs. |
| **Token leak** | Token in logs, URL, or client storage | Never log token; token in header preferred over query; docs warn about screenshot/sharing. |
| **Engine adapter crash** | One engine panics or exits unexpectedly | Adapter in goroutine; session marked exited; cleanup PTY; return status to client. |
| **Rate limit bypass** | Many requests from one token | Per-token bucket; optional per-IP limit; return 429 with Retry-After. |
| **Buffer overflow (ring buffer)** | Replay larger than configured | Fixed-size ring buffer; replay only last N KB; no unbounded allocation. |
| **Concurrent attach** | Multiple clients attach same session | Allow multiple attaches; broadcast output to all; input from one or first-come (document behavior). |
