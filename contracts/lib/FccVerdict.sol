// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @notice Real Flare Confidential Compute `ActionResult` signing chain,
/// reconstructed from tee-node v0.0.24 (pkg/types/actions.go Hash(),
/// internal/router/utils.go SignResult) and go-flare-common
/// v1.2.2-...-09a10067e6a4 (pkg/signing/hash.go Payload.Hash()). Byte-exact
/// equivalence proven against a Go-signed vector in
/// docs/evidence/fcc-signature-spike-v1.json (test/FccSignatureSpike.t.sol).
///
/// Only result.data, result.id, result.submissionTag, and result.status are
/// covered by the official ActionResult.Hash() formula — opType/opCommand/
/// version/log are routing checks only, never settlement authority.
library FccVerdict {
    bytes32 internal constant TEE_ACTION_RESULT_PREFIX = "TEE_ACTION_RESULT";

    // secp256k1 curve order / 2, used to reject non-canonical (high-S) sigs,
    // matching tee-node's own CheckCanonicalSignature.
    uint256 private constant SECP256K1_HALF_N = 0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0;

    /// @dev keccak256(keccak256(data) || id || keccak256(bytes(submissionTag)) || status)
    function actionResultHash(bytes memory data, bytes32 id, string memory submissionTag, uint8 status)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(keccak256(data), id, keccak256(bytes(submissionTag)), status));
    }

    /// @dev keccak256(abi.encode(bytes32("TEE_ACTION_RESULT"), chainId, dataHash))
    function payloadHash(uint256 chainId, bytes32 dataHash) internal pure returns (bytes32) {
        return keccak256(abi.encode(TEE_ACTION_RESULT_PREFIX, chainId, dataHash));
    }

    /// @dev EIP-191 personal-sign wrap, matching go-ethereum's accounts.TextHash
    /// over a 32-byte payload: keccak256("\x19Ethereum Signed Message:\n32" || hash).
    function ethSignedHash(bytes32 hash) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", hash));
    }

    /// @dev Recovers the signer of a full verdict, rejecting malformed and
    /// non-canonical (high-S) signatures. Returns address(0) on any rejection
    /// rather than reverting, so callers can attribute a clean "wrong signer"
    /// failure alongside genuinely wrong-key results.
    function recoverVerdictSigner(
        bytes memory data,
        bytes32 id,
        string memory submissionTag,
        uint8 status,
        uint256 chainId,
        bytes memory signature
    ) internal pure returns (address) {
        bytes32 arHash = actionResultHash(data, id, submissionTag, status);
        bytes32 digest = ethSignedHash(payloadHash(chainId, arHash));
        return recoverCanonical(digest, signature);
    }

    function recoverCanonical(bytes32 digest, bytes memory signature) internal pure returns (address) {
        if (signature.length != 65) return address(0);
        bytes32 r;
        bytes32 s;
        uint8 v;
        assembly ("memory-safe") {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }
        if (v < 27) v += 27;
        if (v != 27 && v != 28) return address(0);
        if (uint256(s) > SECP256K1_HALF_N) return address(0);
        return ecrecover(digest, v, r, s);
    }
}
