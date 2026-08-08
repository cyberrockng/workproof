import { mkdir, readFile, writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = fileURLToPath(new URL("..", import.meta.url));
const sourcePath = join(root, "fixtures", "workproof-fixture-v1.json");
const fixture = JSON.parse(await readFile(sourcePath, "utf8"));
const verdict = fixture.verdict;

const generatedRoot = join(root, "generated");
await mkdir(join(generatedRoot, "typescript"), { recursive: true });
await mkdir(join(generatedRoot, "go"), { recursive: true });
await mkdir(join(generatedRoot, "solidity"), { recursive: true });

const header = "Generated from packages/schema/fixtures/workproof-fixture-v1.json. Do not edit by hand.";

await writeFile(
  join(generatedRoot, "typescript", "workproofFixture.ts"),
  `// ${header}
export const workProofFixture = ${JSON.stringify(fixture, null, 2)} as const;
`
);

await writeFile(
  join(generatedRoot, "go", "workproof_fixture.go"),
  `// ${header}
package fixtures

const FixtureName = ${JSON.stringify(fixture.name)}
const ChainID uint64 = ${verdict.chainId}
const EscrowAddress = ${JSON.stringify(verdict.escrowAddress)}
const ArtifactAddress = ${JSON.stringify(verdict.artifactAddress)}
const SpecHash = ${JSON.stringify(verdict.specHash)}
const PrivateBundleHash = ${JSON.stringify(verdict.privateBundleHash)}
const InstructionID = ${JSON.stringify(verdict.instructionId)}
const ArtifactCodeHash = ${JSON.stringify(verdict.artifactCodeHash)}
const RandomValueHash = ${JSON.stringify(verdict.randomValueHash)}
const EngineVersionHash = ${JSON.stringify(verdict.engineVersionHash)}
const ReportHash = ${JSON.stringify(verdict.reportHash)}
`
);

await writeFile(
  join(generatedRoot, "solidity", "WorkProofFixture.sol"),
  `// SPDX-License-Identifier: MIT
// ${header}
pragma solidity ^0.8.27;

library WorkProofFixture {
    string internal constant FIXTURE_NAME = "${fixture.name}";
    uint256 internal constant CHAIN_ID = ${verdict.chainId};
    address internal constant ESCROW_ADDRESS = ${verdict.escrowAddress};
    address internal constant ARTIFACT_ADDRESS = ${verdict.artifactAddress};
    bytes32 internal constant SPEC_HASH = ${verdict.specHash};
    bytes32 internal constant PRIVATE_BUNDLE_HASH = ${verdict.privateBundleHash};
    bytes32 internal constant INSTRUCTION_ID = ${verdict.instructionId};
    bytes32 internal constant ARTIFACT_CODE_HASH = ${verdict.artifactCodeHash};
    bytes32 internal constant RANDOM_VALUE_HASH = ${verdict.randomValueHash};
    bytes32 internal constant ENGINE_VERSION_HASH = ${verdict.engineVersionHash};
    bytes32 internal constant REPORT_HASH = ${verdict.reportHash};
}
`
);

console.log("generated TypeScript, Go, and Solidity fixtures");
