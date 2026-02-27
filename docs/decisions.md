# Key Decisions (Change-My-Mind)

For each key decision: claim, reasoning, counterargument, and what would change the decision.

---

## D1: Host in Go (not Node/Python)

- **Claim:** Host daemon is implemented in Go.
- **Reasoning:** Single binary, good PTY and concurrency support, low footprint, fits tech stack.
- **Counterargument:** Team may prefer Node/Python for consistency with web tooling.
- **Change-my-mind:** Hard requirement from spec; only switch if Go cannot meet PTY/WS/security requirements (no such evidence).

---

## D2: WebSocket per session (not one WS multiplexing all sessions)

- **Claim:** One WebSocket per session: `/ws/sessions/{id}`.
- **Reasoning:** Simple model; attach = open WS to that session; no session id in every message.
- **Counterargument:** Many sessions could mean many WS connections.
- **Change-my-mind:** If we need to support 100+ concurrent sessions per host and connection count becomes a problem, consider a single WS with session id in each message.

---

## D3: Ring buffer for replay (not full history)

- **Claim:** On attach, send only last N KB of output (ring buffer), not full history.
- **Reasoning:** Bounded memory and network; sufficient for “see what just happened.”
- **Counterargument:** User may want full scrollback.
- **Change-my-mind:** If product requires full history, add optional persistent log per session with size limit and format (e.g. line-based).

---

## D4: Auth via Bearer token only for M1/M2

- **Claim:** No OAuth or certificates for MVP; Bearer token in header (or query for WS).
- **Reasoning:** Simplest to implement and use; M3 pairing adds device-bound tokens.
- **Counterargument:** Token theft gives full access.
- **Change-my-mind:** If we must support multi-user or shared hosts before M3, add scoped tokens or OAuth.

---

## D5: Android terminal via WebView (xterm.js) in M1

- **Claim:** M1 Android app embeds WebView with xterm.js as the terminal renderer.
- **Reasoning:** Reuse web terminal; avoid building a full VT emulator in native code for M1.
- **Counterargument:** WebView may have latency or keyboard limitations.
- **Change-my-mind:** If testing shows unacceptable UX, move to native TerminalView in M2/M3.

---

## D6: Structured events optional (Event mode) in M2

- **Claim:** When engine supports structured output (e.g. JSONL), we emit `event` messages; otherwise PTY stream only.
- **Reasoning:** Preserves CLI capabilities; allows timeline/tool_call UI when data exists.
- **Counterargument:** Two code paths and UI modes add complexity.
- **Change-my-mind:** If no CLI provides usable structured output, keep only “Terminal mode” and defer Event mode.

---

## D7: Relay routes encrypted blobs only (M3)

- **Claim:** If relay is added, it only forwards opaque encrypted blobs; no plaintext session data or tokens.
- **Reasoning:** Minimizes trust in relay; host and client hold keys.
- **Counterargument:** Key exchange and key storage on mobile are hard.
- **Change-my-mind:** If key management becomes blocking, consider TLS to relay with relay not logging content, and document threat model.
