// Command fcc-spike is the mandatory FCC signature-compatibility spike from
// WORKPROOF_EXECUTION_PLAN.md section 9: it builds a real ActionResult using
// the pinned tee-node types, signs it exactly the way tee-node's internal
// router.SignResult does (Payload{TEEActionResult, chainId, ActionResult.Hash()}.Hash(),
// then crypto.Sign(accounts.TextHash(...), key)) using only real, unmodified
// go-ethereum/go-flare-common/tee-node exported code, and writes a JSON
// vector that the Foundry spike test reconstructs and recovers against.
//
// router.SignResult itself lives in tee-node's internal/ package and cannot
// be imported outside that module, so this reproduces its three-line body
// using the same exported primitives it calls, rather than reimplementing
// any cryptography.
package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	csigning "github.com/flare-foundation/go-flare-common/pkg/signing"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

type vector struct {
	PrivateKeyTestOnly string `json:"privateKeyTestOnly"`
	ExpectedTeeAddress string `json:"expectedTeeAddress"`
	ChainID             uint64 `json:"chainId"`
	ActionResult        struct {
		ID            string `json:"id"`
		SubmissionTag string `json:"submissionTag"`
		Status        uint8  `json:"status"`
		DataHex       string `json:"dataHex"`
	} `json:"actionResult"`
	ActionResultHash string `json:"actionResultHash"`
	PayloadHash      string `json:"payloadHash"`
	EthSignedHash    string `json:"ethSignedHash"`
	Signature        string `json:"signature"`
	SignatureR       string `json:"signatureR"`
	SignatureS       string `json:"signatureS"`
	SignatureV       uint8  `json:"signatureV"`
}

func buildVector(privKey *ecdsa.PrivateKey, chainID uint64, id common.Hash, tag teetypes.SubmissionTag, status uint8, data []byte) (vector, error) {
	ar := teetypes.ActionResult{ID: id, SubmissionTag: tag, Status: status, Data: data}
	arHash := common.BytesToHash(ar.Hash())

	payload := csigning.NewPayload(csigning.TEEActionResult, chainID, arHash)
	payloadHash, err := payload.Hash()
	if err != nil {
		return vector{}, fmt.Errorf("payload hash: %w", err)
	}

	ethSignedHash := accounts.TextHash(payloadHash[:])

	sig, err := crypto.Sign(ethSignedHash, privKey)
	if err != nil {
		return vector{}, fmt.Errorf("sign: %w", err)
	}

	addr := crypto.PubkeyToAddress(privKey.PublicKey)

	v := vector{
		PrivateKeyTestOnly: common.Bytes2Hex(crypto.FromECDSA(privKey)),
		ExpectedTeeAddress: addr.Hex(),
		ChainID:            chainID,
	}
	v.ActionResult.ID = id.Hex()
	v.ActionResult.SubmissionTag = string(tag)
	v.ActionResult.Status = status
	v.ActionResult.DataHex = "0x" + common.Bytes2Hex(data)
	v.ActionResultHash = arHash.Hex()
	v.PayloadHash = common.Bytes2Hex(payloadHash[:])
	if len(v.PayloadHash) > 0 {
		v.PayloadHash = "0x" + v.PayloadHash
	}
	v.EthSignedHash = "0x" + common.Bytes2Hex(ethSignedHash)
	v.Signature = "0x" + common.Bytes2Hex(sig)
	v.SignatureR = "0x" + common.Bytes2Hex(sig[:32])
	v.SignatureS = "0x" + common.Bytes2Hex(sig[32:64])
	v.SignatureV = sig[64] + 27

	return v, nil
}

func main() {
	out := flag.String("out", "", "output JSON path")
	flag.Parse()

	// Fixed, well-known throwaway test key (Anvil default account #0). Never
	// use for anything holding value; this is a local signature fixture only.
	privKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	id := common.HexToHash("0x1111")
	data := []byte(`{"outcome":"PASS","attempt":1}`)

	primary, err := buildVector(privKey, 114, id, teetypes.Submit, 1, data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	crossChain, err := buildVector(privKey, 16, id, teetypes.Submit, 1, data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	doc := map[string]any{
		"protocol":               "TEE_ACTION_RESULT",
		"actionResultHashMethod": "keccak256(keccak256(data) || id || keccak256(submissionTag) || status)",
		"payloadHashMethod":      "keccak256(abi.encode(bytes32('TEE_ACTION_RESULT'), uint256(chainId), actionResultHash))",
		"signingMethod":          "secp256k1(crypto.Sign(accounts.TextHash(payloadHash), key)) -- EIP-191 personal-sign wrap over payloadHash",
		"source":                 "tee-node v0.0.24 pkg/types/actions.go Hash(), internal/router/utils.go SignResult, pkg/utils/crypto.go Sign; go-flare-common v1.2.2-...-09a10067e6a4 pkg/signing/hash.go Payload.Hash()",
		"note":                   "router.SignResult is an internal tee-node package and cannot be imported cross-module; this reproduces its exact 3-line body using only the same real exported primitives (ActionResult.Hash, signing.Payload.Hash, accounts.TextHash, crypto.Sign), not a reimplementation of any cryptography.",
		"coston2":                primary,
		"crossChainReplayCheck":  crossChain,
	}

	buf, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *out == "" {
		fmt.Println(string(buf))
		return
	}
	if err := os.WriteFile(*out, append(buf, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
