# Android (thin wrapper)

This repo includes a minimal Android client that wraps the web UI in a `WebView`.

## What it does

- Stores **Base URL** and **Bearer token** locally (SharedPreferences).
- Loads the web UI from the configured Base URL.
- Injects `rc-token` and `rc-base` into `localStorage` so the web UI can authenticate.

## Build (debug)

Prereqs:

- Android SDK + platform tools (via Android Studio)
- JDK 17

Commands:

- `cd android`
- `./gradlew assembleDebug`

If the Android SDK is not installed/configured, Gradle will fail with an error about missing `ANDROID_HOME` / SDK path.

## Run

- Install the debug APK from `android/app/build/outputs/apk/debug/`.
- Open the app, set:
  - Base URL: `http://<host>:8787` (host must be reachable from the phone)
  - Token: the host Bearer token (from `host/.dev-token` in dev)

## Security notes

- The app does not log the token.
- The Android manifest enables cleartext HTTP so LAN usage works; prefer VPN/TLS when exposing beyond a trusted network.

