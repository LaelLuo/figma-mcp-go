// Manual smoke test for the local fallback server.
// Usage:
//   bun scripts/manual-smoke.ts
//   bun scripts/manual-smoke.ts --nodeId=516:6761 --includeScreenshot=true
//   bun scripts/manual-smoke.ts --exe=D:/Applications/Scoop/shims/figma-mcp-go.exe
import { spawn } from "node:child_process";

type Json =
  | null
  | boolean
  | number
  | string
  | Json[]
  | { [key: string]: Json };

type JsonRpcMessage = {
  jsonrpc: "2.0";
  id?: number;
  method?: string;
  params?: Json;
  result?: Json;
  error?: {
    code: number;
    message: string;
    data?: Json;
  };
};

const DEFAULT_EXE = "D:/Applications/Scoop/shims/figma-mcp-go.exe";
const DEFAULT_TIMEOUT_MS = 30_000;

if (process.argv.includes("--help")) {
  console.log(`manual-smoke.ts

Options:
  --exe=<path>                Path to figma-mcp-go executable
  --fileKey=<value>           Optional compatibility fileKey to send
  --nodeId=<value>            Optional node ID for get_design_context
  --includeScreenshot=true    Request screenshot payload from get_design_context
  --timeoutMs=<number>        Overall timeout in milliseconds
`);
  process.exit(0);
}

const args = new Map<string, string>();
for (const rawArg of process.argv.slice(2)) {
  if (!rawArg.startsWith("--")) continue;
  const [key, ...rest] = rawArg.slice(2).split("=");
  args.set(key, rest.join("=") || "true");
}

const exePath = args.get("exe") || DEFAULT_EXE;
const timeoutMs = Number(args.get("timeoutMs") || DEFAULT_TIMEOUT_MS);
const nodeId = args.get("nodeId");
const includeScreenshot = args.get("includeScreenshot") === "true";
const fileKey = args.get("fileKey") || "dummy";

const plannedCalls = [
  {
    id: 1,
    name: "get_variable_defs",
    arguments: {},
  },
  {
    id: 2,
    name: "get_design_context",
    arguments: {
      fileKey,
      ...(nodeId ? { nodeId } : {}),
      excludeScreenshot: !includeScreenshot,
      forceCode: false,
    },
  },
  {
    id: 3,
    name: "use_figma",
    arguments: {
      fileKey,
      description: "Read current file name",
      code: "return figma.root.name",
    },
  },
] as const;

type RuntimeRole = "LEADER" | "FOLLOWER";

function pretty(value: unknown): string {
  return JSON.stringify(value, null, 2);
}

function summarizeContent(result: Record<string, unknown>): string {
  const content = Array.isArray(result.content) ? result.content : [];
  if (content.length === 0) return "(no content)";

  return content
    .map((entry, index) => {
      if (!entry || typeof entry !== "object") {
        return `[${index}] ${pretty(entry)}`;
      }

      const item = entry as Record<string, unknown>;
      if (item.type === "text") {
        const text = typeof item.text === "string" ? item.text : "";
        return `[${index}] text\n${text}`;
      }
      if (item.type === "image") {
        const mimeType = typeof item.mimeType === "string" ? item.mimeType : "unknown";
        const data = typeof item.data === "string" ? item.data : "";
        return `[${index}] image mime=${mimeType} bytes(base64)=${data.length}`;
      }
      return `[${index}] ${pretty(item)}`;
    })
    .join("\n\n");
}

async function main() {
  console.log(`Starting smoke test with ${exePath}`);

  const child = spawn(exePath, [], {
    stdio: ["pipe", "pipe", "pipe"],
    windowsHide: true,
  });

  let stdoutBuffer = "";
  let initialized = false;
  let observedRole: RuntimeRole | null = null;
  let printedRoleWarning = false;
  const pendingResults = new Set(plannedCalls.map((call) => call.id));

  const cleanup = () => {
    if (!child.killed) {
      child.kill();
    }
  };

  const timer = setTimeout(() => {
    console.error(`Smoke test timed out after ${timeoutMs}ms`);
    cleanup();
    process.exitCode = 1;
  }, timeoutMs);

  const send = (message: JsonRpcMessage) => {
    child.stdin.write(`${JSON.stringify(message)}\n`);
  };

  const maybePrintRoleWarning = () => {
    if (printedRoleWarning || observedRole !== "FOLLOWER") return;
    printedRoleWarning = true;
    console.warn(
      [
        "Warning: the spawned process became FOLLOWER.",
        "Requests were handled by the existing leader already listening on localhost:1994.",
        "If you want to verify this binary end-to-end, stop the current figma-mcp-go leader first and rerun the smoke test.",
      ].join(" "),
    );
  };

  child.stderr.on("data", (chunk) => {
    const text = chunk.toString("utf8");
    if (text.includes("role: FOLLOWER") || text.includes("became FOLLOWER")) {
      observedRole = "FOLLOWER";
    }
    if (text.includes("role: LEADER") || text.includes("became LEADER")) {
      observedRole = "LEADER";
    }
    process.stderr.write(`[mcp] ${text}`);
    maybePrintRoleWarning();
  });

  child.on("exit", (code, signal) => {
    clearTimeout(timer);
    if (pendingResults.size > 0 && process.exitCode !== 1) {
      console.error(
        `Smoke test exited before all responses arrived. remaining=${[
          ...pendingResults,
        ].join(",")} code=${code} signal=${signal}`,
      );
      process.exitCode = 1;
    }
  });

  child.stdout.on("data", (chunk) => {
    stdoutBuffer += chunk.toString("utf8");

    while (true) {
      const newlineIndex = stdoutBuffer.indexOf("\n");
      if (newlineIndex === -1) break;

      const line = stdoutBuffer.slice(0, newlineIndex).trim();
      stdoutBuffer = stdoutBuffer.slice(newlineIndex + 1);
      if (!line) continue;

      let message: JsonRpcMessage;
      try {
        message = JSON.parse(line);
      } catch (error) {
        console.error(`Failed to parse stdout line:\n${line}`);
        console.error(error);
        process.exitCode = 1;
        cleanup();
        return;
      }

      if (message.id === 0 && !initialized) {
        initialized = true;
        send({ jsonrpc: "2.0", method: "notifications/initialized" });
        for (const call of plannedCalls) {
          send({
            jsonrpc: "2.0",
            id: call.id,
            method: "tools/call",
            params: {
              name: call.name,
              arguments: call.arguments as unknown as Json,
            },
          });
        }
        continue;
      }

      if (typeof message.id !== "number") {
        console.log(`Notification:\n${pretty(message)}`);
        continue;
      }

      const matchedCall = plannedCalls.find((call) => call.id === message.id);
      if (!matchedCall) {
        console.log(`Unexpected response:\n${pretty(message)}`);
        continue;
      }

      maybePrintRoleWarning();
      pendingResults.delete(message.id);
      console.log(`\n=== ${matchedCall.name} ===`);

      if (message.error) {
        console.log(`JSON-RPC error:\n${pretty(message.error)}`);
      } else {
        const result =
          message.result && typeof message.result === "object"
            ? (message.result as Record<string, unknown>)
            : {};
        console.log(`isError: ${String(result.isError === true)}`);
        console.log(`content:\n${summarizeContent(result)}`);
        if ("structuredContent" in result) {
          console.log(`structuredContent:\n${pretty(result.structuredContent)}`);
        }
      }

      if (pendingResults.size === 0) {
        clearTimeout(timer);
        cleanup();
      }
    }
  });

  send({
    jsonrpc: "2.0",
    id: 0,
    method: "initialize",
    params: {
      protocolVersion: "2024-11-05",
      capabilities: {},
      clientInfo: {
        name: "figma-mcp-go-manual-smoke",
        version: "1.0.0",
      },
    },
  });
}

void main().catch((error) => {
  console.error(error);
  process.exit(1);
});
