package utils

import (
	"context"
	"time"

	"extension-scaffold/tools/pkg/contracts/workproofescrow"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

// DeployWorkProofEscrow deploys the real production WorkProofEscrow
// contract -- NOT HelloWorldInstructionSender. Both registry constructor
// args are the FlareTeeManager diamond proxy (it routes ExtensionManager
// and MachineManager calls to the right facets), the same pattern
// DeployInstructionSender uses for the scaffold's own sample contract.
// WorkProofEscrow is itself the instruction sender: it calls
// TEE_EXTENSION_REGISTRY.sendInstructions directly from
// dispatchVerification, so unlike HelloWorldInstructionSender it never
// needs a second wrapper contract.
func DeployWorkProofEscrow(s *support.Support, treasury common.Address, protocolFeeBps uint16) (common.Address, *workproofescrow.WorkProofEscrow, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to create transactor: %s", err)
	}

	address, tx, contract, err := workproofescrow.DeployWorkProofEscrow(
		opts, s.ChainClient, s.Addresses.FlareTeeManager, s.Addresses.FlareTeeManager, treasury, protocolFeeBps,
	)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("failed to deploy contract: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	receipt, err := bind.WaitMined(ctx, s.ChainClient, tx)
	if err != nil {
		return common.Address{}, nil, errors.Errorf("deployment tx not mined within 2 minutes (tx: %s): %s", tx.Hash().Hex(), err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, nil, errors.New("contract deployment failed")
	}

	return address, contract, nil
}

// SetWorkProofEscrowExtensionId calls WorkProofEscrow.setExtensionId() on
// the deployed contract. This can only succeed AFTER the registry-side
// registration (register-extension / fccutils.SetupExtension, which is
// already fully generic and needs no WorkProof-specific changes) has run
// for this exact address: setExtensionId() scans
// TEE_EXTENSION_REGISTRY.getTeeExtensionInstructionsSender(i) for the entry
// that already points at address(this) and adopts that id -- it does not
// register anything itself.
func SetWorkProofEscrowExtensionId(s *support.Support, escrowAddress common.Address) error {
	escrow, err := workproofescrow.NewWorkProofEscrow(escrowAddress, s.ChainClient)
	if err != nil {
		return errors.Errorf("failed to bind contract: %s", err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("failed to create transactor: %s", err)
	}

	tx, err := escrow.SetExtensionId(opts)
	if err != nil {
		reason := fccutils.DecodeRevertReason(err)
		if reason == "" {
			parsed, _ := workproofescrow.WorkProofEscrowMetaData.GetAbi()
			if parsed != nil {
				callData, packErr := parsed.Pack("setExtensionId")
				if packErr == nil {
					from := crypto.PubkeyToAddress(s.Prv.PublicKey)
					reason = fccutils.SimulateAndDecodeRevert(
						s.ChainClient, from, escrowAddress, nil, callData,
					)
				}
			}
		}
		if reason != "" {
			return errors.Errorf("failed to call setExtensionId: %s (revert reason: %s)", err, reason)
		}
		return errors.Errorf("failed to call setExtensionId: %s", err)
	}

	receipt, err := bind.WaitMined(context.Background(), s.ChainClient, tx)
	if err != nil {
		return errors.Errorf("failed waiting for transaction: %s", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		parsed, _ := workproofescrow.WorkProofEscrowMetaData.GetAbi()
		if parsed != nil {
			callData, packErr := parsed.Pack("setExtensionId")
			if packErr == nil {
				from := crypto.PubkeyToAddress(s.Prv.PublicKey)
				reason := fccutils.SimulateAndDecodeRevert(
					s.ChainClient, from, escrowAddress, nil, callData,
				)
				if reason != "" {
					return errors.Errorf("setExtensionId transaction failed (revert reason: %s)", reason)
				}
			}
		}
		return errors.New("setExtensionId transaction failed")
	}

	return nil
}
