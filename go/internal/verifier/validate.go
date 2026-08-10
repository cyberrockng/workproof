package verifier

import (
	"fmt"
	"regexp"

	"extension-scaffold/internal/config"
)

var (
	templateVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	bytes32Pattern         = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	bytes4Pattern          = regexp.MustCompile(`^0x[0-9a-fA-F]{8}$`)
	addressPattern         = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	bytesPattern           = regexp.MustCompile(`^0x([0-9a-fA-F]{2})*$`)
	uint256HexPattern      = regexp.MustCompile(`^0x[0-9a-fA-F]{1,64}$`)
)

// Validate enforces packages/schema/schemas/workproof-bundle-v1.schema.json
// exactly, plus the resource-limit caps from plan section 11 / config.go.
// Called before any vector is ever executed.
func (b *WorkProofBundle) Validate() error {
	if b.FormatVersion != 1 {
		return fmt.Errorf("formatVersion must be 1, got %d", b.FormatVersion)
	}
	if len(b.TemplateID) < 1 || len(b.TemplateID) > 64 {
		return fmt.Errorf("templateId length out of bounds")
	}
	if !templateVersionPattern.MatchString(b.TemplateVersion) {
		return fmt.Errorf("templateVersion does not match vX.Y.Z")
	}
	if b.TargetChainID != 114 {
		return fmt.Errorf("targetChainId must be 114 (Coston2), got %d", b.TargetChainID)
	}
	if !bytes32Pattern.MatchString(b.PublicSpecHash) {
		return fmt.Errorf("publicSpecHash malformed")
	}
	if b.VectorCount < 1 || b.VectorCount > config.MaxVectorCount {
		return fmt.Errorf("vectorCount out of bounds: %d", b.VectorCount)
	}
	if b.SelectionCount < 1 || b.SelectionCount > config.MaxSelectionCount {
		return fmt.Errorf("selectionCount out of bounds: %d", b.SelectionCount)
	}
	if b.SelectionCount > b.VectorCount {
		return fmt.Errorf("selectionCount %d exceeds vectorCount %d", b.SelectionCount, b.VectorCount)
	}
	if b.GasLimitPerCall < 21000 || b.GasLimitPerCall > 5_000_000 {
		return fmt.Errorf("gasLimitPerCall out of schema bounds: %d", b.GasLimitPerCall)
	}
	if b.GasLimitPerCall > config.CallGasCap {
		return fmt.Errorf("gasLimitPerCall %d exceeds engine cap %d", b.GasLimitPerCall, config.CallGasCap)
	}
	if b.TimeoutMsPerCall < 100 || b.TimeoutMsPerCall > 30000 {
		return fmt.Errorf("timeoutMsPerCall out of bounds: %d", b.TimeoutMsPerCall)
	}
	if b.MaxResponseBytes < 1 || b.MaxResponseBytes > 65536 {
		return fmt.Errorf("maxResponseBytes out of schema bounds: %d", b.MaxResponseBytes)
	}
	if b.MaxResponseBytes > config.MaxResponseBytes {
		return fmt.Errorf("maxResponseBytes %d exceeds engine cap %d", b.MaxResponseBytes, config.MaxResponseBytes)
	}
	if len(b.Vectors) != b.VectorCount {
		return fmt.Errorf("vectors length %d does not match declared vectorCount %d", len(b.Vectors), b.VectorCount)
	}
	if len(b.Vectors) < 1 || len(b.Vectors) > config.MaxVectorCount {
		return fmt.Errorf("vectors length out of bounds: %d", len(b.Vectors))
	}

	seenIDs := make(map[string]bool, len(b.Vectors))
	for i := range b.Vectors {
		if err := b.Vectors[i].validate(); err != nil {
			return fmt.Errorf("vector[%d] (%s): %w", i, b.Vectors[i].ID, err)
		}
		if seenIDs[b.Vectors[i].ID] {
			return fmt.Errorf("duplicate vector id %q", b.Vectors[i].ID)
		}
		seenIDs[b.Vectors[i].ID] = true
	}
	return nil
}

func (v *Vector) validate() error {
	if len(v.ID) < 1 || len(v.ID) > 96 {
		return fmt.Errorf("id length out of bounds")
	}
	switch v.Type {
	case VectorEthCallEquals:
		if !bytesPattern.MatchString(v.Calldata) || !addressPattern.MatchString(v.Caller) ||
			!uint256HexPattern.MatchString(v.Value) || !bytesPattern.MatchString(v.ExpectedReturn) {
			return fmt.Errorf("malformed ETH_CALL_EQUALS fields")
		}
		if v.ExpectedRevertSelector != "" || v.ExpectedRevertPattern != "" || v.InterfaceID != "" ||
			v.Expected != nil || v.MinBytes != 0 || v.MaxBytes != 0 || v.Slot != "" || v.ExpectedValue != "" {
			return fmt.Errorf("unexpected fields for ETH_CALL_EQUALS")
		}
	case VectorEthCallReverts:
		if !bytesPattern.MatchString(v.Calldata) || !addressPattern.MatchString(v.Caller) ||
			!uint256HexPattern.MatchString(v.Value) {
			return fmt.Errorf("malformed ETH_CALL_REVERTS fields")
		}
		if v.ExpectedRevertSelector != "" && !bytes4Pattern.MatchString(v.ExpectedRevertSelector) {
			return fmt.Errorf("malformed expectedRevertSelector")
		}
		if len(v.ExpectedRevertPattern) > 128 {
			return fmt.Errorf("expectedRevertPattern too long")
		}
		if v.ExpectedReturn != "" || v.InterfaceID != "" || v.Expected != nil ||
			v.MinBytes != 0 || v.MaxBytes != 0 || v.Slot != "" || v.ExpectedValue != "" {
			return fmt.Errorf("unexpected fields for ETH_CALL_REVERTS")
		}
	case VectorErc165SupportsInterface:
		if !bytes4Pattern.MatchString(v.InterfaceID) || v.Expected == nil {
			return fmt.Errorf("malformed ERC165_SUPPORTS_INTERFACE fields")
		}
		if v.Calldata != "" || v.Caller != "" || v.Value != "" || v.ExpectedReturn != "" ||
			v.ExpectedRevertSelector != "" || v.ExpectedRevertPattern != "" ||
			v.MinBytes != 0 || v.MaxBytes != 0 || v.Slot != "" || v.ExpectedValue != "" {
			return fmt.Errorf("unexpected fields for ERC165_SUPPORTS_INTERFACE")
		}
	case VectorCodeSizeRange:
		if v.MinBytes < 1 || v.MaxBytes < 1 || v.MinBytes > v.MaxBytes {
			return fmt.Errorf("malformed CODE_SIZE_RANGE bounds")
		}
		if v.Calldata != "" || v.Caller != "" || v.Value != "" || v.ExpectedReturn != "" ||
			v.ExpectedRevertSelector != "" || v.ExpectedRevertPattern != "" || v.InterfaceID != "" ||
			v.Expected != nil || v.Slot != "" || v.ExpectedValue != "" {
			return fmt.Errorf("unexpected fields for CODE_SIZE_RANGE")
		}
	case VectorStorageAtEquals:
		if !bytes32Pattern.MatchString(v.Slot) || !bytes32Pattern.MatchString(v.ExpectedValue) {
			return fmt.Errorf("malformed STORAGE_AT_EQUALS fields")
		}
		if v.Calldata != "" || v.Caller != "" || v.Value != "" || v.ExpectedReturn != "" ||
			v.ExpectedRevertSelector != "" || v.ExpectedRevertPattern != "" || v.InterfaceID != "" ||
			v.Expected != nil || v.MinBytes != 0 || v.MaxBytes != 0 {
			return fmt.Errorf("unexpected fields for STORAGE_AT_EQUALS")
		}
	default:
		return fmt.Errorf("unknown vector type %q", v.Type)
	}
	return nil
}
