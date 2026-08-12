// server-entry.integration.test.ts は src/server.ts を子プロセスとして起動し、本番拒否ガードと
// 通常起動時の応答を検証する（カバレッジ対象外にしている理由は README の Tests 節）。
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { fileURLToPath } from "node:url";

const entry = fileURLToPath(new URL("../src/server.ts", import.meta.url));

test("本番モードでは起動を拒否し、終了コード 1 で落ちる", async () => {
  const child = spawn(process.execPath, [entry], {
    env: { ...process.env, NODE_ENV: "production" },
    stdio: ["ignore", "ignore", "pipe"],
  });

  let stderr = "";
  child.stderr.on("data", (chunk: Buffer) => {
    stderr += chunk.toString("utf8");
  });

  const [code] = (await once(child, "exit")) as [number | null];
  assert.equal(code, 1);
  assert.match(stderr, /must not run in production/);
});

test("通常起動では待ち受けを開始し /health に応答する", async () => {
  const child = spawn(process.execPath, [entry], {
    env: { ...process.env, NODE_ENV: "development", OIDC_PORT: "0" },
    stdio: ["ignore", "pipe", "inherit"],
  });

  try {
    let stdout = "";
    for await (const chunk of child.stdout) {
      stdout += (chunk as Buffer).toString("utf8");
      if (stdout.includes("\n")) {
        break;
      }
    }

    const started = JSON.parse(stdout.split("\n")[0]) as { msg: string; port: number };
    assert.equal(started.msg, "mock-auth-server started");

    const res = await fetch(`http://localhost:${started.port}/health`);
    assert.equal(res.status, 200);
    assert.deepEqual(await res.json(), { status: "ok" });
  } finally {
    child.kill();
    await once(child, "exit");
  }
});
