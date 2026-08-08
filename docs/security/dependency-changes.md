# Dependency Changes

Scaffold base: `ffb6c4ca7c160c49be59e00fe537e24d2477b000`

Use this table for every future change:

| Date | Area | Package | Previous | New | Reason | Validation |
|---|---|---|---|---|---|---|
| 2026-08-08 | Solidity (test-only) | `foundry-rs/forge-std` (git submodule, `lib/forge-std`) | not present | v1.16.2 (`bf647bd6046f2f7da30d0c2bf435e5c76a780c1b`) | Needed `vm.sign`/`vm.prank`/`vm.expectRevert`/`vm.warp`/`bound` cheatcodes and `Test` assertions to build the Phase 2 required-simulation suite and the FCC signature-compatibility spike; not linked into any production contract. | `forge test`: 29/29 pass (`test/FccSignatureSpike.t.sol`, `test/WorkProofEscrow.t.sol`); `forge build`/`forge fmt --check` clean. |
