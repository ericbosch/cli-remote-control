# Verified Facts (Preflight Recon)

Evidence from commands run on the host machine. No secrets included.

## OS & Environment

| Fact | Evidence | Why it matters |
|------|----------|----------------|
| **Native Linux (not WSL2)** | `uname`: Linux, `grep -i microsoft /proc/version` empty → NATIVE_LINUX | PTY and process handling are standard Linux; no WSL-specific path or network quirks. |
| **Ubuntu 24.04 LTS** | `/etc/os-release`: VERSION_ID="24.04", PRETTY_NAME="Ubuntu 24.04.4 LTS" | Package names and systemd behavior; docs can target Ubuntu/Debian. |
| **Kernel 6.17.0** | `uname -r`: 6.17.0-14-generic | Modern kernel; PTY and namespace support assumed. |
| **Architecture x86_64** | `uname -m`: x86_64 | Binaries and Android emulator targets; M1/M2 note in spec refers to future Mac support. |

**Commands run:**
```bash
uname -a
cat /etc/os-release
grep -i microsoft /proc/version || echo "NATIVE_LINUX"
```

## Runtimes

| Fact | Evidence | Why it matters |
|------|----------|----------------|
| **Go** | `go version` → command not found at recon time | Host daemon is Go; must be installed for build/run. Document install in setup-linux.md. |
| **Node.js** | `node --version` → v22.22.0 | Web client (Vite) and tooling; LTS-compatible. |
| **npm** | `npm --version` → 10.9.4 | Web deps and scripts. |
| **Java** | `java -version` → openjdk 17.0.18 | Android build (Gradle/Kotlin); 17+ required. |

**Commands run:**
```bash
go version
node --version
npm --version
java -version
```

## CLI Binaries (AI / Shell)

| CLI | Path | Version / status | Why it matters |
|-----|------|------------------|----------------|
| **codex** | `/home/krinekk/.npm-global/bin/codex` | codex-cli 0.104.0 | M2 Codex adapter; check for `--json` / app-server. |
| **cursor** | `/home/krinekk/.local/bin/cursor` | Binary exists; `cursor --version` may require IDE/agent context | M2 Cursor adapter; fallback to PTY if no stream-json. |
| **claude** | `/home/krinekk/.local/bin/claude` | 2.1.59 (Claude Code) | M2 Claude adapter; look for --output-format json/stream-json. |
| **gemini** | `/home/krinekk/.npm-global/bin/gemini` | 0.29.5 | M2 Gemini adapter; stream-json or json. |
| **ollama** | not in PATH | not found | M2 Ollama adapter uses HTTP API (optional local install). Shell/PTY does not depend on it. |

**Commands run:**
```bash
which codex cursor claude gemini ollama
codex --version; cursor --version; claude --version; gemini --version
```

**Design impact:** If any CLI is missing, implement adapter with mock/fallback and document how to enable the real CLI. Do not block M1 (shell-only) or M2 (multiple engines with fallbacks).

## Android SDK

| Fact | Evidence | Why it matters |
|------|----------|----------------|
| **SDK present** | `~/Android/Sdk` exists with build-tools, cmdline-tools | Android app can be built; may need ANDROID_HOME set. |
| **ANDROID_HOME** | Not set in recon shell | Docs should instruct setting ANDROID_HOME to ~/Android/Sdk (or install location). |

**Commands run:**
```bash
echo $ANDROID_HOME
ls ~/Android/Sdk
```

## Network Context

| Assumption | Note |
|------------|------|
| **localhost** | Default bind 127.0.0.1; phone connects via same LAN or VPN (e.g. Tailscale). |
| **No cloud** | All services run on this host; no dependency on external relay for M1/M2. |

(Network facts are environment-specific; recon did not change any network config.)
