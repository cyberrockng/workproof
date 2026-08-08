# @workproof/schema

Source-controlled schemas for P0 WorkProof artifacts. These schemas define the shared wire contract before Solidity, Go, relayer, and web implementation.

The schema package intentionally has no runtime dependencies in Phase 1. `npm test` validates that every checked-in JSON schema parses and has the required top-level metadata.
