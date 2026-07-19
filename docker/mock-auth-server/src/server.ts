// server.ts は疑似 OIDC 認証サーバーの HTTP エントリポイント。
// 提供範囲は署名・Claim 検証まで（A 範囲）: Discovery / JWKS / /health / /test/token（固定 User + A 異常系）。
// Authorization Code Flow / Login UI / /userinfo / Key Rotation は後続フェーズ。
import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { jwks, ALG } from "./keys.ts";
import { issueToken, PROFILES, ACCESS_TTL_SECONDS } from "./tokens.ts";
import type { OidcConfig, User } from "./types.ts";

const config: OidcConfig = {
  port: Number(process.env.OIDC_PORT ?? "4000"),
  issuer: process.env.OIDC_ISSUER ?? "http://localhost:4000",
  audience: process.env.OIDC_AUDIENCE ?? "go-boilerplate-api",
  clientId: process.env.OIDC_CLIENT_ID ?? "go-boilerplate-client",
  // testEndpointsEnabled は /test/* を有効にするか。本番相当では 'disabled' にして無効化する（受入条件5）。
  testEndpointsEnabled: (process.env.MOCK_AUTH_TEST_ENDPOINTS ?? "enabled") !== "disabled",
};

// loadUsers は固定 User Fixture を読み込む。fixtures/users.json はサンプルデータのため
// 欠如し得る（サンプル削除で除去 → reset-mock-auth-users.mjs で再生成）。欠如・破損時は
// 空扱いで継続する（トークン発行は任意 subject で可能なため、mock は停止させない）。
function loadUsers(path: string): User[] {
  try {
    return JSON.parse(readFileSync(path, "utf8")) as User[];
  } catch {
    return [];
  }
}

const usersPath = fileURLToPath(new URL("../fixtures/users.json", import.meta.url));
const users = loadUsers(usersPath);

// defaultSubject は subject 省略時のフォールバック。fixtures があれば先頭 User、無ければ中立値。
// サンプル固有名を実装へ焼き込まないため、既定はデータ側から導出する。
const defaultSubject = users[0]?.subject ?? "user-example";

// discoveryDocument は OIDC Discovery（/.well-known/openid-configuration）のレスポンス。
function discoveryDocument() {
  return {
    issuer: config.issuer,
    jwks_uri: `${config.issuer}/.well-known/jwks.json`,
    authorization_endpoint: `${config.issuer}/authorize`,
    token_endpoint: `${config.issuer}/token`,
    response_types_supported: ["code"],
    grant_types_supported: ["authorization_code"],
    subject_types_supported: ["public"],
    id_token_signing_alg_values_supported: [ALG],
    scopes_supported: ["openid", "profile", "email", "api.read", "api.write"],
    token_endpoint_auth_methods_supported: ["client_secret_basic"],
    code_challenge_methods_supported: ["S256"],
    claims_supported: ["sub", "iss", "aud", "exp", "iat", "nbf", "email", "name"],
  };
}

// sendJSON は JSON レスポンスを返す。
function sendJSON(res: ServerResponse, status: number, body: unknown): void {
  const payload = JSON.stringify(body);
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(payload);
}

// readJSONBody はリクエストボディを JSON として読み取る。
function readJSONBody(req: IncomingMessage): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on("data", (chunk: Buffer) => chunks.push(chunk));
    req.on("end", () => {
      const raw = Buffer.concat(chunks).toString("utf8");
      if (raw.trim() === "") {
        resolve({});
        return;
      }
      try {
        resolve(JSON.parse(raw) as Record<string, unknown>);
      } catch (err) {
        reject(err as Error);
      }
    });
    req.on("error", reject);
  });
}

// handleTestToken は POST /test/token を処理し、{subject, profile} から Token を発行する。
async function handleTestToken(req: IncomingMessage, res: ServerResponse): Promise<void> {
  let body: Record<string, unknown>;
  try {
    body = await readJSONBody(req);
  } catch {
    sendJSON(res, 400, { error: "invalid_request", error_description: "body must be valid JSON" });
    return;
  }

  const subject = typeof body.subject === "string" ? body.subject : defaultSubject;
  const profile = typeof body.profile === "string" ? body.profile : "valid";
  if (!PROFILES.includes(profile)) {
    sendJSON(res, 400, { error: "invalid_request", error_description: `unknown profile: ${profile}` });
    return;
  }

  const accessToken = await issueToken(config, subject, profile);
  sendJSON(res, 200, {
    access_token: accessToken,
    token_type: "Bearer",
    expires_in: ACCESS_TTL_SECONDS,
  });
}

const server = createServer((req: IncomingMessage, res: ServerResponse) => {
  const url = new URL(req.url ?? "/", `http://${req.headers.host ?? "localhost"}`);
  const path = url.pathname;

  if (req.method === "GET" && path === "/health") {
    sendJSON(res, 200, { status: "ok" });
    return;
  }
  if (req.method === "GET" && path === "/.well-known/openid-configuration") {
    sendJSON(res, 200, discoveryDocument());
    return;
  }
  if (req.method === "GET" && path === "/.well-known/jwks.json") {
    sendJSON(res, 200, jwks);
    return;
  }

  if (path.startsWith("/test/")) {
    if (!config.testEndpointsEnabled) {
      // 本番相当では /test/* を無効化する（受入条件5）。
      sendJSON(res, 404, { error: "not_found" });
      return;
    }
    if (req.method === "POST" && path === "/test/token") {
      void handleTestToken(req, res);
      return;
    }
    if (req.method === "GET" && path === "/test/users") {
      sendJSON(res, 200, { users });
      return;
    }
  }

  sendJSON(res, 404, { error: "not_found" });
});

server.listen(config.port, () => {
  console.log(
    JSON.stringify({
      msg: "mock-auth-server started",
      issuer: config.issuer,
      audience: config.audience,
      port: config.port,
      users: users.length,
      test_endpoints: config.testEndpointsEnabled ? "enabled" : "disabled",
    }),
  );
});
