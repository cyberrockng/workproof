// set-workproof-extension-id calls WorkProofEscrow.setExtensionId() on an
// already-deployed, already-registered escrow. Must run AFTER
// register-extension has registered the same address with the
// FlareTeeManager diamond -- setExtensionId() only adopts an id the
// registry already associated with this address, it does not register
// anything itself.
package main

import (
	"flag"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	instrutils "extension-scaffold/tools/pkg/utils"
	"extension-scaffold/tools/pkg/validate"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	escrowF := flag.String("escrow", "", "WorkProofEscrow contract address (required)")
	flag.Parse()

	if *escrowF == "" {
		logger.Fatal("--escrow flag is required")
	}

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	escrowAddress := common.HexToAddress(*escrowF)
	if err := validate.AddressHasCode(testSupport.ChainClient, escrowAddress, "WorkProofEscrow"); err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("Setting extension ID on WorkProofEscrow %s...", escrowAddress.Hex())
	if err := instrutils.SetWorkProofEscrowExtensionId(testSupport, escrowAddress); err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("Extension ID set successfully.")
}
