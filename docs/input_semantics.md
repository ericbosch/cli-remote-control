# Input semantics (canonical)

Goal: a "Cloud Remote Control" UX where the stream view is read-only by default and input is sent as full lines/messages from a composer.

## Expected behavior (default)

- Terminal/stream view is **read-only** and does not capture keyboard input by default.
- Input is sent **only** from the composer:
  - Enter: send the current line/message (adds a trailing `\n` for PTY-like sessions such as `shell`)
  - Shift+Enter: insert a newline into the composer
  - Send button: same as Enter
- No optimistic "echo" is rendered per keystroke; the shell naturally echoes commands as output.

Optional:

- Raw mode can be enabled explicitly to send keypresses to interactive programs (vim/top/etc). Raw mode is **off** by default.

## Prior bug (repro note)

When the terminal view captured keyboard input (xterm `onData`), the host published `user` events for every input write.
If the UI rendered those `user` events into the terminal, it produced per-character spam like:

- `[you 12:34:56] s`
- `[you 12:34:56] l`

Fix: keep the terminal view read-only by default and never render `user` events as terminal output.

