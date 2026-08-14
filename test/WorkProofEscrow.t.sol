// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {WorkProofEscrow} from "../contracts/WorkProofEscrow.sol";
import {FccVerdict} from "../contracts/lib/FccVerdict.sol";
import {ITeeExtensionRegistry} from "../contracts/interfaces/ITeeExtensionRegistry.sol";
import {ITeeMachineRegistry} from "../contracts/interfaces/ITeeMachineRegistry.sol";
import {ContractRegistry} from "flare-periphery/coston2/ContractRegistry.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {
    MockToken,
    MockFlareContractRegistry,
    MockAssetManager,
    MockRelay,
    MockRandomNumberV2,
    MockTeeExtensionRegistry,
    MockTeeMachineRegistry
} from "./mocks/Mocks.sol";

/// @notice Local, deterministic Phase 3 test suite. WorkProofEscrow's
/// constructor unconditionally resolves the real hardcoded
/// FlareContractRegistry address, so a fully local (non-forked) test
/// installs a mock registry there via `vm.etch` before deploying — this is
/// the "harness/vm.etch" approach the corrected instructions call for,
/// keeping these tests fast and network-independent without weakening the
/// production contract itself (it still only ever calls the real registry
/// address; only the fork test in test/RealRegistryFork.t.sol proves that
/// real address resolves correctly on live Coston2).
contract WorkProofEscrowTest is Test {
    WorkProofEscrow escrow;
    MockToken token;
    MockAssetManager assetManager;
    MockRelay relay;
    MockRandomNumberV2 randomNumberV2;
    MockTeeExtensionRegistry mockExt;
    MockTeeMachineRegistry mockMachine;

    address constant TREASURY = address(0x5452454153555259000000000000000000000000);
    address constant CLIENT = address(0xC11E7);
    address constant CONTRACTOR = address(0xC0117AC706);

    uint256 constant TEE_KEY = 0xA11CE;
    uint256 constant WRONG_TEE_KEY = 0xBAD;
    address teeAddr;
    address wrongTeeAddr;

    address artifact;

    // Cached once in setUp(), never fetched via an inline `escrow.OP_TYPE()`
    // call inside a test body: vm.expectRevert() catches the *next* call
    // frame, and evaluating a view getter as a settleAttempt() argument
    // after expectRevert would let that staticcall consume the expectation
    // instead of the intended call (a real bug caught while writing these
    // tests -- see git history / phase2-simulation.md for the same class of
    // issue in the Phase 2 harness).
    bytes32 opType;
    bytes32 opCommand;

    bytes32 constant SPEC_HASH = keccak256("spec-v1");
    bytes32 constant BUNDLE_HASH = keccak256("bundle-v1");
    bytes32 constant ENGINE_HASH = keccak256("engine-v1");
    bytes32 constant CIPHERTEXT_HASH = keccak256("ciphertext-v1");

    uint16 constant FEE_BPS = 100; // 1%
    uint64 constant VERIFICATION_TIMEOUT = 1 hours;
    string constant RESULT_SUBMISSION_TAG = "threshold";

    function setUp() external {
        // Install a mock FlareContractRegistry at the real hardcoded address
        // the escrow's constructor calls unconditionally.
        MockFlareContractRegistry registryImpl = new MockFlareContractRegistry();
        vm.etch(ContractRegistry.FLARE_CONTRACT_REGISTRY_ADDRESS, address(registryImpl).code);
        MockFlareContractRegistry registry = MockFlareContractRegistry(ContractRegistry.FLARE_CONTRACT_REGISTRY_ADDRESS);

        token = new MockToken();
        assetManager = new MockAssetManager(IERC20(address(token)));
        relay = new MockRelay();
        randomNumberV2 = new MockRandomNumberV2();

        registry.configure("AssetManagerFXRP", address(assetManager));
        registry.configure("Relay", address(relay));
        registry.configure("RandomNumberV2", address(randomNumberV2));

        teeAddr = vm.addr(TEE_KEY);
        wrongTeeAddr = vm.addr(WRONG_TEE_KEY);
        mockExt = new MockTeeExtensionRegistry();
        mockMachine = new MockTeeMachineRegistry();
        artifact = address(mockExt);

        address[] memory ids = new address[](1);
        ids[0] = teeAddr;
        mockMachine.setNextRandomIds(ids);
        mockMachine.setStatus(teeAddr, 2); // PRODUCTION

        escrow = new WorkProofEscrow(mockExt, mockMachine, TREASURY, FEE_BPS);
        mockExt.registerSender(address(escrow));
        escrow.setExtensionId();

        token.mint(CLIENT, 1_000_000e6);
        vm.prank(CLIENT);
        token.approve(address(escrow), type(uint256).max);

        opType = escrow.OP_TYPE();
        opCommand = escrow.OP_COMMAND();
    }

    // ---------------------------------------------------------------------
    // Helpers
    // ---------------------------------------------------------------------

    // Wide submitBy/graceEnds window relative to VERIFICATION_TIMEOUT (1h):
    // a dispatch -> timeout -> resubmit cycle must comfortably fit inside
    // submitBy without tripping the *other* deadline (a real bug caught
    // while writing these tests -- a 1-hour timeout warp landed past a
    // 2000-second submitBy in an earlier, tighter version of this helper).
    function _createJob(uint128 principal) internal returns (uint256 id) {
        vm.prank(CLIENT);
        id = escrow.createJob(
            CONTRACTOR,
            principal,
            uint64(block.timestamp + 1_000),
            uint64(block.timestamp + 200_000),
            uint64(block.timestamp + 300_000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
    }

    function _accept(uint256 id) internal {
        vm.prank(CONTRACTOR);
        escrow.acceptJob(id);
    }

    function _submit(uint256 id) internal {
        vm.prank(CONTRACTOR);
        escrow.submitAttempt(id, artifact);
    }

    function _lockSecure(uint256 id) internal {
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        randomNumberV2.setSecure(j.current.targetRound, uint256(keccak256(abi.encode("secure", j.current.targetRound))));
        escrow.lockRandomness(id);
    }

    function _fullyDispatch(uint256 id) internal {
        _accept(id);
        _submit(id);
        _lockSecure(id);
        escrow.dispatchVerification(id);
    }

    function _matchingVerdict(uint256 id, WorkProofEscrow.Outcome outcome)
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
            attempt: j.current.attempt,
            instructionId: j.current.instructionId,
            specHash: j.terms.specHash,
            privateBundleHash: j.terms.privateBundleHash,
            artifactAddress: j.current.artifactAddress,
            artifactBlock: j.current.artifactBlock
        });
        v.result = WorkProofEscrow.VerdictOutcome({
            artifactCodeHash: j.current.artifactCodeHash,
            randomRound: j.current.randomRound,
            randomValueHash: j.current.randomValueHash,
            engineVersionHash: j.terms.engineVersionHash,
            outcome: outcome,
            passedCount: outcome == WorkProofEscrow.Outcome.Pass ? 5 : 3,
            executedCount: 5,
            reportHash: keccak256("report"),
            issuedAt: uint64(block.timestamp),
            expiresAt: j.terms.graceEnds
        });
    }

    function _sign(uint256 key, bytes memory data, bytes32 instructionId, uint8 status)
        internal
        view
        returns (bytes memory signature)
    {
        bytes32 digest = FccVerdict.ethSignedHash(
            FccVerdict.payloadHash(
                block.chainid, FccVerdict.actionResultHash(data, instructionId, RESULT_SUBMISSION_TAG, status)
            )
        );
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(key, digest);
        signature = abi.encodePacked(r, s, v);
    }

    function _settleWith(uint256 id, WorkProofEscrow.VerdictV1 memory v, uint256 key) internal {
        bytes memory data = abi.encode(v);
        bytes memory sig = _sign(key, data, v.id.instructionId, 1);
        escrow.settleAttempt(id, data, opType, opCommand, RESULT_SUBMISSION_TAG, 1, sig);
    }

    // =======================================================================
    // Construction / owner / fee-schedule invariants
    // =======================================================================

    function testConstructorRejectsZeroTreasury() external {
        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(mockExt, mockMachine, address(0), FEE_BPS);
    }

    function testConstructorRejectsTreasuryEqualToSelfPattern() external {
        // Treasury cannot equal the resolved token address (a real, catchable misconfiguration).
        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(mockExt, mockMachine, address(token), FEE_BPS);
    }

    function testConstructorRejectsZeroExtensionRegistry() external {
        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(ITeeExtensionRegistry(address(0)), mockMachine, TREASURY, FEE_BPS);
    }

    function testConstructorRejectsZeroMachineRegistry() external {
        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(mockExt, ITeeMachineRegistry(address(0)), TREASURY, FEE_BPS);
    }

    function testConstructorRejectsFeeAboveCap() external {
        // Computed before expectRevert: vm.expectRevert catches the *next*
        // call frame, and evaluating this as an inline constructor argument
        // would make that staticcall the "next call" instead of the
        // constructor itself.
        uint16 tooHigh = escrow.MAX_PROTOCOL_FEE_BPS() + 1;
        vm.expectRevert(WorkProofEscrow.FeeTooHigh.selector);
        new WorkProofEscrow(mockExt, mockMachine, TREASURY, tooHigh);
    }

    function testConstructorRejectsNoCodeExtensionRegistry() external {
        // A plain nonzero address with no deployed bytecode -- distinct
        // from the ZeroAddress branch, and from a real-but-misbehaving
        // registry.
        vm.expectRevert(WorkProofEscrow.NoCode.selector);
        new WorkProofEscrow(ITeeExtensionRegistry(address(0xBEEF)), mockMachine, TREASURY, FEE_BPS);
    }

    function testConstructorRejectsNoCodeMachineRegistry() external {
        vm.expectRevert(WorkProofEscrow.NoCode.selector);
        new WorkProofEscrow(mockExt, ITeeMachineRegistry(address(0xBEEF)), TREASURY, FEE_BPS);
    }

    function testConstructorRejectsTreasuryEqualToEscrowItself() external {
        // The escrow's own would-be address is deterministic (CREATE),
        // computable ahead of the constructor call.
        uint256 nonce = vm.getNonce(address(this));
        address predicted = vm.computeCreateAddress(address(this), nonce);
        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(mockExt, mockMachine, predicted, FEE_BPS);
    }

    /// @dev AssetManagerFXRP resolving to a real contract whose fAsset()
    /// validly returns address(0) -- distinct from an unconfigured
    /// registry entry, which would fail earlier with a decode error rather
    /// than reaching this specific check.
    function testConstructorRejectsZeroFxrpResolution() external {
        MockFlareContractRegistry registry = MockFlareContractRegistry(ContractRegistry.FLARE_CONTRACT_REGISTRY_ADDRESS);
        MockAssetManager brokenAssetManager = new MockAssetManager(IERC20(address(0)));
        registry.configure("AssetManagerFXRP", address(brokenAssetManager));

        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(mockExt, mockMachine, TREASURY, FEE_BPS);
    }

    function testConstructorRejectsUnresolvedRelay() external {
        MockFlareContractRegistry registry = MockFlareContractRegistry(ContractRegistry.FLARE_CONTRACT_REGISTRY_ADDRESS);
        registry.configure("Relay", address(0));

        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(mockExt, mockMachine, TREASURY, FEE_BPS);
    }

    function testConstructorRejectsUnresolvedRandomNumberV2() external {
        MockFlareContractRegistry registry = MockFlareContractRegistry(ContractRegistry.FLARE_CONTRACT_REGISTRY_ADDRESS);
        registry.configure("RandomNumberV2", address(0));

        vm.expectRevert(WorkProofEscrow.ZeroAddress.selector);
        new WorkProofEscrow(mockExt, mockMachine, TREASURY, FEE_BPS);
    }

    function testSetExtensionIdCannotBeSetTwice() external {
        // setUp() already called setExtensionId() once successfully.
        vm.expectRevert(bytes("Extension ID already set."));
        escrow.setExtensionId();
    }

    function testOnlyOwnerCanPauseOrSetFee() external {
        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.NotOwner.selector);
        escrow.pauseNewWork();

        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.NotOwner.selector);
        escrow.setProtocolFee(0);
    }

    function testSetProtocolFeeCappedAndAppliesOnlyToFutureJobs() external {
        uint256 id1 = _createJob(100e6);
        WorkProofEscrow.Job memory j1 = escrow.getJob(id1);
        assertEq(j1.terms.fee, 1e6); // 1% of 100e6

        uint16 tooHigh = escrow.MAX_PROTOCOL_FEE_BPS() + 1;
        vm.expectRevert(WorkProofEscrow.FeeTooHigh.selector);
        escrow.setProtocolFee(tooHigh);

        escrow.setProtocolFee(500); // 5%
        uint256 id2 = _createJob(100e6);
        WorkProofEscrow.Job memory j2 = escrow.getJob(id2);
        assertEq(j2.terms.fee, 5e6);

        // Existing job's fee is untouched by the schedule change.
        WorkProofEscrow.Job memory j1After = escrow.getJob(id1);
        assertEq(j1After.terms.fee, 1e6);
    }

    // =======================================================================
    // createJob validation + TEE pinning at creation
    // =======================================================================

    function testCreateJobPinsProductionTeeImmediately() external {
        uint256 id = _createJob(100e6);
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(j.terms.expectedTee, teeAddr);
    }

    function testCreateJobRejectsZeroContractor() external {
        vm.prank(CLIENT);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.createJob(
            address(0),
            100e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
    }

    function testCreateJobRejectsBadDeadlineOrdering() external {
        vm.startPrank(CLIENT);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.createJob(
            CONTRACTOR,
            100e6,
            uint64(block.timestamp),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.createJob(
            CONTRACTOR,
            100e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 500),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.createJob(
            CONTRACTOR,
            100e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 1500),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
        vm.stopPrank();
    }

    function testCreateJobRejectsZeroPrincipalOrTimeout() external {
        vm.startPrank(CLIENT);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.createJob(
            CONTRACTOR,
            0,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.createJob(
            CONTRACTOR,
            100e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            0,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
        vm.stopPrank();
    }

    function testCreateJobRejectsZeroTeeResponse() external {
        address[] memory zeroIds = new address[](1);
        zeroIds[0] = address(0);
        mockMachine.setNextRandomIds(zeroIds);

        vm.prank(CLIENT);
        vm.expectRevert(WorkProofEscrow.TeeNotProduction.selector);
        escrow.createJob(
            CONTRACTOR,
            100e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
    }

    /// @dev The registry returning a different length than the requested
    /// count (here: 2 machines for a 1-machine request) -- only reachable
    /// via the mock's forceRawResponse escape hatch, since the normal
    /// fallback always normalizes to the requested count.
    function testCreateJobRejectsMultipleTeeResponse() external {
        address[] memory twoIds = new address[](2);
        twoIds[0] = teeAddr;
        twoIds[1] = vm.addr(0xC0FFEE);
        mockMachine.setNextRandomIds(twoIds);
        mockMachine.setForceRawResponse(true);

        vm.prank(CLIENT);
        vm.expectRevert(WorkProofEscrow.TeeNotProduction.selector);
        escrow.createJob(
            CONTRACTOR,
            100e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
    }

    function testCreateJobRejectsNonProductionTee() external {
        mockMachine.setStatus(teeAddr, 1); // INITIALIZED, not PRODUCTION
        vm.prank(CLIENT);
        vm.expectRevert(WorkProofEscrow.TeeNotProduction.selector);
        escrow.createJob(
            CONTRACTOR,
            100e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
    }

    // =======================================================================
    // acceptJob / cancelUnaccepted deadline + authority
    // =======================================================================

    function testAcceptJobRejectsWrongCallerAndDeadline() external {
        uint256 id = _createJob(100e6);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.acceptJob(id); // not the contractor

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        vm.warp(uint256(j.terms.acceptBy) + 1);
        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.Deadline.selector);
        escrow.acceptJob(id);
    }

    function testCancelUnacceptedOnlyByClientBeforeAcceptance() external {
        uint256 id = _createJob(100e6);

        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.cancelUnaccepted(id); // not the client

        uint256 clientBefore = token.balanceOf(CLIENT);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.prank(CLIENT);
        escrow.cancelUnaccepted(id);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Cancelled));
        assertTrue(j.settled);
        assertEq(token.balanceOf(CLIENT), clientBefore + 100e6 + uint256(j0.terms.fee));
    }

    function testCancelUnacceptedRejectedAfterAcceptance() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        vm.prank(CLIENT);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.cancelUnaccepted(id);
    }

    // =======================================================================
    // submitAttempt: authority, deadline, artifact validity, on-chain codehash
    // =======================================================================

    function testSubmitAttemptRejectsWrongCallerOrState() external {
        uint256 id = _createJob(100e6);
        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector); // not yet accepted
        escrow.submitAttempt(id, artifact);
    }

    function testSubmitAttemptRejectsAfterSubmitByDeadline() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        vm.warp(uint256(j.terms.submitBy) + 1);
        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.Deadline.selector);
        escrow.submitAttempt(id, artifact);
    }

    function testSubmitAttemptRejectsEoaArtifact() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.submitAttempt(id, address(0xBEEF)); // no code
    }

    function testSubmitAttemptRecordsRealCodehash() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        _submit(id);
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(j.current.artifactCodeHash, artifact.codehash);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.RandomnessPending));
    }

    // =======================================================================
    // Randomness: not-ready / insecure-advances-one / secure-locks
    // =======================================================================

    function testLockRandomnessNotReadyLeavesStorageUnchanged() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        _submit(id);
        WorkProofEscrow.Job memory before = escrow.getJob(id);

        vm.expectRevert(WorkProofEscrow.RandomNotReady.selector);
        escrow.lockRandomness(id);

        WorkProofEscrow.Job memory afterCall = escrow.getJob(id);
        assertEq(afterCall.current.targetRound, before.current.targetRound);
        assertFalse(afterCall.current.randomLocked);
        assertEq(uint8(afterCall.state), uint8(WorkProofEscrow.State.RandomnessPending));
    }

    function testLockRandomnessInsecureAdvancesExactlyOneRound() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        _submit(id);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);

        randomNumberV2.setInsecure(j0.current.targetRound);
        escrow.lockRandomness(id);

        WorkProofEscrow.Job memory j1 = escrow.getJob(id);
        assertEq(j1.current.targetRound, j0.current.targetRound + 1);
        assertFalse(j1.current.randomLocked);
        assertEq(uint8(j1.state), uint8(WorkProofEscrow.State.RandomnessPending));

        // second insecure round advances by exactly one again
        randomNumberV2.setInsecure(j1.current.targetRound);
        escrow.lockRandomness(id);
        WorkProofEscrow.Job memory j2 = escrow.getJob(id);
        assertEq(j2.current.targetRound, j0.current.targetRound + 2);

        // now secure -> locks
        randomNumberV2.setSecure(j2.current.targetRound, 42);
        escrow.lockRandomness(id);
        WorkProofEscrow.Job memory j3 = escrow.getJob(id);
        assertTrue(j3.current.randomLocked);
        assertEq(j3.current.randomRound, j2.current.targetRound);
        assertEq(uint8(j3.state), uint8(WorkProofEscrow.State.ReadyToVerify));
    }

    function testLockRandomnessCannotRelockAfterSecure() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        _submit(id);
        _lockSecure(id);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.lockRandomness(id);
    }

    // =======================================================================
    // dispatchVerification: TEE re-confirmation, never re-selects
    // =======================================================================

    function testDispatchRejectsIfPinnedTeeBecomesInactive() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        _submit(id);
        _lockSecure(id);

        mockMachine.setStatus(teeAddr, 1); // demoted from PRODUCTION after creation
        vm.expectRevert(WorkProofEscrow.TeeNotProduction.selector);
        escrow.dispatchVerification(id);
    }

    function testDispatchNeverSwapsThePinnedTee() external {
        uint256 id = _createJob(100e6);
        WorkProofEscrow.Job memory jCreated = escrow.getJob(id);

        // Registry's "random" pick changes after creation; must not matter.
        address otherTee = vm.addr(0xC0FFEE);
        address[] memory otherIds = new address[](1);
        otherIds[0] = otherTee;
        mockMachine.setNextRandomIds(otherIds);
        mockMachine.setStatus(otherTee, 2);

        _accept(id);
        _submit(id);
        _lockSecure(id);
        escrow.dispatchVerification(id);

        WorkProofEscrow.Job memory jDispatched = escrow.getJob(id);
        assertEq(jDispatched.terms.expectedTee, jCreated.terms.expectedTee);
        assertEq(jDispatched.terms.expectedTee, teeAddr);
    }

    function testDispatchRejectsWrongState() external {
        uint256 id = _createJob(100e6);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.dispatchVerification(id);
    }

    function testExpireVerificationRejectsNeverDispatchedJob() external {
        uint256 id = _createJob(100e6);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.expireVerification(id);
    }

    function testSettleAttemptRejectsNeverDispatchedJob() external {
        uint256 id = _createJob(100e6);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.settleAttempt(id, hex"", opType, opCommand, RESULT_SUBMISSION_TAG, 1, hex"");
    }

    /// @dev A degenerate zero instructionId from the registry is a real,
    /// untrusted-external-input scenario (unlike settled/instructionId
    /// checks elsewhere that turned out to be provably unreachable given
    /// the state machine) -- the job proceeds to Verifying with
    /// instructionId=0, and settleAttempt's defensive check must catch it
    /// rather than ever reaching signature recovery.
    function testSettleAttemptRejectsZeroInstructionIdFromRegistry() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        _submit(id);
        _lockSecure(id);
        mockExt.setReturnZeroInstructionId(true);
        escrow.dispatchVerification(id);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(j.current.instructionId, bytes32(0));

        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.settleAttempt(id, hex"", opType, opCommand, RESULT_SUBMISSION_TAG, 1, hex"");
    }

    // =======================================================================
    // expireVerification / timeout freshness
    // =======================================================================

    function testExpireVerificationBeforeTimeoutReverts() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.expireVerification(id);
    }

    function testExpireVerificationAfterTimeoutInvalidatesInstruction() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);

        vm.warp(uint256(j0.current.timeoutAt) + 1);
        escrow.expireVerification(id);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Retryable));
        assertEq(j.current.instructionId, bytes32(0));
    }

    function testSettleRejectedAfterTimeoutEvenWithoutExplicitExpire() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.current.timeoutAt) + 1);

        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // A verificationTimeout deliberately larger than the gap to graceEnds,
    // so timeoutAt (= dispatchedAt + verificationTimeout) lands well past
    // graceEnds -- reproducing the exact scenario the default _createJob
    // helper's tight VERIFICATION_TIMEOUT (1h) never exercises: dispatch's
    // own timeout window outliving the client's refund deadline.
    function _createJobWithLateTimeout(uint128 principal) internal returns (uint256 id) {
        vm.prank(CLIENT);
        id = escrow.createJob(
            CONTRACTOR,
            principal,
            uint64(block.timestamp + 1_000),
            uint64(block.timestamp + 200_000),
            uint64(block.timestamp + 300_000), // graceEnds
            400_000, // verificationTimeout -- outlives graceEnds from dispatch
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );
    }

    /// @dev Regression test for a real bug: settleAttempt only checked
    /// timeoutAt, never graceEnds, so a Pass dispatched with a long
    /// verificationTimeout could still settle (paying the contractor) after
    /// the client's refund deadline had already passed.
    function testSettleRejectedAfterGraceEndsEvenWithinTimeoutWindow() external {
        uint256 id = _createJobWithLateTimeout(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        // Confirm the scenario is real: timeoutAt must genuinely outlive
        // graceEnds, or this test would not exercise the bug at all.
        assertGt(j0.current.timeoutAt, j0.terms.graceEnds);

        vm.warp(uint256(j0.terms.graceEnds) + 1);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        _settleWith(id, v, TEE_KEY);
    }

    /// @dev Exact boundary: settlement must still succeed AT graceEnds
    /// itself (only strictly-after is rejected), matching every other
    /// deadline check's `>` semantics in this contract.
    function testSettleSucceedsExactlyAtGraceEndsBoundary() external {
        uint256 id = _createJobWithLateTimeout(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);

        vm.warp(uint256(j0.terms.graceEnds));
        uint256 contractorBefore = token.balanceOf(CONTRACTOR);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Paid));
        assertGt(token.balanceOf(CONTRACTOR), contractorBefore);
    }

    /// @dev The actual economic race the audit flagged: once graceEnds has
    /// passed, a late Pass must never be able to pay the contractor -- the
    /// client must be able to win the race via refundExpired instead.
    function testRefundExpiredWinsRaceAfterGraceEndsNotLatePass() external {
        uint256 id = _createJobWithLateTimeout(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.terms.graceEnds) + 1);

        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        _settleWith(id, v, TEE_KEY);

        uint256 clientBefore = token.balanceOf(CLIENT);
        uint256 contractorBefore = token.balanceOf(CONTRACTOR);
        escrow.refundExpired(id);

        assertGt(token.balanceOf(CLIENT), clientBefore);
        assertEq(token.balanceOf(CONTRACTOR), contractorBefore, "contractor must not have been paid");
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Refunded));
    }

    /// @dev Exact boundary for acceptBy: at the deadline itself acceptJob
    /// must still succeed (only strictly-after is rejected). The existing
    /// testAcceptJobRejectsWrongCallerAndDeadline only proves the
    /// strictly-after side; this closes the other half of the boundary.
    function testAcceptJobSucceedsExactlyAtAcceptByBoundary() external {
        uint256 id = _createJob(100e6);
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        vm.warp(uint256(j.terms.acceptBy));
        vm.prank(CONTRACTOR);
        escrow.acceptJob(id);
        j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Accepted));
    }

    /// @dev Exact boundary for submitBy: same reasoning as acceptBy above.
    function testSubmitAttemptSucceedsExactlyAtSubmitByBoundary() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        vm.warp(uint256(j.terms.submitBy));
        vm.prank(CONTRACTOR);
        escrow.submitAttempt(id, artifact);
        j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.RandomnessPending));
    }

    function testOldInstructionCannotSettleAfterExpireAndResubmit() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory staleV = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        bytes memory staleData = abi.encode(staleV);
        bytes memory staleSig = _sign(TEE_KEY, staleData, staleV.id.instructionId, 1);

        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.current.timeoutAt) + 1);
        escrow.expireVerification(id);

        _submit(id);
        _lockSecure(id);
        escrow.dispatchVerification(id);

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settleAttempt(id, staleData, opType, opCommand, RESULT_SUBMISSION_TAG, 1, staleSig);
    }

    // =======================================================================
    // settleAttempt: PASS/FAIL/INCONCLUSIVE, full binding matrix
    // =======================================================================

    function testPassTransfersExactPrincipalAndFeeToTreasury() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);

        uint256 contractorBefore = token.balanceOf(CONTRACTOR);
        uint256 treasuryBefore = token.balanceOf(TREASURY);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Paid));
        assertTrue(j.settled);
        assertEq(token.balanceOf(CONTRACTOR) - contractorBefore, 100e6);
        assertEq(token.balanceOf(TREASURY) - treasuryBefore, 1e6);
    }

    function testOutcomeFailMovesNoBalance() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);

        uint256 before = token.balanceOf(CONTRACTOR);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Fail);
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.AwaitingResubmission));
        assertFalse(j.settled);
        assertEq(token.balanceOf(CONTRACTOR), before);
    }

    function testInconclusiveMovesNoBalance() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);

        uint256 before = token.balanceOf(CONTRACTOR);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Inconclusive);
        _settleWith(id, v, TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Retryable));
        assertEq(token.balanceOf(CONTRACTOR), before);
    }

    function testResubmissionAfterFailThenPassPaysOnce() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        _settleWith(id, _matchingVerdict(id, WorkProofEscrow.Outcome.Fail), TEE_KEY);

        _submit(id);
        _lockSecure(id);
        escrow.dispatchVerification(id);

        uint256 before = token.balanceOf(CONTRACTOR);
        _settleWith(id, _matchingVerdict(id, WorkProofEscrow.Outcome.Pass), TEE_KEY);
        assertEq(token.balanceOf(CONTRACTOR) - before, 100e6);
    }

    function testWrongSignerRejected() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, WRONG_TEE_KEY);
    }

    function testMalformedSignatureRejected() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        bytes memory data = abi.encode(_matchingVerdict(id, WorkProofEscrow.Outcome.Pass));
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settleAttempt(id, data, opType, opCommand, RESULT_SUBMISSION_TAG, 1, hex"1234");
    }

    function testMalformedAbiDataReverts() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        bytes memory garbage = hex"deadbeef";
        vm.expectRevert();
        escrow.settleAttempt(id, garbage, opType, opCommand, RESULT_SUBMISSION_TAG, 1, hex"00");
    }

    function testWrongOpTypeOrCommandRejected() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        bytes memory data = abi.encode(v);
        bytes memory sig = _sign(TEE_KEY, data, v.id.instructionId, 1);

        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settleAttempt(id, data, bytes32("WRONG"), opCommand, RESULT_SUBMISSION_TAG, 1, sig);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settleAttempt(id, data, opType, bytes32("WRONG"), RESULT_SUBMISSION_TAG, 1, sig);
    }

    function testWrongStatusRejected() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        bytes memory data = abi.encode(v);
        bytes memory sig = _sign(TEE_KEY, data, v.id.instructionId, 0);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settleAttempt(id, data, opType, opCommand, RESULT_SUBMISSION_TAG, 0, sig);
    }

    function testWrongSubmissionTagRejected() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        bytes memory data = abi.encode(v);
        bytes memory sig = _sign(TEE_KEY, data, v.id.instructionId, 1);
        // signature was made over "threshold"; passing "submit" both fails the tag
        // check AND (independently) would fail signature recovery.
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        escrow.settleAttempt(id, data, opType, opCommand, "submit", 1, sig);
    }

    /// @dev After any outcome (PASS/FAIL/INCONCLUSIVE) settles once, job
    /// state moves away from Verifying, so a replay is actually caught by
    /// the leading state guard (InvalidState), not the verdict-binding
    /// check -- state-machine gating alone already fully prevents replay
    /// here; the `settled` bool is a secondary, currently-redundant defense
    /// for this exact path. Confirmed by first running this test with the
    /// wrong expectation (InvalidVerdict) and observing the real revert.
    function testReplayRejected() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        bytes memory data = abi.encode(v);
        bytes memory sig = _sign(TEE_KEY, data, v.id.instructionId, 1);
        escrow.settleAttempt(id, data, opType, opCommand, RESULT_SUBMISSION_TAG, 1, sig);

        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.settleAttempt(id, data, opType, opCommand, RESULT_SUBMISSION_TAG, 1, sig);
    }

    function testMutationRejected_specHash() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.specHash = keccak256("tampered");
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_bundleHash() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.privateBundleHash = keccak256("tampered");
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_artifactAddress() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.artifactAddress = address(0xBAD1);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_artifactBlock() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.artifactBlock = v.id.artifactBlock + 1;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_artifactCodeHash() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.artifactCodeHash = keccak256("tampered");
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_randomRound() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.randomRound = v.result.randomRound + 1;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_randomValueHash() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.randomValueHash = keccak256("tampered");
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_engineVersionHash() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.engineVersionHash = keccak256("tampered");
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_reportHashZero() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.reportHash = bytes32(0);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_zeroExecutedCount() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.passedCount = 0;
        v.result.executedCount = 0;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_passedCountAboveExecutedCount() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Fail);
        v.result.passedCount = 6;
        v.result.executedCount = 5;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_passOutcomeWithPartialCounts() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.passedCount = 4;
        v.result.executedCount = 5;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_nonPassOutcomeWithAllPassedCounts() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Fail);
        v.result.passedCount = 5;
        v.result.executedCount = 5;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);

        // Repeat on a fresh job because the first revert leaves the original
        // job in Verifying but this keeps the outcome branch under test
        // unambiguous.
        uint256 id2 = _createJob(100e6);
        _fullyDispatch(id2);
        v = _matchingVerdict(id2, WorkProofEscrow.Outcome.Inconclusive);
        v.result.passedCount = 5;
        v.result.executedCount = 5;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id2, v, TEE_KEY);
    }

    function testMutationRejected_expiresAtNotEqualGraceEnds() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.expiresAt = v.result.expiresAt + 1; // TEE claiming a later expiry than the job's real grace deadline
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_issuedAtInFuture() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.issuedAt = uint64(block.timestamp) + 1000;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    /// @dev Foundry's local (non-forked) block.timestamp starts at exactly
    /// 1 (confirmed via console.log while writing this test), so a
    /// hardcoded `issuedAt = 1` was not actually before dispatchedAt (also
    /// ~1) -- warp forward first so "before dispatch" is unambiguous
    /// regardless of the harness's starting clock.
    function testMutationRejected_issuedAtBeforeDispatch() external {
        vm.warp(1_000_000);
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.Job memory jd = escrow.getJob(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.result.issuedAt = jd.current.dispatchedAt - 1;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_jobId() external {
        uint256 id = _createJob(100e6);
        uint256 otherId = _createJob(50e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.jobId = otherId;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_attempt() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.attempt = v.id.attempt + 1;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_instructionId() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.instructionId = keccak256("forged");
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_chainId() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.chainId = 999;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_escrowAddress() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.escrowAddress = address(0x1234);
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    function testMutationRejected_schemaVersion() external {
        uint256 id = _createJob(100e6);
        _fullyDispatch(id);
        WorkProofEscrow.VerdictV1 memory v = _matchingVerdict(id, WorkProofEscrow.Outcome.Pass);
        v.id.schemaVersion = 2;
        vm.expectRevert(WorkProofEscrow.InvalidVerdict.selector);
        _settleWith(id, v, TEE_KEY);
    }

    // =======================================================================
    // refundExpired
    // =======================================================================

    function testRefundExpiredBeforeGraceEndsReverts() external {
        uint256 id = _createJob(100e6);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.refundExpired(id);
    }

    function testRefundExpiredReturnsPrincipalAndFeeOnlyToClient() external {
        uint256 id = _createJob(100e6);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.terms.graceEnds) + 1);

        uint256 clientBefore = token.balanceOf(CLIENT);
        escrow.refundExpired(id);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Refunded));
        assertTrue(j.settled);
        assertEq(token.balanceOf(CLIENT), clientBefore + 100e6 + uint256(j0.terms.fee));
        assertEq(token.balanceOf(CONTRACTOR), 0);
    }

    function testCannotRefundTwice() external {
        uint256 id = _createJob(100e6);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        vm.warp(uint256(j0.terms.graceEnds) + 1);
        escrow.refundExpired(id);
        vm.expectRevert(WorkProofEscrow.InvalidState.selector);
        escrow.refundExpired(id);
    }

    // =======================================================================
    // Pause liveness: blocks only new jobs/attempts, never existing progress
    // =======================================================================

    function testPauseBlocksCreateJobAndSubmitAttempt() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        escrow.pauseNewWork();

        vm.prank(CLIENT);
        vm.expectRevert(WorkProofEscrow.IsPaused.selector);
        escrow.createJob(
            CONTRACTOR,
            1e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );

        vm.prank(CONTRACTOR);
        vm.expectRevert(WorkProofEscrow.IsPaused.selector);
        escrow.submitAttempt(id, artifact);
    }

    function testPauseDoesNotBlockAcceptLockDispatchSettleExpireCancelRefund() external {
        uint256 id = _createJob(100e6);
        uint256 cancelableId = _createJob(100e6);
        escrow.pauseNewWork();

        // accept still works
        vm.prank(CONTRACTOR);
        escrow.acceptJob(id);

        escrow.unpauseNewWork();
        _submit(id);
        escrow.pauseNewWork();

        // lockRandomness still works while paused
        _lockSecure(id);
        // dispatchVerification still works while paused
        escrow.dispatchVerification(id);
        // settleAttempt still works while paused
        uint256 before = token.balanceOf(CONTRACTOR);
        _settleWith(id, _matchingVerdict(id, WorkProofEscrow.Outcome.Pass), TEE_KEY);
        assertEq(token.balanceOf(CONTRACTOR) - before, 100e6);

        // cancelUnaccepted still works while paused
        vm.prank(CLIENT);
        escrow.cancelUnaccepted(cancelableId);
        WorkProofEscrow.Job memory jc = escrow.getJob(cancelableId);
        assertEq(uint8(jc.state), uint8(WorkProofEscrow.State.Cancelled));
    }

    function testPauseDoesNotBlockExpireOrRefund() external {
        // Both jobs must exist before pausing -- createJob is one of the two
        // functions pause is *supposed* to block (a real bug caught while
        // writing this test: creating id2 after pauseNewWork() correctly
        // reverted with IsPaused, which was never the property under test).
        uint256 id = _createJob(100e6);
        uint256 id2 = _createJob(50e6);
        _fullyDispatch(id);
        WorkProofEscrow.Job memory j0 = escrow.getJob(id);
        WorkProofEscrow.Job memory j2t = escrow.getJob(id2);
        escrow.pauseNewWork();

        vm.warp(uint256(j0.current.timeoutAt) + 1);
        escrow.expireVerification(id); // still works while paused
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Retryable));

        vm.warp(uint256(j2t.terms.graceEnds) + 1);
        escrow.refundExpired(id2); // still works while paused
        WorkProofEscrow.Job memory j2 = escrow.getJob(id2);
        assertEq(uint8(j2.state), uint8(WorkProofEscrow.State.Refunded));
    }

    // =======================================================================
    // Reentrancy hardening
    // =======================================================================

    /// @dev The reentrant call happens *inside* transferFrom, i.e. inside the
    /// first createJob's execution. nonReentrant makes that inner call
    /// revert; our mock's low-level `.call()` swallows that failure rather
    /// than propagating it, so the OUTER createJob still succeeds exactly
    /// once. The test asserts on that: exactly one job created, exactly one
    /// principal+fee debit -- not two.
    function testMaliciousTokenCannotReenterCreateJob() external {
        token.armReentrancy(
            address(escrow),
            abi.encodeWithSelector(
                WorkProofEscrow.createJob.selector,
                CONTRACTOR,
                uint128(1e6),
                uint64(block.timestamp + 1000),
                uint64(block.timestamp + 2000),
                uint64(block.timestamp + 3000),
                VERIFICATION_TIMEOUT,
                SPEC_HASH,
                BUNDLE_HASH,
                ENGINE_HASH,
                CIPHERTEXT_HASH
            )
        );

        uint256 jobsBefore = escrow.nextJobId();
        uint256 clientBefore = token.balanceOf(CLIENT);

        vm.prank(CLIENT);
        uint256 id = escrow.createJob(
            CONTRACTOR,
            1e6,
            uint64(block.timestamp + 1000),
            uint64(block.timestamp + 2000),
            uint64(block.timestamp + 3000),
            VERIFICATION_TIMEOUT,
            SPEC_HASH,
            BUNDLE_HASH,
            ENGINE_HASH,
            CIPHERTEXT_HASH
        );

        assertEq(id, jobsBefore);
        assertEq(escrow.nextJobId(), jobsBefore + 1); // exactly one job, not two
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(clientBefore - token.balanceOf(CLIENT), uint256(j.terms.principal) + j.terms.fee);
    }

    function testMaliciousExtensionRegistryCannotReenterDispatch() external {
        uint256 id = _createJob(100e6);
        _accept(id);
        _submit(id);
        _lockSecure(id);

        mockExt.armReentrancy(
            address(escrow), abi.encodeWithSelector(WorkProofEscrow.dispatchVerification.selector, id)
        );

        // The reentrant inner call reverts (nonReentrant); our mock swallows
        // that failure and returns normally, so the OUTER dispatch still
        // succeeds exactly once -- assert it only ever reaches Verifying once
        // with a single instructionId, never double-dispatched.
        escrow.dispatchVerification(id);
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(WorkProofEscrow.State.Verifying));
    }

    // =======================================================================
    // Fee rounding (6-decimal FTestXRP)
    // =======================================================================

    function testFuzzFeeRoundingNeverExceedsPrincipalPlusExactFee(uint128 principal, uint16 bps) external {
        principal = uint128(bound(uint256(principal), 1, 1_000_000e6));
        bps = uint16(bound(uint256(bps), 0, escrow.MAX_PROTOCOL_FEE_BPS()));
        escrow.setProtocolFee(bps);
        token.mint(CLIENT, uint256(principal));

        uint256 expectedFee = (uint256(principal) * bps) / 10_000;
        uint256 clientBefore = token.balanceOf(CLIENT);
        uint256 id = _createJob(principal);
        WorkProofEscrow.Job memory j = escrow.getJob(id);

        assertEq(j.terms.fee, expectedFee);
        assertEq(clientBefore - token.balanceOf(CLIENT), uint256(principal) + expectedFee);
        assertEq(token.balanceOf(address(escrow)), uint256(principal) + expectedFee);
    }

    // =======================================================================
    // Solvency / single-payout / correct-recipient invariant, fuzzed
    // =======================================================================

    function testFuzzSolvencyAndSinglePayout(uint128 principal, uint8 outcomeSeed) external {
        principal = uint128(bound(uint256(principal), 1e6, 100_000e6));
        token.mint(CLIENT, uint256(principal));

        uint256 id = _createJob(principal);
        _fullyDispatch(id);

        uint256 escrowBefore = token.balanceOf(address(escrow));
        uint256 clientBefore = token.balanceOf(CLIENT);
        uint256 contractorBefore = token.balanceOf(CONTRACTOR);
        uint256 treasuryBefore = token.balanceOf(TREASURY);

        WorkProofEscrow.Outcome outcome = WorkProofEscrow.Outcome(outcomeSeed % 3);
        _settleWith(id, _matchingVerdict(id, outcome), TEE_KEY);

        WorkProofEscrow.Job memory j = escrow.getJob(id);
        uint256 escrowAfter = token.balanceOf(address(escrow));
        uint256 contractorAfter = token.balanceOf(CONTRACTOR);
        uint256 treasuryAfter = token.balanceOf(TREASURY);

        if (outcome == WorkProofEscrow.Outcome.Pass) {
            assertEq(contractorAfter - contractorBefore, principal);
            assertEq(treasuryAfter - treasuryBefore, uint256(j.terms.fee));
            assertEq(escrowBefore - escrowAfter, uint256(principal) + j.terms.fee);
            assertTrue(j.settled);

            // Single payout: settling again must be impossible (already not
            // Verifying), and total paid never exceeds what was funded.
            assertTrue(uint256(principal) + j.terms.fee <= escrowBefore);
        } else {
            assertEq(contractorAfter, contractorBefore);
            assertEq(treasuryAfter, treasuryBefore);
            assertEq(escrowAfter, escrowBefore);
            assertEq(token.balanceOf(CLIENT), clientBefore);
            assertFalse(j.settled);
        }
    }
}
