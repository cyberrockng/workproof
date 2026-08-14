// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {WorkProofEscrow} from "../contracts/WorkProofEscrow.sol";
import {FccVerdict} from "../contracts/lib/FccVerdict.sol";
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

/// @notice Stateful handler driving WorkProofEscrow through arbitrary
/// interleaved sequences across MANY jobs at once (create/accept/submit/
/// lock/dispatch/settle/refund/cancel, in any order, on any job), which is
/// exactly what the existing single-job `testFuzzSolvencyAndSinglePayout`
/// (test/WorkProofEscrow.t.sol) structurally cannot reach -- that test
/// fuzzes outcome/principal for ONE job per run, never cross-job
/// interleaving. Every state-changing call is wrapped in `try/catch` so an
/// expected revert (wrong state, deadline passed, etc.) just skips that
/// call rather than aborting the whole fuzz run -- the fuzzer explores
/// whatever sequences the real contract actually allows.
contract WorkProofInvariantHandler is Test {
    WorkProofEscrow public immutable escrow;
    MockToken public immutable token;
    MockRandomNumberV2 public immutable randomNumberV2;
    address public immutable treasury;
    address public immutable artifact;
    uint256 public immutable teeKey;
    address public immutable client;
    address public immutable contractor;

    uint256[] public jobIds;

    // Ghost accounting: an independent ledger of every token movement this
    // handler has actually observed, compared against the escrow's real
    // balance by invariant_solvency below -- not re-reading the same
    // storage the contract already trusts.
    uint256 public ghost_funded;
    uint256 public ghost_paidToContractor;
    uint256 public ghost_feesToTreasury;
    uint256 public ghost_refundedToClient;

    mapping(uint256 => bool) public everSettledPaid;
    mapping(uint256 => bool) public everSettledRefunded;
    string constant RESULT_SUBMISSION_TAG = "threshold";

    constructor(
        WorkProofEscrow _escrow,
        MockToken _token,
        MockRandomNumberV2 _randomNumberV2,
        address _treasury,
        address _artifact,
        uint256 _teeKey,
        address _client,
        address _contractor
    ) {
        escrow = _escrow;
        token = _token;
        randomNumberV2 = _randomNumberV2;
        treasury = _treasury;
        artifact = _artifact;
        teeKey = _teeKey;
        client = _client;
        contractor = _contractor;
    }

    function jobCount() external view returns (uint256) {
        return jobIds.length;
    }

    function warp(uint256 seed) public {
        vm.warp(block.timestamp + bound(seed, 1, 250_000));
    }

    function createJob(uint128 principalSeed) public {
        uint128 principal = uint128(bound(principalSeed, 1e6, 1_000_000e6));
        token.mint(client, principal);
        vm.prank(client);
        token.approve(address(escrow), type(uint256).max);

        vm.prank(client);
        try escrow.createJob(
            contractor,
            principal,
            uint64(block.timestamp + 1_000),
            uint64(block.timestamp + 200_000),
            uint64(block.timestamp + 300_000),
            1 hours,
            keccak256(abi.encode("spec", jobIds.length)),
            keccak256(abi.encode("bundle", jobIds.length)),
            keccak256(abi.encode("engine", jobIds.length)),
            keccak256(abi.encode("cipher", jobIds.length))
        ) returns (
            uint256 id
        ) {
            jobIds.push(id);
            WorkProofEscrow.Job memory j = escrow.getJob(id);
            ghost_funded += uint256(j.terms.principal) + j.terms.fee;
        } catch {}
    }

    function acceptJob(uint256 idx) public {
        if (jobIds.length == 0) return;
        vm.prank(contractor);
        try escrow.acceptJob(jobIds[idx % jobIds.length]) {} catch {}
    }

    function submitAttempt(uint256 idx) public {
        if (jobIds.length == 0) return;
        vm.prank(contractor);
        try escrow.submitAttempt(jobIds[idx % jobIds.length], artifact) {} catch {}
    }

    function lockRandomness(uint256 idx) public {
        if (jobIds.length == 0) return;
        uint256 id = jobIds[idx % jobIds.length];
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        randomNumberV2.setSecure(
            j.current.targetRound, uint256(keccak256(abi.encode("secure", id, j.current.targetRound)))
        );
        try escrow.lockRandomness(id) {} catch {}
    }

    function dispatchVerification(uint256 idx) public {
        if (jobIds.length == 0) return;
        try escrow.dispatchVerification(jobIds[idx % jobIds.length]) {} catch {}
    }

    function settleAttempt(uint256 idx, uint8 outcomeSeed) public {
        if (jobIds.length == 0) return;
        uint256 id = jobIds[idx % jobIds.length];
        WorkProofEscrow.Job memory j = escrow.getJob(id);
        if (j.state != WorkProofEscrow.State.Verifying) return;

        WorkProofEscrow.Outcome outcome = WorkProofEscrow.Outcome(outcomeSeed % 3);
        WorkProofEscrow.VerdictV1 memory v;
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

        bytes memory data = abi.encode(v);
        bytes32 digest = FccVerdict.ethSignedHash(
            FccVerdict.payloadHash(
                block.chainid, FccVerdict.actionResultHash(data, v.id.instructionId, RESULT_SUBMISSION_TAG, 1)
            )
        );
        (uint8 sv, bytes32 sr, bytes32 ss) = vm.sign(teeKey, digest);
        bytes memory sig = abi.encodePacked(sr, ss, sv);

        uint256 contractorBefore = token.balanceOf(contractor);
        uint256 treasuryBefore = token.balanceOf(treasury);

        try escrow.settleAttempt(id, data, escrow.OP_TYPE(), escrow.OP_COMMAND(), RESULT_SUBMISSION_TAG, 1, sig) {
            if (outcome == WorkProofEscrow.Outcome.Pass) {
                everSettledPaid[id] = true;
                ghost_paidToContractor += token.balanceOf(contractor) - contractorBefore;
                ghost_feesToTreasury += token.balanceOf(treasury) - treasuryBefore;
            }
        } catch {}
    }

    function refundExpired(uint256 idx) public {
        if (jobIds.length == 0) return;
        uint256 id = jobIds[idx % jobIds.length];
        uint256 clientBefore = token.balanceOf(client);
        try escrow.refundExpired(id) {
            everSettledRefunded[id] = true;
            ghost_refundedToClient += token.balanceOf(client) - clientBefore;
        } catch {}
    }

    function cancelUnaccepted(uint256 idx) public {
        if (jobIds.length == 0) return;
        uint256 id = jobIds[idx % jobIds.length];
        uint256 clientBefore = token.balanceOf(client);
        vm.prank(client);
        try escrow.cancelUnaccepted(id) {
            everSettledRefunded[id] = true;
            ghost_refundedToClient += token.balanceOf(client) - clientBefore;
        } catch {}
    }
}

contract WorkProofEscrowInvariantTest is Test {
    WorkProofEscrow escrow;
    MockToken token;
    MockRandomNumberV2 randomNumberV2;
    MockTeeExtensionRegistry mockExt;
    WorkProofInvariantHandler handler;

    address constant TREASURY = address(0x5452454153555259000000000000000000000000);
    address constant CLIENT = address(0xC11E7);
    address constant CONTRACTOR = address(0xC0117AC706);
    uint256 constant TEE_KEY = 0xA11CE;
    uint16 constant FEE_BPS = 100;

    function setUp() external {
        MockFlareContractRegistry registryImpl = new MockFlareContractRegistry();
        vm.etch(ContractRegistry.FLARE_CONTRACT_REGISTRY_ADDRESS, address(registryImpl).code);
        MockFlareContractRegistry registry = MockFlareContractRegistry(ContractRegistry.FLARE_CONTRACT_REGISTRY_ADDRESS);

        token = new MockToken();
        MockAssetManager assetManager = new MockAssetManager(IERC20(address(token)));
        MockRelay relay = new MockRelay();
        randomNumberV2 = new MockRandomNumberV2();

        registry.configure("AssetManagerFXRP", address(assetManager));
        registry.configure("Relay", address(relay));
        registry.configure("RandomNumberV2", address(randomNumberV2));

        address teeAddr = vm.addr(TEE_KEY);
        mockExt = new MockTeeExtensionRegistry();
        MockTeeMachineRegistry mockMachine = new MockTeeMachineRegistry();

        address[] memory ids = new address[](1);
        ids[0] = teeAddr;
        mockMachine.setNextRandomIds(ids);
        mockMachine.setStatus(teeAddr, 2); // PRODUCTION

        escrow = new WorkProofEscrow(mockExt, mockMachine, TREASURY, FEE_BPS);
        mockExt.registerSender(address(escrow));
        escrow.setExtensionId();

        handler = new WorkProofInvariantHandler(
            escrow, token, randomNumberV2, TREASURY, address(mockExt), TEE_KEY, CLIENT, CONTRACTOR
        );

        targetContract(address(handler));
    }

    /// @dev Conservation of funds: whatever the escrow currently holds must
    /// equal everything ever funded into it, minus everything the handler
    /// has actually observed leaving it. If any code path ever moved funds
    /// without going through the tracked paths (or moved more/less than it
    /// should), this diverges.
    function invariant_solvency() external view {
        assertEq(
            token.balanceOf(address(escrow)),
            handler.ghost_funded() - handler.ghost_paidToContractor() - handler.ghost_feesToTreasury()
                - handler.ghost_refundedToClient(),
            "escrow balance must equal funded minus paid minus refunded"
        );
    }

    /// @dev No job the handler has ever seen pay out can also have ever
    /// been refunded (or vice versa) -- a single job settling both ways
    /// would mean the contractor and the client were both paid for the same
    /// unit of work, exactly the double-spend WorkProofEscrow's `settled`
    /// flag and state machine exist to prevent.
    function invariant_noJobEverPaysAndRefunds() external view {
        uint256 n = handler.jobCount();
        for (uint256 i = 0; i < n; i++) {
            uint256 id = handler.jobIds(i);
            bool paid = handler.everSettledPaid(id);
            bool refunded = handler.everSettledRefunded(id);
            assertFalse(paid && refunded, "a job must never be both paid and refunded");
        }
    }

    /// @dev Once the handler has observed a job pay out or refund, the
    /// contract's own `settled` flag for that job must independently agree
    /// -- catching any drift between what actually happened (real balance
    /// deltas the handler measured) and what the contract's own bookkeeping
    /// claims happened.
    function invariant_settledFlagMatchesObservedOutcome() external view {
        uint256 n = handler.jobCount();
        for (uint256 i = 0; i < n; i++) {
            uint256 id = handler.jobIds(i);
            if (handler.everSettledPaid(id) || handler.everSettledRefunded(id)) {
                WorkProofEscrow.Job memory j = escrow.getJob(id);
                assertTrue(j.settled, "contract must also consider this job settled");
            }
        }
    }
}
