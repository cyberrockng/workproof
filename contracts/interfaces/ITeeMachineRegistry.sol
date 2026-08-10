// SPDX-License-Identifier: MIT
pragma solidity >=0.7.6 <0.9;

// TODO: Replace this minimal interface with the full import once flare-smart-contracts-v2
// is published as a package:
//   import { ITeeMachineRegistry } from "flare-smart-contracts-v2/contracts/userInterfaces/tee/ITeeMachineRegistry.sol";
interface ITeeMachineRegistry {
    function getRandomTeeIds(uint256 _extensionId, uint256 _count) external view returns (address[] memory);

    /// @notice Live status of a specific TEE machine (1=INITIALIZED, 2=PRODUCTION).
    /// Reverts with a distinct custom error (selector 0xceb05b68 on the live
    /// Coston2 FlareTeeManager diamond, confirmed via `cast call` against a
    /// bogus address and cross-checked against a genuinely nonexistent
    /// function's different revert selector) if no machine is registered at
    /// the given address.
    function getTeeMachineStatus(address _teeId) external view returns (uint8);
}
