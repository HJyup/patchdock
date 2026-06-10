import { randomBytes } from "node:crypto";

// Mirrors internal/contracts/id.go: "<prefix>-<12 hex chars>".
export function newId(prefix: string): string {
  return `${prefix}-${randomBytes(6).toString("hex")}`;
}
