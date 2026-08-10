package verifier

import "testing"

// Regression test for a real bug: bundle.Validate() only format-checked
// PublicSpecHash (a regex match), and nothing anywhere ever compared it to
// the real on-chain-committed instr.SpecHash. A client could submit a
// private bundle whose declared publicSpecHash -- and therefore whose
// hidden tests -- had nothing to do with the public agreement the job was
// actually created against.
func TestCheckPublicSpecHashRejectsMismatch(t *testing.T) {
	specHash := [32]byte{0x01, 0x02, 0x03}
	wrongHash := "0x" + repeat("ff", 32)

	if err := checkPublicSpecHash(wrongHash, specHash); err == nil {
		t.Fatal("expected an error when bundle publicSpecHash does not match instr.SpecHash")
	}
}

func TestCheckPublicSpecHashAcceptsMatch(t *testing.T) {
	specHash := [32]byte{0xde, 0xad, 0xbe, 0xef}
	matching := "0xdeadbeef" + repeat("00", 28)

	if err := checkPublicSpecHash(matching, specHash); err != nil {
		t.Fatalf("expected the matching hash to be accepted, got: %v", err)
	}
}

func TestCheckPublicSpecHashRejectsMalformed(t *testing.T) {
	specHash := [32]byte{0x01}
	cases := []string{"", "not-hex", "0x1234", "0x" + repeat("ab", 31)}
	for _, c := range cases {
		if err := checkPublicSpecHash(c, specHash); err == nil {
			t.Errorf("expected an error for malformed publicSpecHash %q", c)
		}
	}
}
