// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {FccVerdict} from "./lib/FccVerdict.sol";

interface IERC20Minimal {
    function transfer(address to, uint256 value) external returns (bool);
    function transferFrom(address from, address to, uint256 value) external returns (bool);
}

/// @notice P0 escrow state machine used by the Phase 2 local simulation lab.
/// settle() now binds the real FCC ActionResult signing chain (FccVerdict,
/// proven byte-exact against a real tee-node signature in
/// test/FccSignatureSpike.t.sol) and the full VerdictV1 field set from
/// SPEC.md, not a placeholder digest. Production deployment must additionally
/// wire registered TeeMachineRegistry/TeeExtensionRegistry checks (Phase 3)
/// before accepting funds on Coston2 — this contract trusts a constructor-
/// pinned `expectedTee` per job, not a live registry lookup.
contract WorkProofEscrow {
    enum State {
        Created,
        Accepted,
        AwaitingResubmission,
        Retryable,
        Paid,
        Cancelled,
        Refunded
    }
    enum Outcome {
        Pass,
        Fail,
        Inconclusive
    }

    struct Job {
        address client;
        address contractor;
        address expectedTee;
        uint128 principal;
        uint128 fee;
        uint64 submitBy;
        uint64 graceEnds;
        uint64 attempt;
        State state;
        bool settled;
        bool paused;
        // Job terms frozen at creation (threat model: "Client changes tests").
        bytes32 specHash;
        bytes32 privateBundleHash;
        bytes32 engineVersionHash;
        // Current-attempt artifact identity, frozen at submit().
        address artifactAddress;
        uint256 artifactBlock;
        bytes32 artifactCodeHash;
        // Current-attempt randomness/instruction binding, frozen at
        // lockRandomness() and cleared on every new submit() so a settled
        // verdict from a stale attempt can never match again.
        uint256 randomRound;
        bytes32 randomValueHash;
        bytes32 instructionId;
    }

    /// @notice Mirrors packages/schema/schemas/verdict-v1.schema.json field-for-field.
    /// This is the ABI-encoded payload the TEE returns as ActionResult.data;
    /// every security-critical field here is checked against job storage
    /// before any transfer, per SPEC.md "VerdictV1 Binding".
    ///
    /// Split into two nested sub-structs purely to keep solc's IR decoder
    /// under its per-call stack budget for a 20-field abi.decode — both
    /// sub-structs contain only static (value) types, so ABI encoding is
    /// byte-identical to a single flat 20-field tuple; off-chain encoders
    /// (the Go verifier) do not need to know about this split.
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

    IERC20Minimal public immutable token;
    address public owner;
    uint256 public nextJobId;
    // internal, not public: a 20-field struct's auto-generated public getter
    // hits the same solc IR stack-depth limit as a large abi.decode; use the
    // explicit getJob() accessor below instead.
    mapping(uint256 => Job) internal jobs;
    address public immutable treasury = address(uint160(uint256(keccak256("WORKPROOF_TREASURY"))));

    error InvalidState();
    error Deadline();
    error InvalidVerdict();
    error TransferFailed();
    error NotOwner();
    error Paused();

    constructor(IERC20Minimal token_) {
        token = token_;
        owner = msg.sender;
    }

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    /// @notice Named-field job accessor; the auto-generated `jobs(id)` getter
    /// returns a 20-element positional tuple that is easy to misalign in
    /// callers/tests, so this is the preferred read path.
    function getJob(uint256 id) external view returns (Job memory) {
        return jobs[id];
    }

    /// @notice Stops new jobs/attempts only. Never blocks refunds, cancellation,
    /// or a valid already-dispatched settlement — per the threat model's
    /// admin policy ("owner can stop new exposure but cannot decide outcomes").
    function setPaused(uint256 id, bool value) external onlyOwner {
        jobs[id].paused = value;
    }

    function createJob(
        address contractor,
        address expectedTee,
        uint128 principal,
        uint128 fee,
        uint64 submitBy,
        uint64 graceEnds,
        bytes32 specHash,
        bytes32 privateBundleHash,
        bytes32 engineVersionHash
    ) external returns (uint256 id) {
        if (
            submitBy <= block.timestamp || graceEnds <= submitBy || contractor == address(0)
                || expectedTee == address(0)
        ) revert InvalidVerdict();
        uint256 total = uint256(principal) + fee;
        if (!token.transferFrom(msg.sender, address(this), total)) revert TransferFailed();
        id = nextJobId++;
        Job storage j = jobs[id];
        j.client = msg.sender;
        j.contractor = contractor;
        j.expectedTee = expectedTee;
        j.principal = principal;
        j.fee = fee;
        j.submitBy = submitBy;
        j.graceEnds = graceEnds;
        j.state = State.Created;
        j.specHash = specHash;
        j.privateBundleHash = privateBundleHash;
        j.engineVersionHash = engineVersionHash;
    }

    function accept(uint256 id) external {
        Job storage j = jobs[id];
        if (j.paused) revert Paused();
        if (j.state != State.Created || msg.sender != j.contractor) revert InvalidState();
        j.state = State.Accepted;
    }

    function submit(uint256 id, address artifactAddress, uint256 artifactBlock, bytes32 artifactCodeHash) external {
        Job storage j = jobs[id];
        if (j.paused) revert Paused();
        if (
            msg.sender != j.contractor
                || (j.state != State.Accepted && j.state != State.AwaitingResubmission && j.state != State.Retryable)
        ) revert InvalidState();
        if (block.timestamp > j.submitBy) revert Deadline();
        if (artifactAddress == address(0)) revert InvalidVerdict();
        j.attempt++;
        j.artifactAddress = artifactAddress;
        j.artifactBlock = artifactBlock;
        j.artifactCodeHash = artifactCodeHash;
        // A new attempt must be re-locked to a fresh instruction before it
        // can settle; this is what makes an old signed verdict stale after
        // resubmission (SPEC.md "Old valid result after retry").
        j.randomRound = 0;
        j.randomValueHash = bytes32(0);
        j.instructionId = bytes32(0);
    }

    /// @notice Locks the deterministic future-round commitment for the
    /// current attempt exactly once. Phase 2 lab stand-in for secure Flare
    /// randomness (Phase 5); anti-grinding property under test:
    /// "insecure randomness advances only one deterministic round".
    function lockRandomness(uint256 id, uint256 round, bytes32 valueHash) external {
        Job storage j = jobs[id];
        if (j.paused) revert Paused();
        if (j.settled || j.attempt == 0) revert InvalidState();
        if (j.instructionId != bytes32(0)) revert InvalidState();
        j.randomRound = round;
        j.randomValueHash = valueHash;
        j.instructionId = keccak256(abi.encode(address(this), block.chainid, id, j.attempt, round, valueHash));
    }

    /// @param data ABI-encoded VerdictV1, i.e. the real ActionResult.data the
    /// FCC node returned.
    /// @param submissionTag tee-node SubmissionTag ("submit" expected for a
    /// final result); part of the signed ActionResult hash.
    /// @param status raw ActionResult.status; only 1 (handler completed) can
    /// ever settle, per plan section 9 step 3.
    function settle(
        uint256 id,
        bytes calldata data,
        string calldata submissionTag,
        uint8 status,
        bytes calldata signature
    ) external {
        Job storage j = jobs[id];
        if (j.settled) revert InvalidVerdict();
        if (status != 1) revert InvalidVerdict();
        if (keccak256(bytes(submissionTag)) != keccak256(bytes("submit"))) revert InvalidVerdict();
        if (j.instructionId == bytes32(0)) revert InvalidState();

        VerdictV1 memory v = _decodeAndAuthenticate(data, submissionTag, status, signature, j.expectedTee);
        _checkBinding(j, v, id);

        if (v.result.outcome == Outcome.Fail) {
            j.state = State.AwaitingResubmission;
            return;
        }
        if (v.result.outcome == Outcome.Inconclusive) {
            j.state = State.Retryable;
            return;
        }

        // Outcome.Pass
        if (v.result.expiresAt < block.timestamp) revert InvalidVerdict();
        j.state = State.Paid;
        j.settled = true;
        if (!token.transfer(j.contractor, j.principal) || !token.transfer(treasury, j.fee)) revert TransferFailed();
    }

    /// @dev Split out of settle() solely to keep per-function live-variable
    /// count low enough for the IR optimizer on a 20-field struct decode.
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
    /// before the outcome is trusted (SPEC.md "VerdictV1 Binding"). Split
    /// across the two sub-structs for the same IR stack-depth reason as
    /// _decodeAndAuthenticate.
    function _checkBinding(Job storage j, VerdictV1 memory v, uint256 id) private view {
        _checkIdentity(j, v.id, id);
        _checkOutcomeBinding(j, v.result);
    }

    function _checkIdentity(Job storage j, VerdictIdentity memory vi, uint256 id) private view {
        if (vi.schemaVersion != 1 || vi.escrowAddress != address(this) || vi.chainId != block.chainid || vi.jobId != id)
        {
            revert InvalidVerdict();
        }
        if (vi.attempt != j.attempt || vi.instructionId != j.instructionId) revert InvalidVerdict();
        if (vi.specHash != j.specHash || vi.privateBundleHash != j.privateBundleHash) revert InvalidVerdict();
        if (vi.artifactAddress != j.artifactAddress || vi.artifactBlock != j.artifactBlock) revert InvalidVerdict();
    }

    function _checkOutcomeBinding(Job storage j, VerdictOutcome memory vo) private view {
        if (vo.artifactCodeHash != j.artifactCodeHash) revert InvalidVerdict();
        if (vo.randomRound != j.randomRound || vo.randomValueHash != j.randomValueHash) revert InvalidVerdict();
        if (vo.engineVersionHash != j.engineVersionHash) revert InvalidVerdict();
        if (vo.reportHash == bytes32(0)) revert InvalidVerdict();
    }

    function cancel(uint256 id) external {
        Job storage j = jobs[id];
        if (j.settled || j.state != State.Created || msg.sender != j.client) revert InvalidState();
        j.state = State.Cancelled;
        j.settled = true;
        if (!token.transfer(j.client, uint256(j.principal) + j.fee)) revert TransferFailed();
    }

    /// @notice Refunds are never blocked by pause — only new jobs/attempts are.
    function refund(uint256 id) external {
        Job storage j = jobs[id];
        if (j.settled || block.timestamp <= j.graceEnds) revert InvalidState();
        j.state = State.Refunded;
        j.settled = true;
        if (!token.transfer(j.client, uint256(j.principal) + j.fee)) revert TransferFailed();
    }
}
