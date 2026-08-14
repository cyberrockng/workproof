// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {WorkProofEscrow} from "../contracts/WorkProofEscrow.sol";
import {ITeeExtensionRegistry} from "../contracts/interfaces/ITeeExtensionRegistry.sol";
import {ITeeMachineRegistry} from "../contracts/interfaces/ITeeMachineRegistry.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/// @notice Proves WorkProofEscrow against the REAL live Coston2 deployment —
/// the real FlareContractRegistry, real AssetManagerFXRP/FTestXRP, real
/// Relay/RandomNumberV2, and the real FlareTeeManager diamond
/// (0x1a9C4A0f9D76c0b1D91d22E24E573a9b377618aE, independently confirmed live
/// via `cast call` and cross-checked against the same address found
/// independently by the Ajose project). This is the fork test the corrected
/// instructions require, deliberately kept separate from the deterministic
/// vm.etch-mocked suite in test/WorkProofEscrow.t.sol.
///
/// What this test CANNOT prove, honestly: full end-to-end dispatch. Our
/// extension is not yet registered on live Coston2 (blocked on Phase 0
/// external dependencies -- GCP Confidential Space + registration), so
/// `setExtensionId()` genuinely has nothing to find, and every function that
/// calls `_getExtensionId()` (createJob, dispatchVerification) will revert
/// until that registration happens. What IS proven here: the constructor's
/// real registry resolution, that the real diamond's read-only functions
/// this contract depends on are real deployed selectors (not guessed ABI),
/// and that the extension-not-yet-found path fails the way a legitimate
/// unregistered deployment should -- not with a wrong-ABI decode crash.
contract RealRegistryForkTest is Test {
    address constant FLARE_TEE_MANAGER = 0x1a9C4A0f9D76c0b1D91d22E24E573a9b377618aE;
    address constant EXPECTED_FXRP = 0x0b6A3645c240605887a5532109323A3E12273dc7;
    address constant EXPECTED_RELAY = 0xa10B672D1c62e5457b17af63d4302add6A99d7dE;
    address constant REAL_FXRP_HOLDER = 0xC0c70925aa5156dC18b4F08235558F1f10b6dfae;
    address constant TREASURY = address(0x5452454153555259000000000000000000000000);

    WorkProofEscrow escrow;

    function setUp() external {
        vm.createSelectFork("coston2");
        escrow = new WorkProofEscrow(
            ITeeExtensionRegistry(FLARE_TEE_MANAGER), ITeeMachineRegistry(FLARE_TEE_MANAGER), TREASURY, 100
        );
    }

    function testConstructorResolvesRealFxrpRelayAndRandomNumberV2() external view {
        assertEq(address(escrow.token()), EXPECTED_FXRP, "FTestXRP resolution mismatch");
        assertEq(address(escrow.RELAY()), EXPECTED_RELAY, "Relay resolution mismatch");
        assertEq(
            address(escrow.RANDOM_NUMBER_V2()),
            EXPECTED_RELAY,
            "RandomNumberV2 resolution mismatch (same address as Relay today)"
        );
        assertEq(address(escrow.TEE_EXTENSION_REGISTRY()), FLARE_TEE_MANAGER);
        assertEq(address(escrow.TEE_MACHINE_REGISTRY()), FLARE_TEE_MANAGER);
        assertEq(escrow.treasury(), TREASURY);
    }

    function testResolvedFxrpIsARealSixDecimalToken() external view {
        IERC20 fxrp = escrow.token();
        // decimals() is not part of IERC20 but every real ERC20 exposes it;
        // low-level call to avoid pulling in the extension interface here.
        (bool ok, bytes memory data) = address(fxrp).staticcall(abi.encodeWithSignature("decimals()"));
        assertTrue(ok, "FXRP.decimals() call failed");
        assertEq(abi.decode(data, (uint8)), 6, "FXRP must be 6 decimals per SPEC.md");
    }

    function testRealFxrpHolderCanFundAClient() external {
        IERC20 fxrp = escrow.token();
        uint256 holderBalance = fxrp.balanceOf(REAL_FXRP_HOLDER);
        assertTrue(holderBalance > 1_000e6, "known real holder unexpectedly low/empty -- re-verify REAL_FXRP_HOLDER");

        address client = address(0xC11E7);
        vm.prank(REAL_FXRP_HOLDER);
        bool ok = fxrp.transfer(client, 1_000e6);
        assertTrue(ok);
        assertEq(fxrp.balanceOf(client), 1_000e6);
    }

    /// @dev getTeeMachineStatus/getTeeMachine both revert with the same
    /// distinct custom-error selector (0xceb05b68, "no machine registered at
    /// this address") for a bogus address on live Coston2 -- confirmed live
    /// via `cast call` during development and re-confirmed here on-chain,
    /// which is different from a wrong-ABI call reverting with the diamond's
    /// generic "function not found" selector. This proves the interface
    /// we extended (contracts/interfaces/ITeeMachineRegistry.sol) matches
    /// real deployed bytecode, not an invented signature.
    function testGetTeeMachineStatusIsARealSelectorOnTheLiveDiamond() external view {
        address bogus = address(0x00000000000000000000000000000000000000dd);
        (bool ok, bytes memory revertData) = FLARE_TEE_MANAGER.staticcall(
            abi.encodeWithSelector(ITeeMachineRegistry.getTeeMachineStatus.selector, bogus)
        );
        assertFalse(ok, "expected a revert for an unregistered machine address");
        // forge-lint: disable-next-line(unsafe-typecast)
        bytes4 revertSelector = bytes4(revertData);
        assertEq(
            revertSelector, bytes4(0xceb05b68), "unexpected revert selector -- interface may not match live bytecode"
        );
    }

    function testNextPublicExtensionIdIsReal() external view {
        uint256 next = ITeeExtensionRegistry(FLARE_TEE_MANAGER).nextPublicExtensionId();
        assertTrue(next >= 0x10000, "nextPublicExtensionId below the documented first-public-id floor");
    }

    /// @notice Honest negative-path proof: our extension genuinely is not
    /// registered yet, so setExtensionId() must fail exactly the way an
    /// unregistered deployment should -- with the scaffold's own
    /// "Extension ID not found." revert reason, not a generic/ABI-mismatch
    /// failure. This is the concrete, on-chain evidence backing the
    /// Phase 0 blocker in docs/operations/external-dependencies.md.
    function testSetExtensionIdFailsHonestlyWhenUnregistered() external {
        vm.expectRevert(bytes("Extension ID not found."));
        escrow.setExtensionId();
    }

    /// @notice createJob calls _getExtensionId() internally; until real
    /// registration happens, every job-creating call must fail the same
    /// honest way, not silently proceed with a fake extension id.
    function testCreateJobFailsHonestlyWhenExtensionUnregistered() external {
        vm.expectRevert(bytes("Extension ID is not set."));
        escrow.createJob(
            address(0xC0117AC706),
            address(0x1111),
            100e6,
            uint64(block.timestamp + 1_000),
            uint64(block.timestamp + 200_000),
            uint64(block.timestamp + 300_000),
            1 hours,
            keccak256("spec"),
            keccak256("bundle"),
            keccak256("engine"),
            keccak256("ciphertext")
        );
    }
}
