package verifier

import "testing"

func validBundle() *WorkProofBundle {
	return &WorkProofBundle{
		FormatVersion:    1,
		TemplateID:       "template-1",
		TemplateVersion:  "v1.0.0",
		TargetChainID:    114,
		PublicSpecHash:   "0x" + repeat("ab", 32),
		VectorCount:      1,
		SelectionCount:   1,
		GasLimitPerCall:  100000,
		TimeoutMsPerCall: 1000,
		MaxResponseBytes: 1024,
		Vectors: []Vector{
			{
				ID:             "v1",
				Type:           VectorEthCallEquals,
				Calldata:       "0x1234",
				Caller:         "0x" + repeat("11", 20),
				Value:          "0x0",
				ExpectedReturn: "0x5678",
			},
		},
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

func TestBundleValidateAccepts(t *testing.T) {
	if err := validBundle().Validate(); err != nil {
		t.Fatalf("expected valid bundle to pass, got: %v", err)
	}
}

func TestBundleValidateRejectsWrongFormatVersion(t *testing.T) {
	b := validBundle()
	b.FormatVersion = 2
	if err := b.Validate(); err == nil {
		t.Fatal("expected rejection for formatVersion != 1")
	}
}

func TestBundleValidateRejectsWrongChain(t *testing.T) {
	b := validBundle()
	b.TargetChainID = 1
	if err := b.Validate(); err == nil {
		t.Fatal("expected rejection for non-Coston2 targetChainId")
	}
}

func TestBundleValidateRejectsSelectionExceedingVectorCount(t *testing.T) {
	b := validBundle()
	b.SelectionCount = 2
	if err := b.Validate(); err == nil {
		t.Fatal("expected rejection when selectionCount > vectorCount")
	}
}

func TestBundleValidateRejectsMismatchedVectorCount(t *testing.T) {
	b := validBundle()
	b.VectorCount = 2
	if err := b.Validate(); err == nil {
		t.Fatal("expected rejection when declared vectorCount != len(vectors)")
	}
}

func TestBundleValidateRejectsGasLimitAboveEngineCap(t *testing.T) {
	b := validBundle()
	b.GasLimitPerCall = 3_000_000 // within schema bounds, above config.CallGasCap
	if err := b.Validate(); err == nil {
		t.Fatal("expected rejection when gasLimitPerCall exceeds engine cap")
	}
}

func TestBundleValidateRejectsDuplicateVectorIDs(t *testing.T) {
	b := validBundle()
	b.VectorCount = 2
	b.Vectors = append(b.Vectors, b.Vectors[0])
	if err := b.Validate(); err == nil {
		t.Fatal("expected rejection for duplicate vector ids")
	}
}

func TestVectorValidateRejectsCrossTypeFields(t *testing.T) {
	v := Vector{ID: "v1", Type: VectorCodeSizeRange, MinBytes: 1, MaxBytes: 10, Calldata: "0x1234"}
	if err := v.validate(); err == nil {
		t.Fatal("expected rejection: CODE_SIZE_RANGE must not carry ETH_CALL fields")
	}
}

func TestVectorValidateAcceptsEachP0Type(t *testing.T) {
	addr := "0x" + repeat("11", 20)
	hash := "0x" + repeat("ab", 32)
	cases := []Vector{
		{ID: "a", Type: VectorEthCallEquals, Calldata: "0x12", Caller: addr, Value: "0x0", ExpectedReturn: "0x34"},
		{ID: "b", Type: VectorEthCallReverts, Calldata: "0x12", Caller: addr, Value: "0x0"},
		{ID: "c", Type: VectorErc165SupportsInterface, InterfaceID: "0x01ffc9a7", Expected: boolPtr(true)},
		{ID: "d", Type: VectorCodeSizeRange, MinBytes: 1, MaxBytes: 100},
		{ID: "e", Type: VectorStorageAtEquals, Slot: hash, ExpectedValue: hash},
	}
	for _, v := range cases {
		if err := v.validate(); err != nil {
			t.Errorf("%s (%s): expected valid, got %v", v.ID, v.Type, err)
		}
	}
}

func boolPtr(b bool) *bool { return &b }

func TestCanonicalizeAndHashIsDeterministic(t *testing.T) {
	b := validBundle()
	_, hash1, err := b.CanonicalizeAndHash()
	if err != nil {
		t.Fatalf("CanonicalizeAndHash: %v", err)
	}
	_, hash2, err := b.CanonicalizeAndHash()
	if err != nil {
		t.Fatalf("CanonicalizeAndHash (2nd call): %v", err)
	}
	if hash1 != hash2 {
		t.Fatal("CanonicalizeAndHash is not deterministic across calls on the same value")
	}
}

func TestCanonicalizeIsKeyOrderIndependent(t *testing.T) {
	// JCS (RFC 8785) canonical output must not depend on the source struct's
	// field declaration order -- two logically-identical bundles built with
	// fields touched in a different order must canonicalize identically.
	a := validBundle()
	b := &WorkProofBundle{
		MaxResponseBytes: a.MaxResponseBytes,
		TimeoutMsPerCall: a.TimeoutMsPerCall,
		GasLimitPerCall:  a.GasLimitPerCall,
		SelectionCount:   a.SelectionCount,
		VectorCount:      a.VectorCount,
		PublicSpecHash:   a.PublicSpecHash,
		TargetChainID:    a.TargetChainID,
		TemplateVersion:  a.TemplateVersion,
		TemplateID:       a.TemplateID,
		FormatVersion:    a.FormatVersion,
		Vectors:          a.Vectors,
	}
	_, hashA, err := a.CanonicalizeAndHash()
	if err != nil {
		t.Fatal(err)
	}
	_, hashB, err := b.CanonicalizeAndHash()
	if err != nil {
		t.Fatal(err)
	}
	if hashA != hashB {
		t.Fatal("canonicalization depends on Go struct field order, which JCS must not")
	}
}
