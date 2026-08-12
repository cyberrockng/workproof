import test from "node:test";
import assert from "node:assert/strict";
import {
  OP_COMMAND,
  OP_TYPE,
  PendingResultError,
  buildSettleArgs,
  castArgsForCall,
  fetchActionResponse,
  isPendingResultStatus,
  parseArgs,
  pollActionResponse,
  validateActionResponse
} from "../src/relay.js";

const INSTRUCTION = `0x${"11".repeat(32)}`;
const DATA = `0x${"22".repeat(64)}`;
const SIGNATURE = `0x${"33".repeat(65)}`;

function response(overrides = {}) {
  const { result: resultOverrides = {}, ...envelopeOverrides } = overrides;
  return {
    result: {
      id: INSTRUCTION,
      submissionTag: "submit",
      status: 1,
      log: "ok",
      opType: OP_TYPE,
      opCommand: OP_COMMAND,
      additionalResultStatus: "0x",
      version: "0.1.0",
      data: DATA,
      ...resultOverrides
    },
    signature: SIGNATURE,
    proxySignature: `0x${"44".repeat(65)}`,
    ...envelopeOverrides
  };
}

test("404 proxy result is pending", async () => {
  assert.equal(isPendingResultStatus(404), true);
  await assert.rejects(
    fetchActionResponse({
      proxyUrl: "https://proxy.example",
      instructionId: INSTRUCTION,
      fetchImpl: async () => ({ status: 404, ok: false })
    }),
    PendingResultError
  );
});

test("poll retries 404 then returns valid action response", async () => {
  let calls = 0;
  const action = await pollActionResponse({
    proxyUrl: "https://proxy.example",
    instructionId: INSTRUCTION,
    timeoutMs: 1_000,
    initialIntervalMs: 1,
    fetchImpl: async () => {
      calls++;
      if (calls === 1) return { status: 404, ok: false };
      return { status: 200, ok: true, json: async () => response() };
    },
    sleep: async () => {}
  });
  assert.equal(calls, 2);
  assert.equal(action.data, DATA);
});

test("malformed action responses are rejected before settlement", () => {
  assert.throws(() => validateActionResponse(response({ result: { status: 0 } }), { instructionId: INSTRUCTION }), /cannot settle/);
  assert.throws(
    () => validateActionResponse(response({ result: { opCommand: `0x${"00".repeat(32)}` } }), { instructionId: INSTRUCTION }),
    /opCommand/
  );
  assert.throws(
    () => validateActionResponse(response({ result: { id: `0x${"aa".repeat(32)}` } }), { instructionId: INSTRUCTION }),
    /does not match/
  );
});

test("wrong proxy signature is ignored; TEE signature remains settlement authority", () => {
  const action = validateActionResponse(response({ proxySignature: `0x${"ff".repeat(65)}` }), { instructionId: INSTRUCTION });
  assert.equal(action.proxySignature, `0x${"ff".repeat(65)}`);
  assert.equal(action.signature, SIGNATURE);
});

test("settleAttempt args preserve FCC routing and signature fields", () => {
  const action = validateActionResponse(response(), { instructionId: INSTRUCTION });
  assert.deepEqual(buildSettleArgs(7n, action), [7n, DATA, OP_TYPE, OP_COMMAND, "submit", 1, SIGNATURE]);
});

test("cast args support dry-run simulation and signed sends", () => {
  const escrow = `0x${"55".repeat(20)}`;
  const privateKey = `0x${"66".repeat(32)}`;
  const dryRun = castArgsForCall({
    escrow,
    signature: "expireVerification(uint256)",
    args: [7n],
    rpcUrl: "https://rpc.example",
    dryRun: true
  });
  assert.deepEqual(dryRun, ["call", escrow, "expireVerification(uint256)", "7", "--rpc-url", "https://rpc.example"]);

  const send = castArgsForCall({
    escrow,
    signature: "expireVerification(uint256)",
    args: [7n],
    rpcUrl: "https://rpc.example",
    privateKey
  });
  assert.deepEqual(send, [
    "send",
    escrow,
    "expireVerification(uint256)",
    "7",
    "--rpc-url",
    "https://rpc.example",
    "--private-key",
    privateKey,
    "--json"
  ]);
});

test("CLI parser defaults to relay and supports recovery commands", () => {
  assert.deepEqual(parseArgs(["--instruction", INSTRUCTION, "--job", "1"]).command, "relay");
  assert.deepEqual(parseArgs(["expire", "--job", "1"]).command, "expire");
  assert.deepEqual(parseArgs(["refund", "--job", "1", "--dry-run"]).args.dryRun, true);
});
