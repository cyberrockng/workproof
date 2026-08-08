export type WorkProofOutcome = "PASS" | "FAIL" | "INCONCLUSIVE";

export interface BundleCommitment {
  publicSpecHash: `0x${string}`;
  privateBundleHash: `0x${string}`;
  ciphertextHash: `0x${string}`;
  locator: string;
}

export interface TeeInfoSnapshot {
  extensionId: `0x${string}`;
  teeId: `0x${string}`;
  platform: string;
  codeHash: `0x${string}`;
  publicKey: string;
}

export const WORKPROOF_TEMPLATE_TYPES = [
  "ETH_CALL_EQUALS",
  "ETH_CALL_REVERTS",
  "ERC165_SUPPORTS_INTERFACE",
  "CODE_SIZE_RANGE",
  "STORAGE_AT_EQUALS"
] as const;
