package verifier

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSelectVectorsIsDeterministic(t *testing.T) {
	seed := keccak256([]byte("seed-1"))
	a := SelectVectors(seed, 20, 5)
	b := SelectVectors(seed, 20, 5)
	if len(a) != 5 || len(b) != 5 {
		t.Fatalf("expected 5 selections, got %d and %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("selection order differs at index %d: %d vs %d -- same seed must always select the same vectors in the same order", i, a[i], b[i])
		}
	}
}

func TestSelectVectorsDifferentSeedsDiffer(t *testing.T) {
	a := SelectVectors(keccak256([]byte("seed-a")), 50, 10)
	b := SelectVectors(keccak256([]byte("seed-b")), 50, 10)
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("two different seeds produced the identical selection order -- suspicious")
	}
}

func TestSelectVectorsNoDuplicatesAndInRange(t *testing.T) {
	seed := keccak256([]byte("seed-range"))
	selected := SelectVectors(seed, 30, 12)
	seen := make(map[int]bool, len(selected))
	for _, idx := range selected {
		if idx < 0 || idx >= 30 {
			t.Fatalf("selected index %d out of range [0,30)", idx)
		}
		if seen[idx] {
			t.Fatalf("duplicate selected index %d", idx)
		}
		seen[idx] = true
	}
}

func TestSelectVectorsCapsAtVectorCount(t *testing.T) {
	seed := keccak256([]byte("seed-cap"))
	selected := SelectVectors(seed, 3, 10) // selectionCount > vectorCount
	if len(selected) != 3 {
		t.Fatalf("expected selection capped at vectorCount=3, got %d", len(selected))
	}
}

func TestTestSeedIsDeterministic(t *testing.T) {
	rn := big.NewInt(123456789)
	escrow := common.HexToAddress("0x1234567890123456789012345678901234567890")
	jobID := big.NewInt(7)
	specHash := keccak256([]byte("spec"))
	codeHash := keccak256([]byte("code"))

	s1, err := TestSeed(rn, escrow, jobID, 2, specHash, codeHash)
	if err != nil {
		t.Fatalf("TestSeed: %v", err)
	}
	s2, err := TestSeed(rn, escrow, jobID, 2, specHash, codeHash)
	if err != nil {
		t.Fatalf("TestSeed (2nd call): %v", err)
	}
	if s1 != s2 {
		t.Fatal("TestSeed is not deterministic for identical inputs")
	}

	s3, err := TestSeed(rn, escrow, jobID, 3, specHash, codeHash) // different attempt
	if err != nil {
		t.Fatalf("TestSeed (different attempt): %v", err)
	}
	if s1 == s3 {
		t.Fatal("TestSeed did not change when attempt changed -- every binding input must affect the seed")
	}
}
