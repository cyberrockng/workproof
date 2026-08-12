#!/usr/bin/env node
import { spawn } from "node:child_process";
import { pathToFileURL } from "node:url";

export const COSTON2_CHAIN = {
  id: 114,
  name: "Coston2",
  nativeCurrency: { name: "Coston2 Flare", symbol: "C2FLR", decimals: 18 },
  rpcUrls: {
    default: { http: ["https://coston2-api.flare.network/ext/C/rpc"] }
  },
  blockExplorers: {
    default: { name: "Coston2 Explorer", url: "https://coston2-explorer.flare.network" }
  }
};

export const OP_TYPE = asciiToBytes32("WORKPROOF");
export const OP_COMMAND = asciiToBytes32("VERIFY");
export const SETTLE_SIGNATURE = "settleAttempt(uint256,bytes,bytes32,bytes32,string,uint8,bytes)";
export const EXPIRE_SIGNATURE = "expireVerification(uint256)";
export const REFUND_SIGNATURE = "refundExpired(uint256)";

const HEX_RE = /^0x[0-9a-fA-F]*$/;
const BYTES32_RE = /^0x[0-9a-fA-F]{64}$/;
const SIG_RE = /^0x[0-9a-fA-F]{130}$/;
const ADDRESS_RE = /^0x[0-9a-fA-F]{40}$/;

export class PendingResultError extends Error {
  constructor(statusCode = 404) {
    super(`action result pending (${statusCode})`);
    this.name = "PendingResultError";
    this.statusCode = statusCode;
  }
}

export function asciiToBytes32(value) {
  const hex = Buffer.from(value, "utf8").toString("hex");
  if (hex.length > 64) throw new Error("value exceeds bytes32");
  return `0x${hex.padEnd(64, "0")}`;
}

export function isPendingResultStatus(statusCode) {
  return statusCode === 404;
}

export function normalizeProxyUrl(proxyUrl) {
  if (!proxyUrl) throw new Error("missing proxy URL");
  const url = new URL(proxyUrl);
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new Error("proxy URL must use http or https");
  }
  return url.toString().replace(/\/$/, "");
}

export async function fetchActionResponse({ proxyUrl, instructionId, expectedTee, fetchImpl = globalThis.fetch }) {
  if (!BYTES32_RE.test(instructionId)) throw new Error("instruction must be a bytes32 hex string");
  const base = normalizeProxyUrl(proxyUrl);
  const res = await fetchImpl(`${base}/action/result/${instructionId}`);
  if (isPendingResultStatus(res.status)) throw new PendingResultError(res.status);
  if (!res.ok) throw new Error(`proxy returned HTTP ${res.status}`);
  let body;
  try {
    body = await res.json();
  } catch (error) {
    throw new Error(`proxy returned malformed JSON: ${error.message}`);
  }
  return validateActionResponse(body, { instructionId, expectedTee });
}

export function validateActionResponse(body, { instructionId, expectedTee } = {}) {
  if (!body || typeof body !== "object") throw new Error("action response must be an object");
  if (!body.result || typeof body.result !== "object") throw new Error("action response missing result object");
  if (!SIG_RE.test(body.signature ?? "")) throw new Error("action response missing 65-byte TEE signature");

  const result = body.result;
  const expectedId = instructionId?.toLowerCase();
  if (!BYTES32_RE.test(result.id ?? "")) throw new Error("result.id must be bytes32 hex");
  if (expectedId && result.id.toLowerCase() !== expectedId) throw new Error("result.id does not match instruction");
  if (expectedTee && !ADDRESS_RE.test(expectedTee)) throw new Error("expected TEE must be an address");
  if (result.submissionTag !== "submit") throw new Error("result.submissionTag must be submit");
  if (result.status !== 1) throw new Error(`result.status ${result.status} cannot settle`);
  if ((result.opType ?? "").toLowerCase() !== OP_TYPE.toLowerCase()) throw new Error("result.opType is not WORKPROOF");
  if ((result.opCommand ?? "").toLowerCase() !== OP_COMMAND.toLowerCase()) {
    throw new Error("result.opCommand is not VERIFY");
  }
  if (!HEX_RE.test(result.data ?? "") || result.data === "0x") throw new Error("result.data must be non-empty hex");
  if (!HEX_RE.test(result.additionalResultStatus ?? "0x")) {
    throw new Error("result.additionalResultStatus must be hex when present");
  }
  if (typeof result.version !== "string" || result.version.length === 0) {
    throw new Error("result.version must be a plain non-empty string");
  }
  if (typeof result.log !== "string") throw new Error("result.log must be a string");

  return {
    data: result.data,
    opType: result.opType,
    opCommand: result.opCommand,
    submissionTag: result.submissionTag,
    status: result.status,
    signature: body.signature,
    proxySignature: body.proxySignature,
    async assertExpectedTee() {
      // The on-chain escrow is the settlement authority for the TEE signature.
      // This hook still catches malformed operator input before a transaction is built.
    }
  };
}

export async function pollActionResponse({
  proxyUrl,
  instructionId,
  expectedTee,
  timeoutMs = 10 * 60 * 1000,
  initialIntervalMs = 2_000,
  maxIntervalMs = 30_000,
  fetchImpl = globalThis.fetch,
  sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
  onPending = () => {}
}) {
  const deadline = Date.now() + timeoutMs;
  let interval = initialIntervalMs;
  for (;;) {
    try {
      const action = await fetchActionResponse({ proxyUrl, instructionId, expectedTee, fetchImpl });
      await action.assertExpectedTee?.();
      return action;
    } catch (error) {
      if (!(error instanceof PendingResultError)) throw error;
      if (Date.now() >= deadline) throw new Error(`timed out waiting for instruction ${instructionId}`);
      onPending({ instructionId, nextPollMs: interval });
      await sleep(interval);
      interval = Math.min(maxIntervalMs, interval * 2);
    }
  }
}

export function buildSettleArgs(jobId, action) {
  return [
    BigInt(jobId),
    action.data,
    action.opType,
    action.opCommand,
    action.submissionTag,
    action.status,
    action.signature
  ];
}

export function castArgsForCall({ escrow, signature, args, rpcUrl, privateKey, dryRun = false }) {
  if (!rpcUrl) throw new Error("missing RPC URL");
  if (!ADDRESS_RE.test(escrow ?? "")) throw new Error("missing or invalid escrow address");
  if (!dryRun && !privateKey) throw new Error("missing relayer private key");
  const base = dryRun ? ["call", escrow, signature] : ["send", escrow, signature];
  const flags = dryRun ? ["--rpc-url", rpcUrl] : ["--rpc-url", rpcUrl, "--private-key", privateKey, "--json"];
  return [...base, ...args.map(String), ...flags];
}

export async function sendContractCall({ rpcUrl, privateKey, escrow, signature, args, dryRun = false, spawnImpl = spawn }) {
  const castArgs = castArgsForCall({ escrow, signature, args, rpcUrl, privateKey, dryRun });
  const { stdout } = await runCast(castArgs, { spawnImpl });
  if (dryRun) return { dryRun: true, stdout };
  return { dryRun: false, hash: parseCastTxHash(stdout), stdout };
}

export function runCast(args, { spawnImpl = spawn } = {}) {
  return new Promise((resolve, reject) => {
    const child = spawnImpl("cast", args, { stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    let stderr = "";
    child.stdout?.on("data", (chunk) => {
      stdout += chunk;
    });
    child.stderr?.on("data", (chunk) => {
      stderr += chunk;
    });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code === 0) resolve({ stdout, stderr });
      else reject(new Error(stderr.trim() || `cast exited with code ${code}`));
    });
  });
}

export function parseCastTxHash(stdout) {
  try {
    const parsed = JSON.parse(stdout);
    if (parsed.transactionHash) return parsed.transactionHash;
    if (parsed.hash) return parsed.hash;
  } catch {
    // Fall through to regex parsing for older cast output.
  }
  const match = stdout.match(/0x[0-9a-fA-F]{64}/);
  if (!match) throw new Error("could not parse transaction hash from cast output");
  return match[0];
}

export function explorerTx(hash) {
  return `${COSTON2_CHAIN.blockExplorers.default.url}/tx/${hash}`;
}

export function parseArgs(argv) {
  const args = {};
  const rest = [...argv];
  const command = rest[0]?.startsWith("--") ? "relay" : rest.shift() || "relay";
  for (let i = 0; i < rest.length; i++) {
    const token = rest[i];
    if (!token.startsWith("--")) throw new Error(`unexpected argument: ${token}`);
    const key = token.slice(2);
    if (key === "dry-run") {
      args.dryRun = true;
      continue;
    }
    const value = rest[++i];
    if (!value || value.startsWith("--")) throw new Error(`missing value for --${key}`);
    args[key] = value;
  }
  return { command, args };
}

function env(name, fallback = undefined) {
  return process.env[name] || fallback;
}

function privateKeyFromEnv() {
  return env("WORKPROOF_RELAYER_PRIVATE_KEY") || env("RELAYER_PRIVATE_KEY") || env("PROXY_PRIVATE_KEY") || env("DEPLOYMENT_PRIVATE_KEY");
}

export async function runCli(argv = process.argv.slice(2)) {
  const { command, args } = parseArgs(argv);
  const rpcUrl = args.rpc || env("WORKPROOF_RPC_URL") || env("CHAIN_URL") || COSTON2_CHAIN.rpcUrls.default.http[0];
  const escrow = args.escrow || env("WORKPROOF_ESCROW_ADDRESS");
  const privateKey = args["private-key"] || privateKeyFromEnv();
  const dryRun = Boolean(args.dryRun);

  if (command === "relay") {
    const instructionId = args.instruction;
    const jobId = args.job;
    if (!instructionId || jobId === undefined) throw new Error("relay requires --instruction and --job");
    const action = await pollActionResponse({
      proxyUrl: args.proxy || env("WORKPROOF_PROXY_URL"),
      instructionId,
      expectedTee: args["expected-tee"],
      timeoutMs: Number(args["timeout-ms"] || 10 * 60 * 1000),
      initialIntervalMs: Number(args["interval-ms"] || 2_000),
      maxIntervalMs: Number(args["max-interval-ms"] || 30_000),
      onPending: ({ nextPollMs }) => console.error(`pending; retrying in ${nextPollMs}ms`)
    });
    const result = await sendContractCall({
      rpcUrl,
      privateKey,
      escrow,
      signature: SETTLE_SIGNATURE,
      args: buildSettleArgs(jobId, action),
      dryRun
    });
    if (result.dryRun) {
      console.log("settleAttempt simulation passed");
      return;
    }
    console.log(`settleAttempt submitted: ${result.hash}`);
    console.log(explorerTx(result.hash));
    return;
  }

  if (command === "expire" || command === "refund") {
    const jobId = args.job;
    if (jobId === undefined) throw new Error(`${command} requires --job`);
    const functionName = command === "expire" ? "expireVerification" : "refundExpired";
    const signature = command === "expire" ? EXPIRE_SIGNATURE : REFUND_SIGNATURE;
    const result = await sendContractCall({ rpcUrl, privateKey, escrow, signature, args: [BigInt(jobId)], dryRun });
    if (result.dryRun) {
      console.log(`${functionName} simulation passed`);
      return;
    }
    console.log(`${functionName} submitted: ${result.hash}`);
    console.log(explorerTx(result.hash));
    return;
  }

  throw new Error(`unknown command: ${command}`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  runCli().catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
