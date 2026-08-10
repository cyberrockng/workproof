// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {FccVerdict} from "../contracts/lib/FccVerdict.sol";

/// @notice The mandatory FCC signature-compatibility spike from
/// WORKPROOF_EXECUTION_PLAN.md section 9. All constants below are copied
/// verbatim (via a generator script, not hand-typed) from
/// docs/evidence/fcc-signature-spike-v1.json, which was produced by signing
/// a real tee-node ActionResult with go-ethereum's real crypto.Sign +
/// accounts.TextHash and the real go-flare-common signing.Payload -- see
/// go/cmd/fcc-spike. If any assertion here fails, the automatic-settlement
/// design must stop per the plan's four-hour gate; a backend-signed
/// substitute is not acceptable.
contract FccSignatureSpikeTest is Test {
    bytes constant DATA = hex"7b226f7574636f6d65223a2250415353222c22617474656d7074223a317d";
    bytes32 constant ID = 0x0000000000000000000000000000000000000000000000000000000000001111;
    string constant TAG = "submit";
    uint8 constant STATUS = 1;

    address constant EXPECTED_SIGNER = 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266;

    uint256 constant CHAIN_ID = 114;
    bytes32 constant EXPECTED_ACTION_RESULT_HASH = 0x44c661155f727f2ed2206c839f94e9d0b0bcc66a425845d2759ec4544a3bdd4a;
    bytes32 constant EXPECTED_PAYLOAD_HASH = 0xd333554b98a27804ad3569d96ebe74b539661700d94c4572ccc74d13c297d4ed;
    bytes constant SIGNATURE =
        hex"42dd2e9b3d17f523d9891abc59d5036a248f155fa426b55549b4b08f5283cf0b1e6218f260da5a279240dc642e47b286b3d89b6fe9d6b3712b7bac69939ca21901";

    uint256 constant OTHER_CHAIN_ID = 16;
    bytes constant OTHER_CHAIN_SIGNATURE =
        hex"4b070174d2c8b0de68a3f2e995e19a43e2b6ef410423a15b731feb8e62e49681760c5c2d6f7749d19287f4275c78a899396b77add3de65b95bed3f52dfd7edfd01";

    function testDigestMatchesGoVector() external pure {
        bytes32 arHash = FccVerdict.actionResultHash(DATA, ID, TAG, STATUS);
        assertEq(arHash, EXPECTED_ACTION_RESULT_HASH, "actionResultHash mismatch vs Go");
        bytes32 pHash = FccVerdict.payloadHash(CHAIN_ID, arHash);
        assertEq(pHash, EXPECTED_PAYLOAD_HASH, "payloadHash mismatch vs Go");
    }

    function testRecoversRealTeeSignature() external pure {
        address signer = FccVerdict.recoverVerdictSigner(DATA, ID, TAG, STATUS, CHAIN_ID, SIGNATURE);
        assertEq(signer, EXPECTED_SIGNER, "did not recover the real Go-signed TEE address");
    }

    function testCrossChainSignatureDoesNotReplay() external pure {
        address wrongChainSigner =
            FccVerdict.recoverVerdictSigner(DATA, ID, TAG, STATUS, CHAIN_ID, OTHER_CHAIN_SIGNATURE);
        assertTrue(wrongChainSigner != EXPECTED_SIGNER, "cross-chain signature replayed");

        address rightChainSigner =
            FccVerdict.recoverVerdictSigner(DATA, ID, TAG, STATUS, OTHER_CHAIN_ID, OTHER_CHAIN_SIGNATURE);
        assertEq(rightChainSigner, EXPECTED_SIGNER, "same signature invalid on its own chain");
    }

    function testMutatedDataFailsRecovery() external pure {
        bytes memory mutated = bytes.concat(DATA, hex"00");
        address signer = FccVerdict.recoverVerdictSigner(mutated, ID, TAG, STATUS, CHAIN_ID, SIGNATURE);
        assertTrue(signer != EXPECTED_SIGNER, "mutated data still recovered real signer");
    }

    function testMutatedIdFailsRecovery() external pure {
        bytes32 mutatedId = bytes32(uint256(ID) + 1);
        address signer = FccVerdict.recoverVerdictSigner(DATA, mutatedId, TAG, STATUS, CHAIN_ID, SIGNATURE);
        assertTrue(signer != EXPECTED_SIGNER, "mutated id still recovered real signer");
    }

    function testMutatedSubmissionTagFailsRecovery() external pure {
        address signer = FccVerdict.recoverVerdictSigner(DATA, ID, "end", STATUS, CHAIN_ID, SIGNATURE);
        assertTrue(signer != EXPECTED_SIGNER, "mutated submissionTag still recovered real signer");
    }

    function testMutatedStatusFailsRecovery() external pure {
        address signer = FccVerdict.recoverVerdictSigner(DATA, ID, TAG, 0, CHAIN_ID, SIGNATURE);
        assertTrue(signer != EXPECTED_SIGNER, "mutated status still recovered real signer");
    }

    function testMalformedSignatureLengthReturnsZero() external pure {
        bytes memory tooShort = hex"1234";
        address signer = FccVerdict.recoverVerdictSigner(DATA, ID, TAG, STATUS, CHAIN_ID, tooShort);
        assertEq(signer, address(0), "malformed signature length did not reject cleanly");
    }

    function testHighSSignatureRejected() external pure {
        bytes memory sig = SIGNATURE;
        bytes32 r;
        bytes32 s;
        uint8 v;
        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }
        uint256 n = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141;
        bytes32 highS = bytes32(n - uint256(s));
        uint8 flippedV = v == 27 ? 28 : 27;
        bytes memory malleated = abi.encodePacked(r, highS, flippedV);
        address signer = FccVerdict.recoverVerdictSigner(DATA, ID, TAG, STATUS, CHAIN_ID, malleated);
        assertTrue(signer != EXPECTED_SIGNER, "high-S malleated signature was accepted");
    }

    /// @dev A v byte that is neither 27/28 nor the raw 0/1 that gets
    /// normalized to them must be rejected cleanly (return address(0)), not
    /// passed through to ecrecover with an out-of-range v.
    function testInvalidVByteReturnsZero() external pure {
        bytes memory sig = SIGNATURE;
        bytes32 r;
        bytes32 s;
        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
        }
        uint8 invalidV = 99;
        bytes memory malformed = abi.encodePacked(r, s, invalidV);
        address signer = FccVerdict.recoverVerdictSigner(DATA, ID, TAG, STATUS, CHAIN_ID, malformed);
        assertEq(signer, address(0), "invalid v byte was not rejected cleanly");
    }
}

