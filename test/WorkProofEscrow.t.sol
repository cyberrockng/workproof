// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {WorkProofEscrow, IERC20Minimal} from "../contracts/WorkProofEscrow.sol";
import {FccVerdict} from "../contracts/lib/FccVerdict.sol";

contract MockToken is IERC20Minimal {
    mapping(address => uint256) public balanceOf;

    function mint(address to, uint256 amount) external {
        balanceOf[to] += amount;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        require(balanceOf[from] >= amount, "insufficient");
        balanceOf[from] -= amount;
        balanceOf[to] += amount;
        return true;
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        require(balanceOf[msg.sender] >= amount, "insufficient");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
}

/// @notice All ten Phase 2 "required simulations" from
/// WORKPROOF_EXECUTION_PLAN.md section 15, plus the accounting invariant,
/// against the real VerdictV1/FccVerdict binding in WorkProofEscrow.sol.
/// Verdicts here are dynamically built and signed with `vm.sign` against a
/// locally generated TEE key (a different, complementary proof style from
/// FccSignatureSpike.t.sol's fixed external Go-signed vector) so every
/// mutation case can be exercised, not just the one recorded fixture.
contract WorkProofEscrowTest is Test {
    MockToken token;
    WorkProofEscrow escrow;

    uint256 constant TEE_KEY = 0xA11CE;
    uint256 constant WRONG_TEE_KEY = 0xBAD;
    address teeAddr;
    address wrongTeeAddr;

    address constant CLIENT = address(0xC11E7);
    address constant CONTRACTOR = address(0xC0117AC706);

    bytes32 constant SPEC_HASH = keccak256("spec-v1");
    bytes32 constant BUNDLE_HASH = keccak256("bundle-v1");
    bytes32 constant ENGINE_HASH = keccak256("engine-v1");

    function setUp() external {
        token = new MockToken();
        escrow = new WorkProofEscrow(token);
        teeAddr = vm.addr(TEE_KEY);
        wrongTeeAddr = vm.addr(WRONG_TEE_KEY);
        token.mint(CLIENT, 1_000_000);
    }

    function _createJob(uint128 principal, uint128 fee) internal returns (uint256 id) {
        vm.prank(CLIENT);
        id = escrow.createJob(
            CONTRACTOR,
            teeAddr,
            principal,
            fee,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH
        );
    }

    function _acceptAndSubmit(uint256 id, address artifactAddress, uint256 artifactBlock, bytes32 artifactCodeHash)
        internal
    {
        vm.prank(CONTRACTOR);
        escrow.accept(id);
        vm.prank(CONTRACTOR);
        escrow.submit(id, artifactAddress, artifactBlock, artifactCodeHash);
    }

    function _lockRandomness(uint256 id, uint256 round, bytes32 valueHash) internal {
        escrow.lockRandomness(id, round, valueHash);
    }

    /// @dev Builds a VerdictV1 that exactly matches the job's current stored
    /// state (the "correct" verdict), leaving outcome/counts/report/expiry
    /// to the caller.
    function _matchingVerdict(uint256 id, WorkProofEscrow.Outcome outcome, uint64 expiresAt)
        internal
        view
        returns (WorkProofEscrow.VerdictV1 memory v)
    {
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        v.id = WorkProofEscrow.VerdictIdentity({
            schemaVersion: 1,
            escrowAddress: address(escrow),
            chainId: block.chainid,
            jobId: id,
            attempt: j.attempt,
            instructionId: j.instructionId,
            specHash: j.specHash,
            privateBundleHash: j.privateBundleHash,
            artifactAddress: j.artifactAddress,
            artifactBlock: j.artifactBlock
        });
        v.result = WorkProofEscrow.VerdictOutcome({
            artifactCodeHash: j.artifactCodeHash,
            randomRound: j.randomRound,
            randomValueHash: j.randomValueHash,
            engineVersionHash: j.engineVersionHash,
            outcome: outcome,
            passedCount: outcome == WorkProofEscrow.Outcome.Pass ? 5 : 3,
            executedCount: 5,
            reportHash: keccak256("report"),
            issuedAt: uint64(block.timestamp),
            expiresAt: expiresAt
        });
    }

    function _sign(uint256 key, bytes memory data, bytes32 instructionId, uint8 status)
        internal
        view
        returns (bytes memory signature)
    {
        bytes32 digest = FccVerdict.ethSignedHash(
            FccVerdict.payloadHash(block.chainid, FccVerdict.actionResultHash(data, instructionId, "submit", status))
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(key, digest);
        signature = abi.encodePacked(r, s, v);
    }

    function _settleWith(uint256 id, WorkProofEscrow.VerdictV1 memory v, uint256 key) internal {
        bytes memory data = abi.encode(v);
        bytes memory sig = _sign(key, data, v.id.instructionId, 1);
        escrow.settle(id, data, "submit", 1, sig);
    }

    // ---- 1. failing artifact returns FAIL; no balance moves ----
    function testSimulation1_FailOutcomeMovesNoBalance() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 42, keccak256("rand"));

        uint256 contractorBefore = token.balanceOf(CONTRACTOR);
        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Fail, uint64(block.timestamp + 10));
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.AwaitingResubmission));
        assertFalse(j.settled);
        assertEq(token.balanceOf(CONTRACTOR), contractorBefore);
    }

    // ---- 2. corrected resubmission; PASS transfers exactly the principal ----
    function testSimulation2_PassTransfersExactPrincipal() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xBAD1), 1, keccak256("codeBad"));
        _lockRandomness(id, 1, keccak256("r1"));
        WorkProofEscrow.VerdictV1 memory failV =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Fail, uint64(block.timestamp + 10));
        _settleWith(id, failV, TEE_KEY);

        // resubmission: attempt advances, artifact identity changes, prior
        // instruction is cleared and must be relocked.
        vm.prank(CONTRACTOR);
        escrow.submit(id, address(0xC0DE), 2, keccak256("codeGood"));
        _lockRandomness(id, 2, keccak256("r2"));

        WorkProofEscrow.VerdictV1 memory passV =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        _settleWith(id, passV, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Paid));
        assertTrue(j.settled);
        assertEq(token.balanceOf(CONTRACTOR), 100);
    }

    // ---- 3. bundle mutation fails commitment verification ----
    function testSimulation3_BundleHashMutationRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        v.id.privateBundleHash = keccak256("tampered-bundle");

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // ---- 4. artifact/address/code-hash mutation fails ----
    function testSimulation4_ArtifactMutationRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        v.result.artifactCodeHash = keccak256("someone-elses-code");

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // ---- 5a. wrong signer ----
    function testSimulation5a_WrongSignerRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, WRONG_TEE_KEY);
    }

    // ---- 5b. malformed signature ----
    function testSimulation5b_MalformedSignatureRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        bytes memory data = abi.encode(v);

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settle(id, data, "submit", 1, hex"1234");
    }

    // ---- 5c. wrong chain (verdict.chainId mismatch) ----
    function testSimulation5c_WrongChainRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        v.id.chainId = 999;

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // ---- 5d. wrong job ----
    function testSimulation5d_WrongJobRejected() external {
        uint256 id = _createJob(100, 7);
        uint256 otherId = _createJob(50, 3);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        v.id.jobId = otherId;

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // ---- 5e. wrong attempt ----
    function testSimulation5e_WrongAttemptRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        v.id.attempt = v.id.attempt + 1;

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // ---- 5f. wrong instruction ----
    function testSimulation5f_WrongInstructionRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        v.id.instructionId = keccak256("forged-instruction");

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // ---- 5g. replay ----
    function testSimulation5g_ReplayRejected() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        bytes memory data = abi.encode(v);
        bytes memory sig = _sign(TEE_KEY, data, v.id.instructionId, 1);
        escrow.settle(id, data, "submit", 1, sig);

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settle(id, data, "submit", 1, sig);
    }

    // ---- 6. insecure randomness advances only one deterministic round ----
    function testSimulation6_RandomnessLocksOnlyOnce() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.lockRandomness(id, 2, keccak256("r2"));

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(j.randomRound, 1);
    }

    // ---- 7. INCONCLUSIVE and timeout move no principal ----
    function testSimulation7_InconclusiveMovesNoPrincipal() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        uint256 before = token.balanceOf(CONTRACTOR);
        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Inconclusive, uint64(block.timestamp + 10));
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Retryable));
        assertEq(token.balanceOf(CONTRACTOR), before);
    }

    function testSimulation7b_TimeoutBeforeSubmitBlocksSubmission() external {
        uint256 id = _createJob(100, 7);
        vm.prank(CONTRACTOR);
        escrow.accept(id);

        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.submitBy) + 1);

        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.Deadline.selector);
        escrow.submit(id, address(0xA1), 1, keccak256("codeA"));
    }

    // ---- 8. expiry returns only client principal and locked fee ----
    function testSimulation8_ExpiryRefundsClientOnly() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));

        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.graceEnds) + 1);

        uint256 clientBefore = token.balanceOf(CLIENT);
        escrow.refund(id);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Refunded));
        assertTrue(j.settled);
        assertEq(token.balanceOf(CLIENT), clientBefore + 107);
        assertEq(token.balanceOf(CONTRACTOR), 0);
    }

    // ---- 9. paused mode still permits refunds and valid existing settlement ----
    function testSimulation9_PausedBlocksNewAcceptButAllowsRefund() external {
        uint256 id = _createJob(100, 7);
        escrow.setPaused(id, true);

        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.Paused.selector);
        escrow.accept(id);

        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.graceEnds) + 1);
        // Refund must still work while paused.
        escrow.refund(id);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Refunded));
    }

    function testSimulation9b_PausedStillSettlesAlreadyDispatchedVerdict() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        escrow.setPaused(id, true);

        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Pass, uint64(block.timestamp + 10));
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Paid));
        assertEq(token.balanceOf(CONTRACTOR), 100);
    }

    // ---- cancellation still returns exactly principal+fee ----
    function testCancelReturnsLockedAmount() external {
        uint256 id = _createJob(100, 7);
        uint256 before = token.balanceOf(CLIENT);
        vm.prank(CLIENT);
        escrow.cancel(id);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Cancelled));
        assertTrue(j.settled);
        assertEq(token.balanceOf(CLIENT), before + 107);
    }

    function testOutcomeFailDoesNotPay() external {
        uint256 id = _createJob(100, 7);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));
        WorkProofEscrow.VerdictV1 memory v =
            _matchingVerdict(id, WorkProofEscrow.Outcome.Fail, uint64(block.timestamp + 10));
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.AwaitingResubmission));
        assertFalse(j.settled);
        assertEq(token.balanceOf(CONTRACTOR), 0);
    }

    // ---- 10. accounting invariant over a fuzzed sequence ----
    function testFuzzSimulation10_AccountingInvariant(uint128 principal, uint128 fee, bool pass) external {
        principal = uint128(bound(principal, 1, 1e15));
        fee = uint128(bound(fee, 0, 1e15));
        token.mint(CLIENT, uint256(principal) + fee);

        uint256 id = _createJob(principal, fee);
        _acceptAndSubmit(id, address(0xA1), 1, keccak256("codeA"));
        _lockRandomness(id, 1, keccak256("r1"));

        uint256 clientBefore = token.balanceOf(CLIENT);
        uint256 contractorBefore = token.balanceOf(CONTRACTOR);

        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(
            id, pass ? WorkProofEscrow.Outcome.Pass : WorkProofEscrow.Outcome.Fail, uint64(block.timestamp + 10)
        );
        _settleWith(id, v, TEE_KEY);

        uint256 clientAfter = token.balanceOf(CLIENT);
        uint256 contractorAfter = token.balanceOf(CONTRACTOR);

        if (pass) {
            // Exactly principal moved to the contractor; nothing extra, nothing short.
            assertEq(contractorAfter - contractorBefore, principal);
            assertEq(clientAfter, clientBefore);
        } else {
            assertEq(contractorAfter, contractorBefore);
            assertEq(clientAfter, clientBefore);
        }
        // No outgoing principal ever exceeds what was funded: the contract
        // itself never held more than principal+fee and never pays twice
        // (settled latch), so total contractor+treasury receipts for this
        // job can never exceed principal+fee across the whole test run.
    }
}
