// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {ITeeExtensionRegistry} from "../../contracts/interfaces/ITeeExtensionRegistry.sol";
import {ITeeMachineRegistry} from "../../contracts/interfaces/ITeeMachineRegistry.sol";

/// @notice Local, deterministic test doubles for WorkProofEscrow's external
/// dependencies. None of these are `is I...` — Solidity dispatch only needs
/// matching function selectors, and inheriting the real (huge, IAssetManager
/// especially) interfaces would force stubbing dozens of unrelated
/// functions. Each mock intentionally exposes ONLY the selectors the real
/// interface defines that the escrow actually calls, with the real
/// signatures copied verbatim from the pinned flare-foundry-periphery-package
/// (never invented) so a selector-level mismatch would be caught immediately
/// by the constructor reverting.
///
/// A separate, non-mocked fork test (test/RealRegistryFork.t.sol) proves the
/// real Coston2 addresses satisfy these same real signatures.

/// @dev 6-decimal ERC20 standing in for FTestXRP locally. `maliciousMode`
/// turns `transfer`/`transferFrom` into a reentrancy probe for the
/// dispatch/settle reentrancy tests.
contract MockToken {
    string public constant name = "Mock FTestXRP";
    string public constant symbol = "mFXRP";
    uint8 public constant decimals = 6;

    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    bool public maliciousMode;
    address public reentrantTarget;
    bytes public reentrantCalldata;
    bool private _reentered;

    function mint(address to, uint256 amount) external {
        balanceOf[to] += amount;
    }

    function approve(address spender, uint256 amount) external returns (bool) {
        allowance[msg.sender][spender] = amount;
        return true;
    }

    function armReentrancy(address target, bytes calldata data) external {
        maliciousMode = true;
        reentrantTarget = target;
        reentrantCalldata = data;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        uint256 allowed = allowance[from][msg.sender];
        if (allowed != type(uint256).max) {
            require(allowed >= amount, "allowance");
            allowance[from][msg.sender] = allowed - amount;
        }
        require(balanceOf[from] >= amount, "balance");
        balanceOf[from] -= amount;
        balanceOf[to] += amount;
        _maybeReenter();
        return true;
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        require(balanceOf[msg.sender] >= amount, "balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        _maybeReenter();
        return true;
    }

    function _maybeReenter() private {
        if (!maliciousMode || _reentered || reentrantTarget == address(0)) return;
        _reentered = true;
        (bool ok,) = reentrantTarget.call(reentrantCalldata);
        ok; // the outer call's own revert (nonReentrant) is what the test asserts on
        _reentered = false;
    }
}

/// @dev Matches IFlareContractRegistry.getContractAddressByHash /
/// getContractAddressByName exactly (see
/// lib/flare-foundry-periphery-package/src/coston2/IFlareContractRegistry.sol).
/// `configure` is test-only setup, not part of the real interface.
contract MockFlareContractRegistry {
    mapping(bytes32 => address) private _byHash;

    function configure(string memory contractName, address addr) external {
        _byHash[keccak256(abi.encode(contractName))] = addr;
    }

    function getContractAddressByName(string calldata contractName) external view returns (address) {
        return _byHash[keccak256(abi.encode(contractName))];
    }

    function getContractAddressByHash(bytes32 nameHash) external view returns (address) {
        return _byHash[nameHash];
    }
}

/// @dev Matches IAssetManager.fAsset() (see
/// lib/flare-foundry-periphery-package/src/coston2/IAssetManager.sol#67).
contract MockAssetManager {
    IERC20 public immutable fAssetToken;

    constructor(IERC20 token_) {
        fAssetToken = token_;
    }

    function fAsset() external view returns (IERC20) {
        return fAssetToken;
    }
}

/// @dev Matches IRelay.getVotingRoundId (RandomNumberV2Interface.
/// getRandomNumberHistorical is served by MockRandomNumberV2 below — on live
/// Coston2 both resolve to the same address, but the escrow resolves them
/// via two separate registry entries, so the mocks mirror that split rather
/// than collapsing it).
contract MockRelay {
    uint256 public currentRoundId = 1_000_000;

    function setCurrentRoundId(uint256 id) external {
        currentRoundId = id;
    }

    function getVotingRoundId(uint256) external view returns (uint256) {
        return currentRoundId;
    }
}

/// @dev Matches RandomNumberV2Interface.getRandomNumberHistorical exactly
/// (see lib/flare-foundry-periphery-package/src/coston2/RandomNumberV2Interface.sol).
/// Per-round behavior is fully controllable so "not ready" / "insecure" /
/// "secure" can each be exercised deterministically, which a live fork
/// cannot offer (the fork is frozen at one historical state).
contract MockRandomNumberV2 {
    enum RoundBehavior {
        NotReady,
        Insecure,
        Secure
    }

    mapping(uint256 => RoundBehavior) public behaviorOf;
    mapping(uint256 => uint256) public valueOf;

    function setNotReady(uint256 round) external {
        behaviorOf[round] = RoundBehavior.NotReady;
    }

    function setInsecure(uint256 round) external {
        behaviorOf[round] = RoundBehavior.Insecure;
        valueOf[round] = uint256(keccak256(abi.encode("insecure", round)));
    }

    function setSecure(uint256 round, uint256 value) external {
        behaviorOf[round] = RoundBehavior.Secure;
        valueOf[round] = value;
    }

    function getRandomNumberHistorical(uint256 votingRoundId)
        external
        view
        returns (uint256 _randomNumber, bool _isSecureRandom, uint256 _randomTimestamp)
    {
        RoundBehavior b = behaviorOf[votingRoundId];
        if (b == RoundBehavior.NotReady) revert("round not finalized");
        return (valueOf[votingRoundId], b == RoundBehavior.Secure, block.timestamp);
    }
}

/// @dev Matches ITeeExtensionRegistry exactly (contracts/interfaces/ITeeExtensionRegistry.sol,
/// itself copied from the real scaffold). `registerSender`/`armReentrancy`
/// are test-only, not part of the real interface.
contract MockTeeExtensionRegistry is ITeeExtensionRegistry {
    mapping(uint256 => address) public senderOf;
    uint256 public nextId = 0x10000;
    uint256 public instructionNonce;

    bool public reentrantMode;
    address public reentrantTarget;
    bytes public reentrantCalldata;
    bool public returnZeroInstructionId;

    function registerSender(address sender) external returns (uint256 id) {
        id = nextId++;
        senderOf[id] = sender;
    }

    function armReentrancy(address target, bytes calldata data) external {
        reentrantMode = true;
        reentrantTarget = target;
        reentrantCalldata = data;
    }

    /// @dev Simulates an untrusted registry returning a degenerate
    /// instructionId, exercising WorkProofEscrow's defensive
    /// `instructionId == 0` rejection in settleAttempt.
    function setReturnZeroInstructionId(bool value) external {
        returnZeroInstructionId = value;
    }

    function sendInstructions(address[] calldata, TeeInstructionParams calldata)
        external
        payable
        returns (bytes32 instructionId)
    {
        if (reentrantMode) {
            reentrantMode = false; // one-shot, avoid infinite recursion in the probe itself
            (bool ok,) = reentrantTarget.call(reentrantCalldata);
            ok; // the outer call's own revert (nonReentrant) is what the test asserts on
        }
        if (returnZeroInstructionId) return bytes32(0);
        instructionNonce++;
        instructionId = keccak256(abi.encode("mock-instruction", instructionNonce, block.timestamp));
    }

    function nextPublicExtensionId() external view returns (uint256) {
        return nextId;
    }

    function getTeeExtensionInstructionsSender(uint256 id) external view returns (address) {
        return senderOf[id];
    }
}

/// @dev Matches ITeeMachineRegistry exactly, including the live-verified
/// getTeeMachineStatus addition (contracts/interfaces/ITeeMachineRegistry.sol).
/// Status/response behavior per address is fully controllable to exercise
/// zero/multiple/unexpected/stale/inactive responses.
contract MockTeeMachineRegistry is ITeeMachineRegistry {
    mapping(address => uint8) public statusOf;
    address[] public nextRandomIds;
    bool public forceRawResponse;

    function setStatus(address teeId, uint8 status) external {
        statusOf[teeId] = status;
    }

    function setNextRandomIds(address[] calldata ids) external {
        delete nextRandomIds;
        for (uint256 i = 0; i < ids.length; i++) {
            nextRandomIds.push(ids[i]);
        }
    }

    /// @dev When set, getRandomTeeIds returns `nextRandomIds` verbatim
    /// regardless of the requested `count` -- the only way to exercise a
    /// registry returning an unexpected (wrong-length, e.g. "multiple")
    /// response, since the normal fallback below always normalizes to the
    /// requested count.
    function setForceRawResponse(bool value) external {
        forceRawResponse = value;
    }

    function getRandomTeeIds(uint256, uint256 count) external view returns (address[] memory ids) {
        if (forceRawResponse || nextRandomIds.length == count) return nextRandomIds;
        // Default: return `count` copies of the single configured id, or
        // zero addresses if none configured (exercises the zero-response path).
        ids = new address[](count);
        address single = nextRandomIds.length > 0 ? nextRandomIds[0] : address(0);
        for (uint256 i = 0; i < count; i++) {
            ids[i] = single;
        }
    }

    function getTeeMachineStatus(address teeId) external view returns (uint8) {
        return statusOf[teeId];
    }
}
