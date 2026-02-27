# Setup (Linux)

## Prerequisites

- **Go 1.21+** — host daemon  
  ```bash
  sudo apt install golang-go   # or: sudo snap install go
  go version
  ```
- **Node.js 18+** — web client build  
  ```bash
  node --version
  ```
- **Java 17+** — Android app build  
  ```bash
  java -version
  ```
- **Android SDK** — for building the Android app  
  Set `ANDROID_HOME` (e.g. `export ANDROID_HOME=$HOME/Android/Sdk`).

## Build & run host

```bash
cd host
go build -o rc-host ./cmd/rc-host
./rc-host serve --generate-dev-token
```

Copy the printed dev token and use it in the web UI and Android app (Settings).

To serve the built web app from the host:

```bash
./scripts/build.sh   # builds web then host
./host/rc-host serve --web-dir=web/dist
```

Then open http://127.0.0.1:8765 and enter the token in Settings.

## Dev (host + web)

```bash
./scripts/dev.sh
```

Starts the host and Vite dev server. Use the web UI at the Vite URL (e.g. http://localhost:5173); set base URL to the host if needed (or rely on Vite proxy).

## Android

1. Set **Server URL** in the app to your PC’s address (e.g. `http://192.168.1.100:8765`).  
   For emulator use `http://10.0.2.2:8765`.
2. Set **Auth token** to the same token used by the host.
3. Create a session and tap **Attach** to open the terminal in the WebView.

## Binding to all interfaces (e.g. for phone on LAN)

```bash
./rc-host serve --bind=0.0.0.0
```

**Warning:** This exposes the service on your network. Use a strong token and prefer a VPN (e.g. Tailscale) for remote access.
