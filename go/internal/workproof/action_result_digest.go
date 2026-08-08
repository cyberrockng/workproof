package workproof

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/signing"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

// ActionResultSigningDigest follows tee-node's published TEE_ACTION_RESULT
// protocol: Payload(prefix, chainId, ActionResult.Hash()).Hash().
func ActionResultSigningDigest(result teetypes.ActionResult, chainID uint64) ([32]byte, error) {
	inner := common.BytesToHash((&result).Hash())
	payload := signing.NewPayload(signing.TEEActionResult, chainID, inner)
	return payload.Hash()
}
