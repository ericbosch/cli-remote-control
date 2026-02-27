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
  - Base URL:
    - Recommended (host serves UI): `http://127.0.0.1:8787` when using SSH port-forwarding from the phone.
    - LAN (only if you intentionally expose it): `http://<LAN_IP>:8787`
  - Token: the host Bearer token (from `host/.dev-token` in dev)

### SSH port-forward (phone)

On your phone (e.g. Termux):

- `ssh -N -L 8787:127.0.0.1:8787 <user>@<your-computer>`

Then set Base URL to `http://127.0.0.1:8787` in the app.

## Security notes

- The app does not log the token.
- The Android manifest enables cleartext HTTP so LAN usage works; prefer VPN/TLS when exposing beyond a trusted network.
