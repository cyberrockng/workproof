package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"extension-scaffold/internal/config"
	"extension-scaffold/internal/verifier"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	teeutils "github.com/flare-foundation/tee-node/pkg/utils"
	"github.com/gowebpki/jcs"
)

const (
	defaultPrincipalWei = int64(1_000)
	feeWei              = int64(1_000_000_000)
)

var (
	erc20ABI  = mustABI(`[{"type":"function","name":"approve","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}],"stateMutability":"nonpayable"},{"type":"function","name":"balanceOf","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}],"stateMutability":"view"},{"type":"function","name":"allowance","inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"outputs":[{"name":"","type":"uint256"}],"stateMutability":"view"}]`)
	escrowABI = mustABI(`[{"type":"function","name":"acceptJob","inputs":[{"name":"id","type":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"createJob","inputs":[{"name":"contractor","type":"address"},{"name":"principal","type":"uint128"},{"name":"acceptBy","type":"uint64"},{"name":"submitBy","type":"uint64"},{"name":"graceEnds","type":"uint64"},{"name":"verificationTimeout","type":"uint64"},{"name":"specHash","type":"bytes32"},{"name":"privateBundleHash","type":"bytes32"},{"name":"engineVersionHash","type":"bytes32"},{"name":"ciphertextHash","type":"bytes32"}],"outputs":[{"name":"id","type":"uint256"}],"stateMutability":"nonpayable"},{"type":"function","name":"dispatchVerification","inputs":[{"name":"id","type":"uint256"}],"outputs":[],"stateMutability":"payable"},{"type":"function","name":"getJob","inputs":[{"name":"id","type":"uint256"}],"outputs":[{"name":"","type":"tuple","components":[{"name":"terms","type":"tuple","components":[{"name":"client","type":"address"},{"name":"contractor","type":"address"},{"name":"expectedTee","type":"address"},{"name":"principal","type":"uint128"},{"name":"fee","type":"uint128"},{"name":"createdAt","type":"uint64"},{"name":"acceptBy","type":"uint64"},{"name":"submitBy","type":"uint64"},{"name":"graceEnds","type":"uint64"},{"name":"verificationTimeout","type":"uint64"},{"name":"specHash","type":"bytes32"},{"name":"privateBundleHash","type":"bytes32"},{"name":"engineVersionHash","type":"bytes32"},{"name":"ciphertextHash","type":"bytes32"}]},{"name":"current","type":"tuple","components":[{"name":"attempt","type":"uint64"},{"name":"artifactAddress","type":"address"},{"name":"artifactBlock","type":"uint256"},{"name":"artifactCodeHash","type":"bytes32"},{"name":"targetRound","type":"uint256"},{"name":"randomRound","type":"uint256"},{"name":"randomValueHash","type":"bytes32"},{"name":"randomLocked","type":"bool"},{"name":"instructionId","type":"bytes32"},{"name":"dispatchedAt","type":"uint64"},{"name":"timeoutAt","type":"uint64"}]},{"name":"state","type":"uint8"},{"name":"settled","type":"bool"}]}],"stateMutability":"view"},{"type":"function","name":"lockRandomness","inputs":[{"name":"id","type":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"nextJobId","inputs":[],"outputs":[{"name":"","type":"uint256"}],"stateMutability":"view"},{"type":"function","name":"protocolFeeBps","inputs":[],"outputs":[{"name":"","type":"uint16"}],"stateMutability":"view"},{"type":"function","name":"refundExpired","inputs":[{"name":"id","type":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"settleAttempt","inputs":[{"name":"id","type":"uint256"},{"name":"data","type":"bytes"},{"name":"opType","type":"bytes32"},{"name":"opCommand","type":"bytes32"},{"name":"submissionTag","type":"string"},{"name":"status","type":"uint8"},{"name":"signature","type":"bytes"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"submitAttempt","inputs":[{"name":"id","type":"uint256"},{"name":"artifactAddress","type":"address"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"token","inputs":[],"outputs":[{"name":"","type":"address"}],"stateMutability":"view"},{"type":"function","name":"treasury","inputs":[],"outputs":[{"name":"","type":"address"}],"stateMutability":"view"},{"type":"event","name":"VerificationDispatched","inputs":[{"name":"jobId","type":"uint256","indexed":true},{"name":"attempt","type":"uint64","indexed":true},{"name":"instructionId","type":"bytes32","indexed":false},{"name":"expectedTee","type":"address","indexed":false},{"name":"timeoutAt","type":"uint64","indexed":false}],"anonymous":false}]`)
)

type jobView struct {
	Terms struct {
		Client              common.Address
		Contractor          common.Address
		ExpectedTee         common.Address
		Principal           *big.Int
		Fee                 *big.Int
		CreatedAt           uint64
		AcceptBy            uint64
		SubmitBy            uint64
		GraceEnds           uint64
		VerificationTimeout uint64
		SpecHash            [32]byte
		PrivateBundleHash   [32]byte
		EngineVersionHash   [32]byte
		CiphertextHash      [32]byte
	}
	Current struct {
		Attempt          uint64
		ArtifactAddress  common.Address
		ArtifactBlock    *big.Int
		ArtifactCodeHash [32]byte
		TargetRound      *big.Int
		RandomRound      *big.Int
		RandomValueHash  [32]byte
		RandomLocked     bool
		InstructionId    [32]byte
		DispatchedAt     uint64
		TimeoutAt        uint64
	}
	State   uint8
	Settled bool
}

type configValues struct {
	rpcURL        string
	proxyURL      string
	escrow        common.Address
	privKey       string
	contractor    common.Address
	artifact      common.Address
	teePublicX    string
	teePublicY    string
	principal     *big.Int
	engineHash    common.Hash
	settleTag     string
	storeCipher   string
	failVector    bool
	refundOnly    bool
	pollTimeout   time.Duration
	acceptDelay   time.Duration
	submitDelay   time.Duration
	graceDelay    time.Duration
	verifyTimeout time.Duration
	resumeJobID   *big.Int
}

func main() {
	cfg := parseFlags()
	ctx := context.Background()

	client, err := ethclient.Dial(cfg.rpcURL)
	check(err, "dial chain")
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	check(err, "chain id")

	key, err := parsePrivateKey(cfg.privKey)
	check(err, "private key")
	from := crypto.PubkeyToAddress(key.PublicKey)
	fmt.Printf("runner account: %s\n", from.Hex())

	escrow := bind.NewBoundContract(cfg.escrow, escrowABI, client, client, client)
	if cfg.resumeJobID != nil {
		resumeFromReady(ctx, client, escrow, key, chainID, cfg)
		return
	}

	token := callAddress(ctx, escrow, "token")
	feeBps := callUint16(ctx, escrow, "protocolFeeBps")
	fee := new(big.Int).Div(new(big.Int).Mul(cfg.principal, big.NewInt(int64(feeBps))), big.NewInt(10_000))
	total := new(big.Int).Add(cfg.principal, fee)
	fmt.Printf("escrow: %s\n", cfg.escrow.Hex())
	fmt.Printf("token: %s principal=%s fee=%s\n", token.Hex(), cfg.principal, fee)

	tokenContract := bind.NewBoundContract(token, erc20ABI, client, client, client)
	balance := callBig(ctx, tokenContract, "balanceOf", from)
	if balance.Cmp(total) < 0 {
		fatalf("runner account token balance %s is below required %s", balance, total)
	}

	specHash := crypto.Keccak256Hash([]byte("workproof-phase5-code-size-v1"))
	bundle, privateBundleHash, err := buildBundle(specHash, cfg.failVector)
	check(err, "build bundle")
	ciphertext, err := encryptForTEE(bundle, cfg.teePublicX, cfg.teePublicY)
	check(err, "encrypt bundle")
	ciphertextHash := crypto.Keccak256Hash(ciphertext)
	if cfg.storeCipher != "" {
		cipherPath := cfg.storeCipher
		if info, err := os.Stat(cipherPath); err == nil && info.IsDir() {
			cipherPath = strings.TrimRight(cipherPath, string(os.PathSeparator)) + string(os.PathSeparator) + hex.EncodeToString(ciphertextHash[:])
		}
		check(os.WriteFile(cipherPath, ciphertext, 0o600), "write ciphertext")
		fmt.Printf("ciphertext written: %s\n", cipherPath)
	}

	engineHash := cfg.engineHash
	if engineHash == (common.Hash{}) {
		engineHash = crypto.Keccak256Hash([]byte(config.WorkProofEngineVersion))
	}
	auth := authFor(ctx, client, key, chainID, nil)
	tx := transact(ctx, client, tokenContract, auth, "approve", cfg.escrow, total)
	fmt.Printf("approve tx: %s\n", tx.Hash())

	nextJobID := callBig(ctx, escrow, "nextJobId")
	now := uint64(time.Now().Unix())
	auth = authFor(ctx, client, key, chainID, nil)
	acceptBy, submitBy, graceEnds := deadlines(now, cfg)
	tx = transact(ctx, client, escrow, auth, "createJob", cfg.contractor, cfg.principal, acceptBy, submitBy, graceEnds, uint64(cfg.verifyTimeout.Seconds()), specHash, [32]byte(privateBundleHash), [32]byte(engineHash), [32]byte(ciphertextHash))
	fmt.Printf("createJob tx: %s jobId=%s\n", tx.Hash(), nextJobID)
	fmt.Printf("deadlines: acceptBy=%d submitBy=%d graceEnds=%d verificationTimeout=%ds\n", acceptBy, submitBy, graceEnds, uint64(cfg.verifyTimeout.Seconds()))

	if cfg.refundOnly {
		waitForGrace(graceEnds)
		clientBefore := callBig(ctx, tokenContract, "balanceOf", from)
		auth = authFor(ctx, client, key, chainID, nil)
		tx = transact(ctx, client, escrow, auth, "refundExpired", nextJobID)
		fmt.Printf("refundExpired tx: %s\n", tx.Hash())
		clientAfter := callBig(ctx, tokenContract, "balanceOf", from)
		job := getJob(ctx, escrow, nextJobID)
		fmt.Printf("refund delta=%s final job state=%d settled=%t\n", new(big.Int).Sub(clientAfter, clientBefore), job.State, job.Settled)
		return
	}

	auth = authFor(ctx, client, key, chainID, nil)
	tx = transact(ctx, client, escrow, auth, "acceptJob", nextJobID)
	fmt.Printf("acceptJob tx: %s\n", tx.Hash())

	auth = authFor(ctx, client, key, chainID, nil)
	tx = transact(ctx, client, escrow, auth, "submitAttempt", nextJobID, cfg.artifact)
	fmt.Printf("submitAttempt tx: %s\n", tx.Hash())

	waitForRandomness(ctx, client, escrow, key, chainID, nextJobID)
	job := getJob(ctx, escrow, nextJobID)
	fmt.Printf("randomness locked: round=%s state=%d\n", job.Current.RandomRound, job.State)

	auth = authFor(ctx, client, key, chainID, big.NewInt(feeWei))
	tx = transact(ctx, client, escrow, auth, "dispatchVerification", nextJobID)
	fmt.Printf("dispatchVerification tx: %s\n", tx.Hash())
	job = getJob(ctx, escrow, nextJobID)
	instructionID := common.BytesToHash(job.Current.InstructionId[:])
	fmt.Printf("instructionId: %s expectedTee=%s\n", instructionID.Hex(), job.Terms.ExpectedTee.Hex())

	result, err := pollActionResult(cfg.proxyURL, instructionID, cfg.settleTag, cfg.pollTimeout)
	check(err, "poll action result")
	fmt.Printf("action result: tag=%s status=%d log=%q dataBytes=%d\n", result.Result.SubmissionTag, result.Result.Status, result.Result.Log, len(result.Result.Data))
	if result.Result.Status != 1 {
		fatalf("TEE returned status=%d; not settling an unsuccessful action result", result.Result.Status)
	}

	auth = authFor(ctx, client, key, chainID, nil)
	tx = transact(ctx, client, escrow, auth, "settleAttempt", nextJobID, []byte(result.Result.Data), [32]byte(result.Result.OPType), [32]byte(result.Result.OPCommand), string(result.Result.SubmissionTag), result.Result.Status, []byte(result.Signature))
	fmt.Printf("settleAttempt tx: %s\n", tx.Hash())
	job = getJob(ctx, escrow, nextJobID)
	fmt.Printf("final job state=%d settled=%t\n", job.State, job.Settled)
}

func parseFlags() configValues {
	cfg := configValues{}
	flag.StringVar(&cfg.rpcURL, "rpc", firstEnv("CHAIN_URL", "WORKPROOF_RPC_URL"), "Coston2 RPC URL")
	flag.StringVar(&cfg.proxyURL, "proxy", firstEnv("EXT_PROXY_URL", "http://localhost:6674"), "extension proxy URL")
	flag.Var(addressValue{&cfg.escrow}, "escrow", "WorkProofEscrow address")
	flag.StringVar(&cfg.privKey, "private-key", os.Getenv("DEPLOYMENT_PRIVATE_KEY"), "funded private key")
	flag.Var(addressValue{&cfg.contractor}, "contractor", "contractor address")
	flag.Var(addressValue{&cfg.artifact}, "artifact", "artifact contract address")
	flag.StringVar(&cfg.teePublicX, "tee-x", "", "TEE public key x hex from /info")
	flag.StringVar(&cfg.teePublicY, "tee-y", "", "TEE public key y hex from /info")
	flag.StringVar(&cfg.settleTag, "tag", "threshold", "submission tag to poll and settle")
	flag.Var(hashValue{&cfg.engineHash}, "engine-hash", "optional engineVersionHash override")
	flag.StringVar(&cfg.storeCipher, "store-cipher", "", "optional path to write encrypted bundle")
	flag.BoolVar(&cfg.failVector, "fail", false, "build a failing CODE_SIZE_RANGE vector")
	flag.BoolVar(&cfg.refundOnly, "refund-only", false, "create a short-deadline job and refund it after graceEnds without accepting")
	flag.DurationVar(&cfg.pollTimeout, "poll-timeout", 10*time.Minute, "max wait for action result")
	flag.DurationVar(&cfg.acceptDelay, "accept-delay", 10*time.Minute, "duration from now to acceptBy")
	flag.DurationVar(&cfg.submitDelay, "submit-delay", time.Hour, "duration from now to submitBy")
	flag.DurationVar(&cfg.graceDelay, "grace-delay", 2*time.Hour, "duration from now to graceEnds")
	flag.DurationVar(&cfg.verifyTimeout, "verification-timeout", 20*time.Minute, "verification timeout stored on the job")
	resumeJob := flag.Int64("resume-job", -1, "resume an existing ready-to-verify job id")
	principal := flag.Int64("principal", defaultPrincipalWei, "escrow principal in token base units")
	flag.Parse()

	if cfg.escrow == (common.Address{}) {
		cfg.escrow = common.HexToAddress(firstEnv("WORKPROOF_ESCROW_ADDRESS"))
	}
	if cfg.contractor == (common.Address{}) {
		if cfg.privKey != "" {
			key, err := parsePrivateKey(cfg.privKey)
			check(err, "derive default contractor")
			cfg.contractor = crypto.PubkeyToAddress(key.PublicKey)
		}
	}
	if cfg.artifact == (common.Address{}) {
		cfg.artifact = cfg.escrow
	}
	if *resumeJob >= 0 {
		cfg.resumeJobID = big.NewInt(*resumeJob)
	}
	cfg.principal = big.NewInt(*principal)
	if cfg.acceptDelay <= 0 || cfg.submitDelay <= cfg.acceptDelay || cfg.graceDelay <= cfg.submitDelay || cfg.verifyTimeout <= 0 {
		fatalf("invalid deadlines: require 0 < accept-delay < submit-delay < grace-delay and verification-timeout > 0")
	}
	required("rpc", cfg.rpcURL)
	required("proxy", cfg.proxyURL)
	required("private-key", cfg.privKey)
	required("tee-x", cfg.teePublicX)
	required("tee-y", cfg.teePublicY)
	if cfg.escrow == (common.Address{}) {
		fatalf("missing escrow address")
	}
	if cfg.contractor == (common.Address{}) {
		fatalf("missing contractor address")
	}
	if cfg.artifact == (common.Address{}) {
		fatalf("missing artifact address")
	}
	return cfg
}

func resumeFromReady(ctx context.Context, client *ethclient.Client, escrow *bind.BoundContract, key *ecdsa.PrivateKey, chainID *big.Int, cfg configValues) {
	id := cfg.resumeJobID
	job := getJob(ctx, escrow, id)
	if job.State != 3 {
		fatalf("resume-job %s is in state %d, expected ReadyToVerify state 3", id, job.State)
	}
	auth := authFor(ctx, client, key, chainID, big.NewInt(feeWei))
	tx := transact(ctx, client, escrow, auth, "dispatchVerification", id)
	fmt.Printf("dispatchVerification tx: %s\n", tx.Hash())
	job = getJob(ctx, escrow, id)
	instructionID := common.BytesToHash(job.Current.InstructionId[:])
	fmt.Printf("instructionId: %s expectedTee=%s\n", instructionID.Hex(), job.Terms.ExpectedTee.Hex())

	result, err := pollActionResult(cfg.proxyURL, instructionID, cfg.settleTag, cfg.pollTimeout)
	check(err, "poll action result")
	fmt.Printf("action result: tag=%s status=%d log=%q dataBytes=%d\n", result.Result.SubmissionTag, result.Result.Status, result.Result.Log, len(result.Result.Data))
	if result.Result.Status != 1 {
		fatalf("TEE returned status=%d; not settling an unsuccessful action result", result.Result.Status)
	}
	auth = authFor(ctx, client, key, chainID, nil)
	tx = transact(ctx, client, escrow, auth, "settleAttempt", id, []byte(result.Result.Data), [32]byte(result.Result.OPType), [32]byte(result.Result.OPCommand), string(result.Result.SubmissionTag), result.Result.Status, []byte(result.Signature))
	fmt.Printf("settleAttempt tx: %s\n", tx.Hash())
	job = getJob(ctx, escrow, id)
	fmt.Printf("final job state=%d settled=%t\n", job.State, job.Settled)
}

func deadlines(now uint64, cfg configValues) (uint64, uint64, uint64) {
	return now + uint64(cfg.acceptDelay.Seconds()),
		now + uint64(cfg.submitDelay.Seconds()),
		now + uint64(cfg.graceDelay.Seconds())
}

func waitForGrace(graceEnds uint64) {
	target := time.Unix(int64(graceEnds)+15, 0)
	if sleep := time.Until(target); sleep > 0 {
		fmt.Printf("waiting %s for graceEnds to pass\n", sleep.Round(time.Second))
		time.Sleep(sleep)
	}
}

func buildBundle(specHash common.Hash, failVector bool) ([]byte, common.Hash, error) {
	maxBytes := 100_000
	if failVector {
		maxBytes = 1
	}
	b := verifier.WorkProofBundle{
		FormatVersion:    1,
		TemplateID:       "phase5-code-size",
		TemplateVersion:  "v1.0.0",
		TargetChainID:    114,
		PublicSpecHash:   specHash.Hex(),
		VectorCount:      1,
		SelectionCount:   1,
		GasLimitPerCall:  100_000,
		TimeoutMsPerCall: 1000,
		MaxResponseBytes: 1024,
		Vectors: []verifier.Vector{{
			ID:       "artifact-has-expected-code-size",
			Type:     verifier.VectorCodeSizeRange,
			MinBytes: 1,
			MaxBytes: maxBytes,
		}},
	}
	if err := b.Validate(); err != nil {
		return nil, common.Hash{}, err
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, common.Hash{}, err
	}
	canonical, err := jcs.Transform(raw)
	if err != nil {
		return nil, common.Hash{}, err
	}
	return canonical, crypto.Keccak256Hash(canonical), nil
}

func encryptForTEE(plaintext []byte, xHex, yHex string) ([]byte, error) {
	x, ok := new(big.Int).SetString(strings.TrimPrefix(xHex, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("invalid tee public x")
	}
	y, ok := new(big.Int).SetString(strings.TrimPrefix(yHex, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("invalid tee public y")
	}
	return teeutils.Encrypt(plaintext, &ecdsa.PublicKey{Curve: crypto.S256(), X: x, Y: y})
}

func waitForRandomness(ctx context.Context, client *ethclient.Client, escrow *bind.BoundContract, key *ecdsa.PrivateKey, chainID, id *big.Int) {
	deadline := time.Now().Add(12 * time.Minute)
	for {
		auth := authFor(ctx, client, key, chainID, nil)
		tx, err := escrow.Transact(auth, "lockRandomness", id)
		if err == nil {
			receipt, err := bind.WaitMined(ctx, client, tx)
			check(err, "wait lockRandomness")
			if receipt.Status == ethtypes.ReceiptStatusSuccessful {
				fmt.Printf("lockRandomness tx: %s\n", tx.Hash())
				return
			}
		}
		if time.Now().After(deadline) {
			fatalf("randomness was not ready before timeout")
		}
		time.Sleep(20 * time.Second)
	}
}

func getJob(ctx context.Context, escrow *bind.BoundContract, id *big.Int) jobView {
	var out []any
	check(escrow.Call(&bind.CallOpts{Context: ctx}, &out, "getJob", id), "getJob")
	if len(out) != 1 {
		fatalf("getJob returned %d values", len(out))
	}
	job, ok := out[0].(jobView)
	if !ok {
		converted := *abi.ConvertType(out[0], new(jobView)).(*jobView)
		return converted
	}
	return job
}

func pollActionResult(proxyURL string, instructionID common.Hash, tag string, timeout time.Duration) (*teetypes.ActionResponse, error) {
	deadline := time.Now().Add(timeout)
	url := strings.TrimRight(proxyURL, "/") + "/action/result/" + instructionID.Hex() + "?submissionTag=" + tag
	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var ar teetypes.ActionResponse
			if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
				return nil, err
			}
			return &ar, nil
		}
		if resp != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
			_ = resp.Body.Close()
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for %s tag %s", instructionID.Hex(), tag)
		}
		time.Sleep(5 * time.Second)
	}
}

func transact(ctx context.Context, client *ethclient.Client, contract *bind.BoundContract, auth *bind.TransactOpts, method string, args ...any) *ethtypes.Transaction {
	tx, err := contract.Transact(auth, method, args...)
	check(err, method)
	receipt, err := bind.WaitMined(ctx, client, tx)
	check(err, "wait "+method)
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		fatalf("%s reverted: %s", method, tx.Hash())
	}
	return tx
}

func authFor(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, chainID *big.Int, value *big.Int) *bind.TransactOpts {
	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	check(err, "transactor")
	nonce, err := client.PendingNonceAt(ctx, auth.From)
	check(err, "nonce")
	auth.Nonce = new(big.Int).SetUint64(nonce)
	auth.Context = ctx
	auth.Value = value
	return auth
}

func callAddress(ctx context.Context, contract *bind.BoundContract, method string, args ...any) common.Address {
	var out []any
	check(contract.Call(&bind.CallOpts{Context: ctx}, &out, method, args...), method)
	return out[0].(common.Address)
}

func callBig(ctx context.Context, contract *bind.BoundContract, method string, args ...any) *big.Int {
	var out []any
	check(contract.Call(&bind.CallOpts{Context: ctx}, &out, method, args...), method)
	return out[0].(*big.Int)
}

func callUint16(ctx context.Context, contract *bind.BoundContract, method string, args ...any) uint16 {
	var out []any
	check(contract.Call(&bind.CallOpts{Context: ctx}, &out, method, args...), method)
	return out[0].(uint16)
}

func mustABI(raw string) abi.ABI {
	parsed, err := abi.JSON(bytes.NewReader([]byte(raw)))
	if err != nil {
		panic(err)
	}
	return parsed
}

func parsePrivateKey(raw string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(strings.TrimPrefix(raw, "0x"))
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

func required(name, value string) {
	if value == "" {
		fatalf("missing %s", name)
	}
}

func check(err error, label string) {
	if err != nil {
		fatalf("%s: %v", label, err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

type addressValue struct{ target *common.Address }

func (v addressValue) String() string {
	if v.target == nil {
		return ""
	}
	return v.target.Hex()
}

func (v addressValue) Set(raw string) error {
	if !common.IsHexAddress(raw) {
		return fmt.Errorf("invalid address %q", raw)
	}
	*v.target = common.HexToAddress(raw)
	return nil
}

type hashValue struct{ target *common.Hash }

func (v hashValue) String() string {
	if v.target == nil {
		return ""
	}
	return v.target.Hex()
}

func (v hashValue) Set(raw string) error {
	decoded, err := hexutil.Decode(raw)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("invalid bytes32 %q", raw)
	}
	*v.target = common.BytesToHash(decoded)
	return nil
}

var _ = hex.EncodeToString
var _ = hexutil.Encode
