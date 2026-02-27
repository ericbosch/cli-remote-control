# Setup (WSL2)

Same steps as [setup-linux.md](setup-linux.md), with these additions:

- **Go/Node/Java:** Install inside WSL2 as on Linux.
- **Binding:** To reach the host from Windows or from your phone (same LAN), start with:
  ```bash
  ./rc-host serve --bind=0.0.0.0 --generate-dev-token
  ```
  You may need to allow the port in Windows Firewall.
- **Android emulator on Windows:** In the app, use `http://10.0.2.2:8765` if the host runs in WSL2 and you’re using the emulator on Windows. For WSL2’s own IP (from Windows or phone), use the WSL2 interface address (e.g. from `hostname -I` in WSL2).
- **Android emulator on Windows:** In the app, use `http://10.0.2.2:8787` if the host runs in WSL2 and you’re using the emulator on Windows. For WSL2’s own IP (from Windows or phone), use the WSL2 interface address (e.g. from `hostname -I` in WSL2).
- **Phone on same LAN:** Use the Windows host IP (not WSL2’s) and the port (e.g. `8787`). Ensure the host is bound to `0.0.0.0` and that Windows Firewall allows the port.
