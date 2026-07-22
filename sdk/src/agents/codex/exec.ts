import { spawn } from "node:child_process";
import { access, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { homedir, tmpdir } from "node:os";
import { join } from "node:path";
import { createInterface } from "node:readline";
import type { StageLogEvent } from "../../context.ts";
import type { JsonSchema } from "./schemas.ts";

type JsonRecord = Record<string, unknown>;
type CodexUsageField =
  | "input_tokens"
  | "cached_input_tokens"
  | "output_tokens"
  | "reasoning_output_tokens"
  | "total_tokens";

export type CodexUsage = Partial<Record<CodexUsageField, number>>;
export type CodexLogEvent = StageLogEvent & { source: "codex" };

export interface CodexInvocation {
  prompt: string;
  cwd: string;
  outputSchema: JsonSchema;
  log: (event: CodexLogEvent) => void;
  model?: string;
}

export interface CodexResult {
  lastMessage: string;
}

// Codex reads credentials from $CODEX_HOME/auth.json (default ~/.codex).
// Inside a stage container that file must be mounted in; fail fast with the
// fix rather than letting codex die mid-run.
async function assertAuth(): Promise<void> {
  if (process.env.OPENAI_API_KEY) return;
  const authFile = join(process.env.CODEX_HOME ?? join(homedir(), ".codex"), "auth.json");
  try {
    await access(authFile);
  } catch {
    throw new Error(
      `codex auth not found at ${authFile} — mount your ~/.codex/auth.json into the container, or set OPENAI_API_KEY`,
    );
  }
}

// runCodex shells out to `codex exec` — Codex is the complete agent (model
// loop + read/write/shell tools); this wrapper only adapts its inputs and
// outputs to the stage contract.
export async function runCodex(inv: CodexInvocation): Promise<CodexResult> {
  await assertAuth();

  const dir = await mkdtemp(join(tmpdir(), "patchdock-codex-"));
  const lastMessageFile = join(dir, "last-message.txt");
  const schemaFile = join(dir, "output-schema.json");
  await writeFile(schemaFile, JSON.stringify(inv.outputSchema));

  const args = [
    "exec",
    // The Docker container is the sandbox; codex's own sandbox/approvals
    // would either double-restrict or hang waiting for a human.
    "--dangerously-bypass-approvals-and-sandbox",
    "--json",
    "--cd",
    inv.cwd,
    "--output-last-message",
    lastMessageFile,
    "--output-schema",
    schemaFile,
  ];
  if (inv.model) {
    args.push("--model", inv.model);
  }
  args.push(inv.prompt);

  try {
    await stream(args, inv.log);
    let lastMessage: string;
    try {
      lastMessage = await readFile(lastMessageFile, "utf8");
    } catch (cause) {
      throw new Error("codex completed without writing its final message", { cause });
    }
    return { lastMessage: lastMessage.trim() };
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

function stream(args: string[], log: (event: CodexLogEvent) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const child = spawn("codex", args, { stdio: ["ignore", "pipe", "pipe"] });
    let failure = "";
    let malformedLines = 0;
    let stderrLines = 0;
    let lastStderr = "";
    let settled = false;

    log({ source: "codex", event: "process_started", level: "info" });

    const stdout = createInterface({ input: child.stdout, crlfDelay: Infinity });
    stdout.on("line", (line) => {
      if (line === "") return;
      let raw: unknown;
      try {
        raw = JSON.parse(line);
      } catch {
        malformedLines++;
        return;
      }

      failure = handleEvent(raw, log) ?? failure;
    });

    // Codex reserves stdout for JSONL. Stderr is retained as a bounded
    // diagnostic summary rather than copied line-for-line into the audit log.
    const stderr = createInterface({ input: child.stderr, crlfDelay: Infinity });
    stderr.on("line", (line) => {
      if (line === "") return;
      stderrLines++;
      lastStderr = safeDiagnostic(line, 500);
    });

    child.once("error", (err) => {
      if (settled) return;
      settled = true;
      const message = err.message.includes("ENOENT")
        ? "codex CLI not found on PATH — install it in the agent image (npm install -g @openai/codex)"
        : safeDiagnostic(err.message, 500);
      log({ source: "codex", event: "process_error", level: "error", message });
      reject(new Error(message, { cause: err }));
    });

    child.once("close", (code) => {
      if (settled) return;
      settled = true;

      if (stderrLines > 0) {
        log({
          source: "codex",
          event: "stderr_summary",
          level: code === 0 ? "warn" : "error",
          line_count: stderrLines,
          last_message: lastStderr,
        });
      }
      if (malformedLines > 0) {
        log({
          source: "codex",
          event: "protocol_warning",
          level: "warn",
          malformed_line_count: malformedLines,
        });
      }

      log({
        source: "codex",
        event: "process_finished",
        level: code === 0 ? "info" : "error",
        exit_code: code,
      });

      if (code === 0) {
        resolve();
      } else {
        const detail = failure || lastStderr || "no structured error was emitted";
        reject(new Error(`codex exec exited with code ${code}; ${detail}`));
      }
    });
  });
}

function handleEvent(
  raw: unknown,
  log: (event: CodexLogEvent) => void,
): string | undefined {
  const event = asRecord(raw);
  if (!event) return undefined;

  const type = stringValue(event.type);
  switch (type) {
    case "thread.started": {
      const output: CodexLogEvent = {
        source: "codex",
        event: "session_started",
        level: "info",
      };
      setString(output, "thread_id", event.thread_id);
      log(output);
      return undefined;
    }
    case "turn.completed": {
      const parsedUsage = parseUsage(event.usage);
      const output: CodexLogEvent = {
        source: "codex",
        event: "turn_completed",
        level: "info",
      };
      if (parsedUsage) output.usage = parsedUsage;
      log(output);
      return undefined;
    }
    case "turn.failed":
    case "error": {
      const message = errorMessage(event) ?? "Codex reported an unspecified failure";
      log({ source: "codex", event: "turn_failed", level: "error", message });
      return message;
    }
    case "item.completed":
      logCompletedItem(event.item, log);
      return undefined;
    default:
      return undefined;
  }
}

function logCompletedItem(raw: unknown, log: (event: CodexLogEvent) => void): void {
  const item = asRecord(raw);
  if (!item) return;

  const itemType = stringValue(item.type);
  if (itemType === "command_execution") {
    const output: CodexLogEvent = {
      source: "codex",
      event: "command_completed",
      level: "info",
    };
    setString(output, "item_id", item.id);
    setString(output, "status", item.status);
    setNumber(output, "exit_code", item.exit_code);
    const command = stringValue(item.command);
    if (command) output.command = summarizeCommand(command);
    log(output);
    return;
  }

  if (itemType === "file_change") {
    const output: CodexLogEvent = {
      source: "codex",
      event: "file_change_completed",
      level: "info",
      changes: summarizeChanges(item),
    };
    setString(output, "item_id", item.id);
    setString(output, "status", item.status);
    log(output);
    return;
  }

  if (itemType === "mcp_tool_call") {
    const output: CodexLogEvent = {
      source: "codex",
      event: "tool_call_completed",
      level: "info",
    };
    setString(output, "item_id", item.id);
    setString(output, "server", item.server ?? item.server_name);
    setString(output, "tool", item.tool ?? item.tool_name);
    setString(output, "status", item.status);
    log(output);
    return;
  }

  // Agent messages, reasoning, command output, tool arguments/results, and
  // patch bodies are intentionally excluded from the audit stream.
}

function summarizeChanges(item: JsonRecord): Array<{ path: string; kind?: string }> {
  const rawChanges = Array.isArray(item.changes) ? item.changes : [];
  const changes: Array<{ path: string; kind?: string }> = [];

  for (const raw of rawChanges.slice(0, 100)) {
    const change = asRecord(raw);
    if (!change) continue;
    const path = stringValue(change.path);
    if (!path) continue;
    const kind = stringValue(change.kind ?? change.type);
    changes.push(
      kind ? { path: safeText(path, 500), kind } : { path: safeText(path, 500) },
    );
  }

  const directPath = stringValue(item.path);
  if (changes.length === 0 && directPath) {
    changes.push({ path: safeText(directPath, 500) });
  }
  return changes;
}

function summarizeCommand(command: string): string {
  return redactSensitive(safeText(command.replace(/\s+/g, " ").trim(), 500));
}

function parseUsage(raw: unknown): CodexUsage | undefined {
  const input = asRecord(raw);
  if (!input) return undefined;

  const output: CodexUsage = {};
  const fields: CodexUsageField[] = [
    "input_tokens",
    "cached_input_tokens",
    "output_tokens",
    "reasoning_output_tokens",
    "total_tokens",
  ];
  for (const field of fields) {
    const value = input[field];
    if (typeof value === "number" && Number.isFinite(value) && value >= 0) {
      output[field] = value;
    }
  }
  return Object.keys(output).length > 0 ? output : undefined;
}

function errorMessage(event: JsonRecord): string | undefined {
  const direct = stringValue(event.message);
  if (direct) return safeDiagnostic(direct, 500);
  const error = asRecord(event.error);
  const nested = error ? stringValue(error.message) : undefined;
  return nested ? safeDiagnostic(nested, 500) : undefined;
}

function asRecord(value: unknown): JsonRecord | undefined {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as JsonRecord)
    : undefined;
}

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" && value !== "" ? value : undefined;
}

function setString(target: JsonRecord, key: string, value: unknown): void {
  const parsed = stringValue(value);
  if (parsed) target[key] = safeText(parsed, 500);
}

function setNumber(target: JsonRecord, key: string, value: unknown): void {
  if (typeof value === "number" && Number.isFinite(value)) target[key] = value;
}

function safeText(value: string, maximum: number): string {
  return value.length <= maximum ? value : `${value.slice(0, maximum - 1)}…`;
}

function safeDiagnostic(value: string, maximum: number): string {
  return redactSensitive(safeText(value, maximum));
}

function redactSensitive(value: string): string {
  return value
    .replace(/(authorization:\s*bearer\s+)[^\s'"]+/gi, "$1[REDACTED]")
    .replace(/((?:api[_-]?key|token|password|secret)\s*=\s*)[^\s'"]+/gi, "$1[REDACTED]");
}
