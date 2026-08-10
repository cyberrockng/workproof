// deploy-workproof-escrow deploys the real production WorkProofEscrow
// contract -- the WorkProof-specific replacement for
// tools/cmd/deploy-contract, which deploys the scaffold's sample
// HelloWorldInstructionSender instead. WorkProofEscrow is itself the
// instruction sender (dispatchVerification calls
// TEE_EXTENSION_REGISTRY.sendInstructions directly) and implements its own
// setExtensionId(), so no second wrapper contract is deployed or needed.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	instrutils "extension-scaffold/tools/pkg/utils"
	"extension-scaffold/tools/pkg/validate"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/pkg/errors"
)

// defaultProtocolFeeBps matches the value every local/fork test suite
// exercises (test/WorkProofEscrow.t.sol FEE_BPS) -- 1%, well under
// WorkProofEscrow's own MAX_PROTOCOL_FEE_BPS (10%) hard cap.
const defaultProtocolFeeBps = 100

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	outFile := flag.String("o", "", "write deployed address to this file (optional)")
	preflightOnly := flag.Bool("preflight-only", false, "run validation checks and exit without deploying")
	flag.Parse()

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	deployer := crypto.PubkeyToAddress(testSupport.Prv.PublicKey)

	treasury, err := resolveTreasury(deployer)
	if err != nil {
		fccutils.FatalWithCause(err)
	}
	protocolFeeBps, err := resolveProtocolFeeBps()
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// --- Pre-flight validation ---
	logger.Infof("Deployer:             %s", deployer.Hex())
	logger.Infof("Chain ID:             %s", testSupport.ChainID.String())
	logger.Infof("FlareTeeManager:      %s", testSupport.Addresses.FlareTeeManager.Hex())
	logger.Infof("Treasury:             %s", treasury.Hex())
	logger.Infof("Protocol fee (bps):   %d", protocolFeeBps)

	if err := validate.AddressNotZero(testSupport.Addresses.FlareTeeManager, "FlareTeeManager"); err != nil {
		fccutils.FatalWithCause(err)
	}
	if err := validate.AddressHasCode(testSupport.ChainClient, testSupport.Addresses.FlareTeeManager, "FlareTeeManager"); err != nil {
		fccutils.FatalWithCause(err)
	}
	if err := validate.AddressNotZero(treasury, "Treasury"); err != nil {
		fccutils.FatalWithCause(err)
	}
	if err := validate.KeyHasFunds(testSupport.ChainClient, testSupport.Prv, validate.MinDeployBalance); err != nil {
		fccutils.FatalWithCause(err)
	}

	if *preflightOnly {
		logger.Infof("Pre-flight checks passed. Exiting without deploying.")
		return
	}

	logger.Infof("Deploying WorkProofEscrow contract...")
	address, _, err := instrutils.DeployWorkProofEscrow(testSupport, treasury, protocolFeeBps)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("WorkProofEscrow deployed at: %s", address.Hex())

	if *outFile != "" {
		os.MkdirAll(filepath.Dir(*outFile), 0755)
		os.WriteFile(*outFile, []byte(address.Hex()), 0644)
	}

	// Machine-readable output on stdout (for scripts) -- must be fmt.Println,
	// not the println builtin, which writes to stderr and would silently
	// break pre-build.sh's `| tail -1` address capture.
	fmt.Println(address.Hex())
}

// resolveTreasury reads WORKPROOF_TREASURY from the environment (set by
// .env.<chain>, matching every other WORKPROOF_* var's convention -- see
// docs/security/dependency-changes.md and .env.example). Falling back to
// the deployer address would silently create an escrow whose fee sink is
// controlled by the same key that deploys it, which is a real
// misconfiguration risk on a live chain, not a safe default -- this must be
// set explicitly.
func resolveTreasury(deployer common.Address) (common.Address, error) {
	raw := os.Getenv("WORKPROOF_TREASURY")
	if raw == "" {
		return common.Address{}, errors.New("WORKPROOF_TREASURY is not set -- refusing to default the fee-sink address")
	}
	treasury := common.HexToAddress(raw)
	if treasury == deployer {
		logger.Warnf("WARNING: WORKPROOF_TREASURY equals the deployer address (%s) -- confirm this is intentional", treasury.Hex())
	}
	return treasury, nil
}

func resolveProtocolFeeBps() (uint16, error) {
	raw := os.Getenv("WORKPROOF_PROTOCOL_FEE_BPS")
	if raw == "" {
		return defaultProtocolFeeBps, nil
	}
	parsed, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		return 0, errors.Errorf("WORKPROOF_PROTOCOL_FEE_BPS is not a valid uint16: %s", raw)
	}
	return uint16(parsed), nil
}
