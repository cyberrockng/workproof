export interface RelayTarget {
  instructionId: `0x${string}`;
  jobId: bigint;
  attempt: number;
}

export interface ActionResponseEnvelope {
  result: unknown;
  signature: `0x${string}`;
  proxySignature?: `0x${string}`;
}

export function isPendingResultStatus(statusCode: number): boolean {
  return statusCode === 404;
}
