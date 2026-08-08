# Evidence Directory

Store sanitized evidence only. Do not commit credentials, private keys, private bundle plaintext, secret canaries, VPN details, or unredacted logs.

Expected files during execution:

- `toolchain.txt`: sanitized preflight command output.
- `deployment-manifest.example.json`: manifest shape before live Coston2 values exist.
- `demo-evidence.example.json`: manifest shape for FAIL, PASS, and REFUND flows.
- `RELEASE_SIGNOFF.md`: final release sign-off after all P0 gates pass.
