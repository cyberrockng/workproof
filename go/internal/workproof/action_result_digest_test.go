package workproof

import (
	"github.com/ethereum/go-ethereum/common"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	"testing"
)

func TestActionResultSigningDigestVector(t *testing.T) {
	r := teetypes.ActionResult{ID: common.HexToHash("0x1111"), SubmissionTag: teetypes.Submit, Status: 1, Data: []byte(`{"outcome":"PASS","attempt":1}`)}
	digest, err := ActionResultSigningDigest(r, 114)
	if err != nil {
		t.Fatal(err)
	}
	if got := common.Bytes2Hex(digest[:]); got != "d333554b98a27804ad3569d96ebe74b539661700d94c4572ccc74d13c297d4ed" {
		t.Fatalf("digest changed: %s", got)
	}
}
