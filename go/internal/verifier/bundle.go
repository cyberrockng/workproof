// Package verifier implements the WorkProof VERIFY handler: ciphertext
// fetch/decrypt, on-chain state cross-check, deterministic vector
// selection, and VerdictV1 production (plan section 15 Phase 4).
package verifier

import (
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
)

// Vector type strings, matching
// packages/schema/schemas/workproof-bundle-v1.schema.json exactly.
const (
	VectorEthCallEquals           = "ETH_CALL_EQUALS"
	VectorEthCallReverts          = "ETH_CALL_REVERTS"
	VectorErc165SupportsInterface = "ERC165_SUPPORTS_INTERFACE"
	VectorCodeSizeRange           = "CODE_SIZE_RANGE"
	VectorStorageAtEquals         = "STORAGE_AT_EQUALS"
)

// Vector is a flat representation covering all five P0 vector types (a
// discriminated union in the JSON schema's oneOf); Type selects which
// fields are meaningful. validateShape enforces the schema's per-type
// required/forbidden field sets, since a flat Go struct can't express
// "additionalProperties: false" per-variant on its own.
type Vector struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	// ETH_CALL_EQUALS / ETH_CALL_REVERTS
	Calldata string `json:"calldata,omitempty"`
	Caller   string `json:"caller,omitempty"`
	Value    string `json:"value,omitempty"`

	// ETH_CALL_EQUALS
	ExpectedReturn string `json:"expectedReturn,omitempty"`

	// ETH_CALL_REVERTS (both optional; absence of both means "any revert")
	ExpectedRevertSelector string `json:"expectedRevertSelector,omitempty"`
	ExpectedRevertPattern  string `json:"expectedRevertPattern,omitempty"`

	// ERC165_SUPPORTS_INTERFACE
	InterfaceID string `json:"interfaceId,omitempty"`
	Expected    *bool  `json:"expected,omitempty"`

	// CODE_SIZE_RANGE
	MinBytes int `json:"minBytes,omitempty"`
	MaxBytes int `json:"maxBytes,omitempty"`

	// STORAGE_AT_EQUALS
	Slot          string `json:"slot,omitempty"`
	ExpectedValue string `json:"expectedValue,omitempty"`
}

// WorkProofBundle mirrors
// packages/schema/schemas/workproof-bundle-v1.schema.json field-for-field.
type WorkProofBundle struct {
	FormatVersion    int      `json:"formatVersion"`
	TemplateID       string   `json:"templateId"`
	TemplateVersion  string   `json:"templateVersion"`
	TargetChainID    int64    `json:"targetChainId"`
	PublicSpecHash   string   `json:"publicSpecHash"`
	VectorCount      int      `json:"vectorCount"`
	SelectionCount   int      `json:"selectionCount"`
	GasLimitPerCall  int64    `json:"gasLimitPerCall"`
	TimeoutMsPerCall int      `json:"timeoutMsPerCall"`
	MaxResponseBytes int      `json:"maxResponseBytes"`
	Vectors          []Vector `json:"vectors"`
}

// CanonicalizeAndHash JCS-canonicalizes (RFC 8785) the bundle exactly as
// SPEC.md/section 11 requires ("the same canonicalizer must be tested in
// TypeScript and Go") and returns both the canonical bytes and their
// keccak256 hash, for comparison against the job's committed
// privateBundleHash.
func (b *WorkProofBundle) CanonicalizeAndHash() (canonical []byte, hash [32]byte, err error) {
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, hash, fmt.Errorf("marshaling bundle: %w", err)
	}
	canonical, err = jcs.Transform(raw)
	if err != nil {
		return nil, hash, fmt.Errorf("JCS canonicalization: %w", err)
	}
	hash = keccak256(canonical)
	return canonical, hash, nil
}
