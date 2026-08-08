# Secret Scanning

Repository secret scanning must be enabled before the first public push if the hosting provider supports it.

Local rules for this project:

- never commit `.env`, private keys, indexer credentials, cloud credentials, plaintext hidden vectors, or unredacted proxy logs;
- commit sanitized `.env.example` files only;
- use separate deployer, client, contractor, treasury, and relayer wallets;
- rotate any key pasted into chat, tickets, logs, screenshots, or browser recordings;
- include a canary secret in local private-bundle tests and prove it does not appear in browser output, proxy logs, reports, transactions, or committed files.
