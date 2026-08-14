// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {FccVerdict} from "./lib/FccVerdict.sol";
import {ITeeExtensionRegistry} from "./interfaces/ITeeExtensionRegistry.sol";
import {ITeeMachineRegistry} from "./interfaces/ITeeMachineRegistry.sol";
import {ContractRegistry} from "flare-periphery/coston2/ContractRegistry.sol";
import {RandomNumberV2Interface} from "flare-periphery/coston2/RandomNumberV2Interface.sol";
import {IRelay} from "flare-periphery/coston2/IRelay.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/// @notice Phase 3 production escrow: registered FCC instruction sender for
/// WorkProof, following the scaffold's HelloWorldInstructionSender pattern
/// (constructor/setExtensionId/_getExtensionId mirror its "DO NOT MODIFY"
/// functions) plus the real FCC ActionResult signature chain from
/// contracts/lib/FccVerdict.sol (proven in test/FccSignatureSpike.t.sol).
///
/// FTestXRP is resolved live through Flare's ContractRegistry ->
/// AssetManagerFXRP.fAsset() at deployment, never pasted in as a constant
/// (plan section 10 "Address resolution"). `getVotingRoundId` is resolved
/// through the dedicated "Relay" registry entry; `getRandomNumberHistorical`
/// through the dedicated "RandomNumberV2" registry entry — on live Coston2
/// today these happen to be the same address, but the registry names are
/// resolved separately (both independently confirmed live via `cast call`)
/// so the two concerns stay correct if Flare ever splits them.
contract WorkProofEscrow is ReentrancyGuard {
    using SafeERC20 for IERC20;

    enum State {
        Created,
        Accepted,
        RandomnessPending,
        ReadyToVerify,
        Verifying,
        AwaitingResubmission,
        Retryable,
        Paid,
        Cancelled,
        Refunded,
        RefundPending
    }

    enum Outcome {
        Pass,
        Fail,
        Inconclusive
    }

    /// @dev Live status values from ITeeMachineRegistry.getTeeMachineStatus,
    /// confirmed live against the FlareTeeManager diamond.
    uint8 private constant TEE_STATUS_PRODUCTION = 2;

    /// @dev Job terms frozen at creation (threat model: "Client changes
    /// tests"). `expectedTee` is supplied by the client and pinned here
    /// because the client must encrypt the private bundle to exactly that
    /// TEE's public key before/at job creation. It never changes for the
    /// life of the job, across any number of resubmitted attempts.
    struct JobTerms {
        address client;
        address contractor;
        address expectedTee;
        uint128 principal;
        uint128 fee;
        uint64 createdAt;
        uint64 acceptBy;
        uint64 submitBy;
        uint64 graceEnds;
        uint64 verificationTimeout;
        bytes32 specHash;
        bytes32 privateBundleHash;
        bytes32 engineVersionHash;
        // Content-addressed locator for the encrypted bundle (SPEC.md Job
        // Terms: "ciphertext hash and content-addressed locator"). Only the
        // hash is stored on-chain -- the fetch URL is deterministically
        // `https://{engine-configured allowlisted gateway}/{ciphertextHash}`,
        // so "content-addressed" already implies the path without needing a
        // second, more expensive on-chain string field.
        bytes32 ciphertextHash;
    }

    /// @dev Mutable per-attempt state; reset whenever a fresh attempt is
    /// submitted so a settled/dispatched verdict from a stale attempt can
    /// never match again.
    struct AttemptState {
        uint64 attempt;
        address artifactAddress;
        uint256 artifactBlock;
        bytes32 artifactCodeHash;
        uint256 targetRound;
        uint256 randomRound;
        bytes32 randomValueHash;
        bool randomLocked;
        bytes32 instructionId;
        uint64 dispatchedAt;
        uint64 timeoutAt;
    }

    struct Job {
        JobTerms terms;
        AttemptState current;
        State state;
        bool settled;
    }

    /// @notice Mirrors packages/schema/schemas/verdict-v1.schema.json field-for-field.
    /// Split into two nested static sub-structs purely to keep solc's IR
    /// decoder under its per-call stack budget for a large abi.decode — both
    /// sub-structs contain only static (value) types, so ABI encoding is
    /// byte-identical to a single flat tuple; the Go verifier does not need
    /// to know about this split.
    struct VerdictIdentity {
        uint8 schemaVersion;
        address escrowAddress;
        uint256 chainId;
        uint256 jobId;
        uint64 attempt;
        bytes32 instructionId;
        bytes32 specHash;
        bytes32 privateBundleHash;
        address artifactAddress;
        uint256 artifactBlock;
    }

    struct VerdictOutcome {
        bytes32 artifactCodeHash;
        uint256 randomRound;
        bytes32 randomValueHash;
        bytes32 engineVersionHash;
        Outcome outcome;
        uint32 passedCount;
        uint32 executedCount;
        bytes32 reportHash;
        uint64 issuedAt;
        uint64 expiresAt;
    }

    struct VerdictV1 {
        VerdictIdentity id;
        VerdictOutcome result;
    }

    /// @dev Payload sent through TeeExtensionRegistry.sendInstructions; the
    /// Go extension decodes this to know what to verify (Phase 4).
    struct WorkProofInstruction {
        uint256 chainId;
        address escrowAddress;
        uint256 jobId;
        uint64 attempt;
        bytes32 specHash;
        bytes32 privateBundleHash;
        bytes32 ciphertextHash;
        address artifactAddress;
        uint256 artifactBlock;
        bytes32 artifactCodeHash;
        uint256 randomRound;
        bytes32 randomValueHash;
        bytes32 engineVersionHash;
        uint64 expiresAt;
    }

    // forge-lint: disable-next-line(unsafe-typecast)
    bytes32 public constant OP_TYPE = bytes32("WORKPROOF");
    // forge-lint: disable-next-line(unsafe-typecast)
    bytes32 public constant OP_COMMAND = bytes32("VERIFY");

    uint16 public constant MAX_PROTOCOL_FEE_BPS = 1_000; // 10% hard cap
    uint64 public constant MIN_VERIFICATION_TIMEOUT = 30 seconds;
    uint64 public constant MAX_VERIFICATION_TIMEOUT = 7 days;
    uint256 private constant FIRST_PUBLIC_EXTENSION_ID = 0x10000;

    IERC20 public immutable token;
    ITeeExtensionRegistry public immutable TEE_EXTENSION_REGISTRY;
    ITeeMachineRegistry public immutable TEE_MACHINE_REGISTRY;
    IRelay public immutable RELAY;
    RandomNumberV2Interface public immutable RANDOM_NUMBER_V2;
    address public immutable treasury;
    address public owner;
    address public pendingOwner;

    uint16 public protocolFeeBps;
    bool public paused;
    uint256 private _extensionId;
    uint256 public nextJobId;
    // internal, not public: a large struct's auto-generated public getter
    // hits the same solc IR stack-depth limit as a large abi.decode; use the
    // explicit getJob() accessor below instead.
    mapping(uint256 => Job) internal jobs;

    error InvalidState();
    error Deadline();
    error InvalidVerdict();
    error NotOwner();
    error IsPaused();
    error ZeroAddress();
    error NoCode();
    error FeeTooHigh();
    error RandomNotReady();
    error TeeNotProduction();

    event OwnershipTransferStarted(address indexed currentOwner, address indexed pendingOwner);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event JobCreated(
        uint256 indexed jobId,
        address indexed client,
        address indexed contractor,
        address expectedTee,
        uint128 principal,
        uint128 fee,
        uint64 acceptBy,
        uint64 submitBy,
        uint64 graceEnds,
        bytes32 specHash,
        bytes32 privateBundleHash,
        bytes32 engineVersionHash
    );
    event JobAccepted(uint256 indexed jobId);
    event JobCancelled(uint256 indexed jobId);
    event AttemptSubmitted(
        uint256 indexed jobId,
        uint64 indexed attempt,
        address artifactAddress,
        uint256 artifactBlock,
        bytes32 artifactCodeHash,
        uint256 targetRound
    );
    event RandomnessLocked(uint256 indexed jobId, uint64 indexed attempt, uint256 randomRound, bytes32 randomValueHash);
    event InsecureRandomRoundSkipped(uint256 indexed jobId, uint64 indexed attempt, uint256 skippedRound);
    event VerificationDispatched(
        uint256 indexed jobId, uint64 indexed attempt, bytes32 instructionId, address expectedTee, uint64 timeoutAt
    );
    event VerificationTimedOut(uint256 indexed jobId, uint64 indexed attempt, bytes32 oldInstructionId);
    event AttemptSettled(uint256 indexed jobId, uint64 indexed attempt, Outcome outcome);
    event PaymentReleased(uint256 indexed jobId, address indexed contractor, uint256 principal, uint256 fee);
    event JobRefunded(uint256 indexed jobId, address indexed client, uint256 amount);
    event ProtocolFeeChangedForFutureJobs(uint16 oldBps, uint16 newBps);
    event NewWorkPaused();
    event NewWorkUnpaused();

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier whenNotPaused() {
        if (paused) revert IsPaused();
        _;
    }

    /// @param _teeExtensionRegistry Address of the TEE extension registry (FlareTeeManager diamond on Coston2).
    /// @param _teeMachineRegistry Address of the TEE machine registry (same diamond address).
    /// @param _treasury Real, non-zero fee-collection address. Never a hash-derived placeholder — nobody holds
    /// the key for `address(uint160(uint256(keccak256(...))))`, so a job's success fee would be permanently lost.
    /// @param _initialProtocolFeeBps Starting protocol fee for future jobs, capped at MAX_PROTOCOL_FEE_BPS.
    constructor(
        ITeeExtensionRegistry _teeExtensionRegistry,
        ITeeMachineRegistry _teeMachineRegistry,
        address _treasury,
        uint16 _initialProtocolFeeBps
    ) {
        if (
            address(_teeExtensionRegistry) == address(0) || address(_teeMachineRegistry) == address(0)
                || _treasury == address(0)
        ) revert ZeroAddress();
        if (address(_teeExtensionRegistry).code.length == 0 || address(_teeMachineRegistry).code.length == 0) {
            revert NoCode();
        }
        if (_initialProtocolFeeBps > MAX_PROTOCOL_FEE_BPS) revert FeeTooHigh();

        TEE_EXTENSION_REGISTRY = _teeExtensionRegistry;
        TEE_MACHINE_REGISTRY = _teeMachineRegistry;

        // Resolve FTestXRP live through the registry -> AssetManagerFXRP.fAsset().
        // A real on-chain call at deploy time, not a pasted constant (plan
        // section 10 "Address resolution").
        IERC20 resolvedToken = ContractRegistry.getAssetManagerFXRP().fAsset();
        if (address(resolvedToken) == address(0)) revert ZeroAddress();
        token = resolvedToken;

        RELAY = ContractRegistry.getRelay();
        if (address(RELAY) == address(0)) revert ZeroAddress();
        RANDOM_NUMBER_V2 = ContractRegistry.getRandomNumberV2();
        if (address(RANDOM_NUMBER_V2) == address(0)) revert ZeroAddress();

        if (_treasury == address(this) || _treasury == address(resolvedToken)) revert ZeroAddress();

        owner = msg.sender;
        treasury = _treasury;
        protocolFeeBps = _initialProtocolFeeBps;
    }

    /// @notice Finds and sets this contract's extension id. Can only be set once.
    /// DO NOT MODIFY this function (mirrors HelloWorldInstructionSender exactly).
    function setExtensionId() external {
        require(_extensionId == 0, "Extension ID already set.");

        uint256 c = TEE_EXTENSION_REGISTRY.nextPublicExtensionId();
        for (uint256 i = FIRST_PUBLIC_EXTENSION_ID; i < c; ++i) {
            if (TEE_EXTENSION_REGISTRY.getTeeExtensionInstructionsSender(i) == address(this)) {
                _extensionId = i;
                return;
            }
        }
        revert("Extension ID not found.");
    }

    function _getExtensionId() internal view returns (uint256) {
        require(_extensionId != 0, "Extension ID is not set.");
        return _extensionId;
    }

    /// @dev Reverts unless exactly one non-zero, live PRODUCTION machine is
    /// confirmed. Used both to select the pinned TEE at job creation and to
    /// re-confirm it at dispatch — never to silently swap it.
    function _confirmProductionTee(address teeId) private view {
        if (teeId == address(0)) revert TeeNotProduction();
        if (TEE_MACHINE_REGISTRY.getTeeMachineStatus(teeId) != TEE_STATUS_PRODUCTION) revert TeeNotProduction();
    }

    /// @notice Named-field job accessor; the auto-generated `jobs(id)` getter
    /// would hit the same IR stack-depth limit as a large abi.decode.
    function getJob(uint256 id) external view returns (Job memory) {
        return jobs[id];
    }

    // ---------------------------------------------------------------------
    // Owner controls: may pause NEW exposure and set a capped future fee.
    // May NOT mutate jobs, substitute TEE results, change a job's token, or
    // withdraw accounted principal. Pause never blocks progress of an
    // already-created job (accept/lock/dispatch/settle/expire/cancel/refund)
    // — only createJob and submitAttempt (new jobs/new attempts) are gated.
    // ---------------------------------------------------------------------

    function pauseNewWork() external onlyOwner {
        paused = true;
        emit NewWorkPaused();
    }

    function unpauseNewWork() external onlyOwner {
        paused = false;
        emit NewWorkUnpaused();
    }

    function setProtocolFee(uint16 newBps) external onlyOwner {
        if (newBps > MAX_PROTOCOL_FEE_BPS) revert FeeTooHigh();
        emit ProtocolFeeChangedForFutureJobs(protocolFeeBps, newBps);
        protocolFeeBps = newBps;
    }

    function transferOwnership(address newOwner) external onlyOwner {
        if (newOwner == address(0)) revert ZeroAddress();
        pendingOwner = newOwner;
        emit OwnershipTransferStarted(owner, newOwner);
    }

    function acceptOwnership() external {
        if (msg.sender != pendingOwner) revert NotOwner();
        address previousOwner = owner;
        owner = msg.sender;
        pendingOwner = address(0);
        emit OwnershipTransferred(previousOwner, msg.sender);
    }

    // ---------------------------------------------------------------------
    // Job lifecycle
    // ---------------------------------------------------------------------

    function createJob(
        address contractor,
        address expectedTee,
        uint128 principal,
        uint64 acceptBy,
        uint64 submitBy,
        uint64 graceEnds,
        uint64 verificationTimeout,
        bytes32 specHash,
        bytes32 privateBundleHash,
        bytes32 engineVersionHash,
        bytes32 ciphertextHash
    ) external whenNotPaused nonReentrant returns (uint256 id) {
        if (
            contractor == address(0) || expectedTee == address(0) || acceptBy <= block.timestamp || submitBy <= acceptBy
                || graceEnds <= submitBy || verificationTimeout < MIN_VERIFICATION_TIMEOUT
                || verificationTimeout > MAX_VERIFICATION_TIMEOUT || principal == 0
        ) revert InvalidVerdict();

        _getExtensionId();
        _confirmProductionTee(expectedTee);

        uint128 fee = uint128((uint256(principal) * protocolFeeBps) / 10_000);
        token.safeTransferFrom(msg.sender, address(this), uint256(principal) + fee);

        id = nextJobId++;
        Job storage j = jobs[id];
        j.terms = JobTerms({
            client: msg.sender,
            contractor: contractor,
            expectedTee: expectedTee,
            principal: principal,
            fee: fee,
            createdAt: uint64(block.timestamp),
            acceptBy: acceptBy,
            submitBy: submitBy,
            graceEnds: graceEnds,
            verificationTimeout: verificationTimeout,
            specHash: specHash,
            privateBundleHash: privateBundleHash,
            engineVersionHash: engineVersionHash,
            ciphertextHash: ciphertextHash
        });
        j.state = State.Created;

        emit JobCreated(
            id,
            msg.sender,
            contractor,
            expectedTee,
            principal,
            fee,
            acceptBy,
            submitBy,
            graceEnds,
            specHash,
            privateBundleHash,
            engineVersionHash
        );
    }

    function acceptJob(uint256 id) external {
        Job storage j = jobs[id];
        if (j.state != State.Created || msg.sender != j.terms.contractor) revert InvalidState();
        if (block.timestamp > j.terms.acceptBy) revert Deadline();
        j.state = State.Accepted;
        emit JobAccepted(id);
    }

    /// @notice Returns tokens to the client whenever the contractor never
    /// accepted — whether the client cancels early or the acceptance window
    /// has simply expired unaccepted; accept() itself already blocks any
    /// acceptance past acceptBy, so State.Created is a sufficient guard.
    function cancelUnaccepted(uint256 id) external nonReentrant {
        Job storage j = jobs[id];
        // `j.state != State.Created` alone is sufficient: every path that
        // sets `settled = true` also moves state away from Created in the
        // same statement, so a separate `j.settled` check here is
        // unreachable dead code, not extra defense (removed after
        // confirming this by inspection: cancelUnaccepted/settleAttempt's
        // Pass branch/refundExpired are the only three places settled ever
        // becomes true, and each also reassigns state in that same call).
        if (j.state != State.Created || msg.sender != j.terms.client) revert InvalidState();
        j.state = State.Cancelled;
        j.settled = true;
        emit JobCancelled(id);
        token.safeTransfer(j.terms.client, uint256(j.terms.principal) + j.terms.fee);
    }

    /// @notice Contract-observed artifact identity: extcodehash is read
    /// directly from the chain, never trusted from caller input. Gated by
    /// pause (a new attempt is new exposure), but NOT by anything about the
    /// job's pinned TEE — that was already fixed at creation.
    function submitAttempt(uint256 id, address artifactAddress) external whenNotPaused nonReentrant {
        Job storage j = jobs[id];
        if (
            msg.sender != j.terms.contractor
                || (j.state != State.Accepted && j.state != State.AwaitingResubmission && j.state != State.Retryable)
        ) revert InvalidState();
        if (block.timestamp > j.terms.submitBy) revert Deadline();
        if (artifactAddress == address(0) || artifactAddress.code.length == 0) revert InvalidVerdict();

        uint64 attempt = j.current.attempt + 1;
        uint256 targetRound = RELAY.getVotingRoundId(block.timestamp) + 1;

        j.current = AttemptState({
            attempt: attempt,
            artifactAddress: artifactAddress,
            artifactBlock: block.number,
            artifactCodeHash: artifactAddress.codehash,
            targetRound: targetRound,
            randomRound: 0,
            randomValueHash: bytes32(0),
            randomLocked: false,
            instructionId: bytes32(0),
            dispatchedAt: 0,
            timeoutAt: 0
        });
        j.state = State.RandomnessPending;

        emit AttemptSubmitted(id, attempt, artifactAddress, block.number, artifactAddress.codehash, targetRound);
    }

    /// @notice Fetches only the committed historical round; accepts only
    /// secure randomness; deterministically advances by exactly one round
    /// on an insecure result. Callable by anyone (permissionless progress);
    /// never blocked by pause -- this only continues an attempt that
    /// already exists, it creates no new exposure.
    function lockRandomness(uint256 id) external nonReentrant {
        Job storage j = jobs[id];
        if (j.state != State.RandomnessPending) revert InvalidState();

        uint256 randomNumber;
        bool isSecure;
        try RANDOM_NUMBER_V2.getRandomNumberHistorical(j.current.targetRound) returns (
            uint256 rn, bool secure, uint256
        ) {
            randomNumber = rn;
            isSecure = secure;
        } catch {
            revert RandomNotReady();
        }

        if (!isSecure) {
            emit InsecureRandomRoundSkipped(id, j.current.attempt, j.current.targetRound);
            j.current.targetRound += 1;
            return;
        }

        j.current.randomRound = j.current.targetRound;
        j.current.randomValueHash = keccak256(abi.encode(randomNumber));
        j.current.randomLocked = true;
        j.state = State.ReadyToVerify;

        emit RandomnessLocked(id, j.current.attempt, j.current.randomRound, j.current.randomValueHash);
    }

    /// @notice Re-confirms the job-pinned TEE is still a live PRODUCTION
    /// machine (never re-selects a different one), sends the VERIFY
    /// instruction through the real extension registry, and starts the
    /// dispatch timeout. Never blocked by pause, for the same liveness
    /// reason as lockRandomness.
    function dispatchVerification(uint256 id) external payable nonReentrant {
        Job storage j = jobs[id];
        if (j.state != State.ReadyToVerify || !j.current.randomLocked) revert InvalidState();

        _confirmProductionTee(j.terms.expectedTee);

        WorkProofInstruction memory instruction = WorkProofInstruction({
            chainId: block.chainid,
            escrowAddress: address(this),
            jobId: id,
            attempt: j.current.attempt,
            specHash: j.terms.specHash,
            privateBundleHash: j.terms.privateBundleHash,
            ciphertextHash: j.terms.ciphertextHash,
            artifactAddress: j.current.artifactAddress,
            artifactBlock: j.current.artifactBlock,
            artifactCodeHash: j.current.artifactCodeHash,
            randomRound: j.current.randomRound,
            randomValueHash: j.current.randomValueHash,
            engineVersionHash: j.terms.engineVersionHash,
            expiresAt: j.terms.graceEnds
        });

        address[] memory teeIds = new address[](1);
        teeIds[0] = j.terms.expectedTee;
        address[] memory cosigners = new address[](0);
        ITeeExtensionRegistry.TeeInstructionParams memory params = ITeeExtensionRegistry.TeeInstructionParams({
            opType: OP_TYPE,
            opCommand: OP_COMMAND,
            message: abi.encode(instruction),
            cosigners: cosigners,
            cosignersThreshold: 0,
            claimBackAddress: msg.sender
        });

        // State is committed to State.Verifying (below) as soon as this
        // external, payable, potentially-reentrant call returns; nonReentrant
        // additionally blocks any reentrant call into this or any other
        // fund-moving function for the duration.
        bytes32 instructionId = TEE_EXTENSION_REGISTRY.sendInstructions{value: msg.value}(teeIds, params);

        j.current.instructionId = instructionId;
        j.current.dispatchedAt = uint64(block.timestamp);
        uint256 rawTimeoutAt = block.timestamp + uint256(j.terms.verificationTimeout);
        uint64 timeoutAt = j.terms.graceEnds;
        if (rawTimeoutAt < j.terms.graceEnds) {
            // forge-lint: disable-next-line(unsafe-typecast)
            timeoutAt = uint64(rawTimeoutAt);
        }
        if (timeoutAt <= block.timestamp) revert Deadline();

        j.current.timeoutAt = timeoutAt;
        j.state = State.Verifying;

        emit VerificationDispatched(id, j.current.attempt, instructionId, j.terms.expectedTee, j.current.timeoutAt);
    }

    /// @notice Makes a timed-out dispatched verification retryable without
    /// moving principal, and invalidates the old instruction generation so a
    /// late-arriving signed result can never settle it.
    function expireVerification(uint256 id) external {
        Job storage j = jobs[id];
        if (j.state != State.Verifying || block.timestamp <= j.current.timeoutAt) revert InvalidState();

        bytes32 oldInstructionId = j.current.instructionId;
        j.current.instructionId = bytes32(0);
        j.current.dispatchedAt = 0;
        j.current.timeoutAt = 0;
        j.state = State.Retryable;

        emit VerificationTimedOut(id, j.current.attempt, oldInstructionId);
    }

    /// @param data ABI-encoded VerdictV1, i.e. the real ActionResult.data the FCC node returned.
    /// @param opType routing-consistency check only, never settlement authority (plan section 9 step 4).
    /// @param opCommand routing-consistency check only, never settlement authority.
    /// @param submissionTag tee-node SubmissionTag ("threshold" expected for the finalized result); part of the signed hash.
    /// @param status raw ActionResult.status; only 1 (handler completed) can ever settle.
    function settleAttempt(
        uint256 id,
        bytes calldata data,
        bytes32 opType,
        bytes32 opCommand,
        string calldata submissionTag,
        uint8 status,
        bytes calldata signature
    ) external nonReentrant {
        Job storage j = jobs[id];
        // `j.state != State.Verifying` alone is sufficient: `settled` only
        // ever becomes true in the same statement that moves state away
        // from Verifying (the Pass branch below, cancelUnaccepted, or
        // refundExpired), so a separate `j.settled` check here would be
        // unreachable dead code -- removed, not left as inert
        // "defense in depth" (see cancelUnaccepted's identical reasoning).
        if (j.state != State.Verifying) revert InvalidState();
        // A timed-out instruction must never settle, even if
        // expireVerification has not yet been explicitly called.
        if (block.timestamp > j.current.timeoutAt) revert InvalidState();
        // Once the client's refund deadline has passed, settlement of ANY
        // outcome (not only Pass) must stop -- graceEnds is an absolute
        // entitlement boundary, not merely a race with refundExpired.
        // Without this, timeoutAt (= dispatchedAt + verificationTimeout,
        // caller-chosen and unbounded relative to graceEnds) could let a
        // late Pass pay the contractor after the point where the client was
        // already entitled to a refund via refundExpired.
        if (block.timestamp > j.terms.graceEnds) revert InvalidState();
        if (status != 1) revert InvalidVerdict();
        if (opType != OP_TYPE || opCommand != OP_COMMAND) revert InvalidVerdict();
        if (keccak256(bytes(submissionTag)) != keccak256(bytes("threshold"))) revert InvalidVerdict();
        if (j.current.instructionId == bytes32(0)) revert InvalidState();

        VerdictV1 memory v = _decodeAndAuthenticate(data, submissionTag, status, signature, j.terms.expectedTee);
        _checkBinding(j, v, id);

        emit AttemptSettled(id, j.current.attempt, v.result.outcome);

        if (v.result.outcome == Outcome.Fail) {
            j.state = block.timestamp > j.terms.submitBy ? State.RefundPending : State.AwaitingResubmission;
            return;
        }
        if (v.result.outcome == Outcome.Inconclusive) {
            j.state = block.timestamp > j.terms.submitBy ? State.RefundPending : State.Retryable;
            return;
        }

        // Outcome.Pass
        j.state = State.Paid;
        j.settled = true;
        emit PaymentReleased(id, j.terms.contractor, j.terms.principal, j.terms.fee);
        token.safeTransfer(j.terms.contractor, j.terms.principal);
        token.safeTransfer(treasury, j.terms.fee);
    }

    /// @dev Split out of settleAttempt() solely to keep per-function
    /// live-variable count low enough for the IR optimizer on a large struct decode.
    function _decodeAndAuthenticate(
        bytes calldata data,
        string calldata submissionTag,
        uint8 status,
        bytes calldata signature,
        address expectedTee
    ) private view returns (VerdictV1 memory v) {
        v = abi.decode(data, (VerdictV1));
        address signer =
            FccVerdict.recoverVerdictSigner(data, v.id.instructionId, submissionTag, status, block.chainid, signature);
        if (signer == address(0) || signer != expectedTee) revert InvalidVerdict();
    }

    /// @dev Every security-critical VerdictV1 field must match escrow storage
    /// before the outcome is trusted (SPEC.md "VerdictV1 Binding"). `expiresAt`
    /// must equal the job's stored grace deadline exactly — the TEE cannot
    /// claim a later expiry than what it was actually dispatched with.
    function _checkBinding(Job storage j, VerdictV1 memory v, uint256 id) private view {
        _checkIdentity(j, v.id, id);
        _checkOutcomeBinding(j, v.result);
    }

    function _checkIdentity(Job storage j, VerdictIdentity memory vi, uint256 id) private view {
        if (vi.schemaVersion != 1 || vi.escrowAddress != address(this) || vi.chainId != block.chainid || vi.jobId != id)
        {
            revert InvalidVerdict();
        }
        if (vi.attempt != j.current.attempt || vi.instructionId != j.current.instructionId) revert InvalidVerdict();
        if (vi.specHash != j.terms.specHash || vi.privateBundleHash != j.terms.privateBundleHash) {
            revert InvalidVerdict();
        }
        if (vi.artifactAddress != j.current.artifactAddress || vi.artifactBlock != j.current.artifactBlock) {
            revert InvalidVerdict();
        }
    }

    function _checkOutcomeBinding(Job storage j, VerdictOutcome memory vo) private view {
        if (j.current.artifactAddress.codehash != j.current.artifactCodeHash) revert InvalidVerdict();
        if (vo.artifactCodeHash != j.current.artifactCodeHash) revert InvalidVerdict();
        if (vo.randomRound != j.current.randomRound || vo.randomValueHash != j.current.randomValueHash) {
            revert InvalidVerdict();
        }
        if (vo.engineVersionHash != j.terms.engineVersionHash) revert InvalidVerdict();
        if (vo.reportHash == bytes32(0)) revert InvalidVerdict();
        if (vo.executedCount == 0 || vo.passedCount > vo.executedCount) revert InvalidVerdict();
        if (vo.outcome == Outcome.Pass && vo.passedCount != vo.executedCount) revert InvalidVerdict();
        if (vo.outcome != Outcome.Pass && vo.passedCount == vo.executedCount) revert InvalidVerdict();
        if (vo.expiresAt != j.terms.graceEnds) revert InvalidVerdict();
        if (vo.issuedAt > block.timestamp || vo.issuedAt < j.current.dispatchedAt) revert InvalidVerdict();
    }

    /// @notice After deadline + grace, returns principal and fee only to the
    /// client. Never blocked by pause.
    function refundExpired(uint256 id) external nonReentrant {
        Job storage j = jobs[id];
        if (j.settled || block.timestamp <= j.terms.graceEnds) revert InvalidState();
        j.state = State.Refunded;
        j.settled = true;
        uint256 amount = uint256(j.terms.principal) + j.terms.fee;
        emit JobRefunded(id, j.terms.client, amount);
        token.safeTransfer(j.terms.client, amount);
    }
}
