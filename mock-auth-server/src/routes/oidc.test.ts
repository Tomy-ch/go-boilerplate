// oidc.test.ts は Authorization Code Flow + PKCE を app.request（プロセス内）で e2e 検証する。
import { describe, expect, it, beforeEach } from "vitest";
import { SignJWT } from "jose";
import type { JWTPayload } from "jose";
import { createApp } from "../router.ts";
import { s256Challenge } from "../pkce.ts";
import { codeStore, sessionStore } from "../store.ts";
import { keyStore, ALG } from "../keys.ts";
import { loadConfig } from "../config.ts";

const ISSUER = loadConfig().issuer;

// 揮発ストアは module singleton のため、テスト間の状態汚染を防ぐ。
beforeEach(() => {
  codeStore.clear();
  sessionStore.clear();
});

const CLIENT_ID = "go-boilerplate-client";
const REDIRECT_URI = "http://localhost:3000/api/auth/callback";
const VERIFIER = "test-code-verifier-abcdefghijklmnopqrstuvwxyz-0123456789";
const CHALLENGE = s256Challenge(VERIFIER);

function authorizePath(params: Record<string, string>): string {
  const search = new URLSearchParams(params);
  return `/oidc/authorize?${search.toString()}`;
}

function decodeJwtPart(part: string): Record<string, unknown> {
  return JSON.parse(Buffer.from(part, "base64url").toString("utf8")) as Record<string, unknown>;
}

async function issueCode(
  app: ReturnType<typeof createApp>,
  extra: Record<string, string> = {},
): Promise<URL> {
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: "openid profile",
      code_challenge: CHALLENGE,
      code_challenge_method: "S256",
      ...extra,
    }),
  );
  expect(res.status).toBe(302);
  return new URL(res.headers.get("location") ?? "");
}

async function tokenRequest(app: ReturnType<typeof createApp>, form: Record<string, string>): Promise<Response> {
  return app.request("/oidc/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams(form).toString(),
  });
}

async function logoutRequest(app: ReturnType<typeof createApp>, form: Record<string, string>): Promise<Response> {
  return app.request("/oidc/logout", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams(form).toString(),
  });
}

// issueTokens は authorize→token を通し access_token / id_token を取り出す（subject は defaultSubject）。
async function issueTokens(
  app: ReturnType<typeof createApp>,
): Promise<{ access_token: string; id_token: string }> {
  const location = await issueCode(app);
  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });
  expect(res.status).toBe(200);
  return (await res.json()) as { access_token: string; id_token: string };
}

it("正常系: openid を含まない scope では id_token を返さない", async () => {
  const app = createApp();
  const location = await issueCode(app, { scope: "api.read" });

  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });

  expect(res.status).toBe(200);
  const body = (await res.json()) as { access_token: string; scope: string; id_token?: string };
  expect(body.scope).toBe("api.read");
  expect(body.id_token).toBe(undefined);
  expect(body.access_token).toBeTruthy();
});

it("正常系: authorize→token で access/id token を発行し claim が正しい", async () => {
  const app = createApp();
  const location = await issueCode(app, { state: "state-xyz", nonce: "nonce-123" });
  const code = location.searchParams.get("code");
  expect(code).toBeTruthy();
  expect(location.searchParams.get("state")).toBe("state-xyz");

  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: code ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });
  expect(res.status).toBe(200);
  const body = (await res.json()) as {
    token_type: string;
    access_token: string;
    id_token: string;
    expires_in: number;
    scope: string;
  };
  expect(body.token_type).toBe("Bearer");
  expect(typeof body.expires_in).toBe("number");
  expect(body.scope).toBe("openid profile");

  const [atHeader, atClaims] = body.access_token.split(".");
  expect(decodeJwtPart(atHeader).typ).toBe("at+jwt");
  expect(decodeJwtPart(atHeader).kid).toBe("mock-key-1");
  expect(decodeJwtPart(atClaims).sub).toBe("user-john-doe");
  expect(decodeJwtPart(atClaims).iss).toBe(ISSUER);
  expect(decodeJwtPart(atClaims).aud).toBe("go-boilerplate-api");
  expect(decodeJwtPart(atClaims).token_use).toBe("access");

  const idClaims = decodeJwtPart(body.id_token.split(".")[1]);
  expect(idClaims.sub).toBe("user-john-doe");
  expect(idClaims.iss).toBe(ISSUER);
  expect(idClaims.aud).toBe(CLIENT_ID);
  expect(idClaims.token_use).toBe("id");
  expect(idClaims.nonce).toBe("nonce-123");
});

it("異常系: code の再利用は 400 invalid_grant で拒否する", async () => {
  const app = createApp();
  const location = await issueCode(app);
  const code = location.searchParams.get("code") ?? "";
  const form = {
    grant_type: "authorization_code",
    code,
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  };
  expect((await tokenRequest(app, form)).status).toBe(200);
  const reuse = await tokenRequest(app, form);
  expect(reuse.status).toBe(400);
  expect(((await reuse.json()) as Record<string, string>).error).toBe("invalid_grant");
});

it("異常系: code_verifier 不一致は 400 invalid_grant で拒否する", async () => {
  const app = createApp();
  const location = await issueCode(app);
  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: "wrong-verifier",
  });
  expect(res.status).toBe(400);
  expect(((await res.json()) as Record<string, string>).error).toBe("invalid_grant");
});

it("異常系: 未登録 redirect_uri は 400（リダイレクト不能）で拒否する", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: "http://evil.example.com/callback",
      scope: "openid",
      code_challenge: CHALLENGE,
      code_challenge_method: "S256",
    }),
  );
  expect(res.status).toBe(400);
});

it("異常系: code_challenge_method!=S256 は 302 error redirect（state を反映）", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: "openid",
      code_challenge: CHALLENGE,
      code_challenge_method: "plain",
      state: "state-abc",
    }),
  );
  expect(res.status).toBe(302);
  const location = new URL(res.headers.get("location") ?? "");
  expect(location.searchParams.get("error")).toBe("invalid_request");
  expect(location.searchParams.get("state")).toBe("state-abc");
});

it("異常系: token で redirect_uri 不一致は 400 invalid_grant で拒否する", async () => {
  const app = createApp();
  const location = await issueCode(app);
  const res = await tokenRequest(app, {
    grant_type: "authorization_code",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: "http://localhost:3000/other",
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });
  expect(res.status).toBe(400);
});

it("正常系: userinfo は Bearer access token で whitelist フィールドのみ返す", async () => {
  const app = createApp();
  const { access_token } = await issueTokens(app);
  const res = await app.request("/oidc/userinfo", {
    headers: { authorization: `Bearer ${access_token}` },
  });
  expect(res.status).toBe(200);
  const info = (await res.json()) as Record<string, unknown>;
  expect(info.sub).toBe("user-john-doe");
  expect(info.email).toBe("john.doe@example.com");
  expect(info.email_verified).toBe(true);
  expect(info.name).toBe("John Doe");
  expect(Object.keys(info).sort()).toEqual(["email", "email_verified", "family_name", "given_name", "name", "sub"]);
});

it("異常系: userinfo は ID Token を 401 で拒否する", async () => {
  const app = createApp();
  const { id_token } = await issueTokens(app);
  const res = await app.request("/oidc/userinfo", {
    headers: { authorization: `Bearer ${id_token}` },
  });
  expect(res.status).toBe(401);
});

it("異常系: userinfo は Bearer 欠如を 401 で拒否する", async () => {
  const app = createApp();
  const res = await app.request("/oidc/userinfo");
  expect(res.status).toBe(401);
});

it("正常系: logout は登録済み post_logout_redirect_uri へ 302（state 反映）", async () => {
  const app = createApp();
  const res = await logoutRequest(app, {
    post_logout_redirect_uri: "http://localhost:3000",
    state: "logout-state",
  });
  expect(res.status).toBe(302);
  const location = new URL(res.headers.get("location") ?? "");
  expect(location.origin).toBe("http://localhost:3000");
  expect(location.searchParams.get("state")).toBe("logout-state");
});

it("異常系: logout は未登録 post_logout_redirect_uri を 400 で拒否する", async () => {
  const app = createApp();
  const res = await logoutRequest(app, { post_logout_redirect_uri: "http://evil.example.com" });
  expect(res.status).toBe(400);
});

it("正常系: logout は post_logout_redirect_uri 未指定で 200 を返す", async () => {
  const app = createApp();
  const res = await logoutRequest(app, {});
  expect(res.status).toBe(200);
  expect(((await res.json()) as Record<string, string>).status).toBe("logged_out");
});

it("正常系: bypass/session が session を作成し、logout(id_token_hint) が破棄する", async () => {
  const app = createApp();
  const created = await app.request("/bypass/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ subject: "user-john-doe" }),
  });
  expect(created.status).toBe(200);
  expect(sessionStore.size).toBe(1);

  const { id_token } = await issueTokens(app);
  await logoutRequest(app, { id_token_hint: id_token });
  expect(sessionStore.size).toBe(0);
});

// bypassToken は /bypass/token で指定 subject / profile の access token を取得する。
async function bypassToken(
  app: ReturnType<typeof createApp>,
  subject: string,
  profile: string,
): Promise<string> {
  const res = await app.request("/bypass/token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ subject, profile }),
  });
  expect(res.status).toBe(200);
  return ((await res.json()) as { access_token: string }).access_token;
}

it("異常系: authorize は response_type!=code を 302 error redirect で拒否する", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "token",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: "openid",
      code_challenge: CHALLENGE,
      code_challenge_method: "S256",
      state: "st",
    }),
  );
  expect(res.status).toBe(302);
  const location = new URL(res.headers.get("location") ?? "");
  expect(location.searchParams.get("error")).toBe("unsupported_response_type");
  expect(location.searchParams.get("state")).toBe("st");
});

it("異常系: authorize は scope 欠如を 302 error redirect で拒否する", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      code_challenge: CHALLENGE,
      code_challenge_method: "S256",
    }),
  );
  expect(res.status).toBe(302);
  expect(new URL(res.headers.get("location") ?? "").searchParams.get("error")).toBe("invalid_request");
});

it("異常系: authorize は code_challenge 欠如を 302 error redirect で拒否する", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: "openid",
      code_challenge_method: "S256",
    }),
  );
  expect(res.status).toBe(302);
  expect(new URL(res.headers.get("location") ?? "").searchParams.get("error")).toBe("invalid_request");
});

it("異常系: authorize は許可外 scope を 302 invalid_scope で拒否する", async () => {
  const app = createApp();
  const res = await app.request(
    authorizePath({
      response_type: "code",
      client_id: CLIENT_ID,
      redirect_uri: REDIRECT_URI,
      scope: "openid superuser",
      code_challenge: CHALLENGE,
      code_challenge_method: "S256",
    }),
  );
  expect(res.status).toBe(302);
  expect(new URL(res.headers.get("location") ?? "").searchParams.get("error")).toBe("invalid_scope");
});

it("異常系: token は authorization_code 以外の grant_type を 400 で拒否する", async () => {
  const app = createApp();
  const location = await issueCode(app);
  const res = await tokenRequest(app, {
    grant_type: "client_credentials",
    code: location.searchParams.get("code") ?? "",
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: VERIFIER,
  });
  expect(res.status).toBe(400);
  expect(((await res.json()) as Record<string, string>).error).toBe("unsupported_grant_type");
});

it("正常系: userinfo は fixture 外 subject に sub のみ返す", async () => {
  const app = createApp();
  const token = await bypassToken(app, "unknown-subject-xyz", "valid");
  const res = await app.request("/oidc/userinfo", { headers: { authorization: `Bearer ${token}` } });
  expect(res.status).toBe(200);
  const info = (await res.json()) as Record<string, unknown>;
  expect(Object.keys(info)).toEqual(["sub"]);
  expect(info.sub).toBe("unknown-subject-xyz");
});

it("異常系: userinfo は期限切れ access token を 401 で拒否する", async () => {
  const app = createApp();
  const token = await bypassToken(app, "user-john-doe", "expired");
  const res = await app.request("/oidc/userinfo", { headers: { authorization: `Bearer ${token}` } });
  expect(res.status).toBe(401);
});

it("異常系: token は必須パラメータを欠いたリクエストを 400 で拒否する", async () => {
  const app = createApp();
  const res = await tokenRequest(app, { grant_type: "authorization_code" });
  expect(res.status).toBe(400);
  expect(((await res.json()) as Record<string, string>).error).toBe("invalid_request");
});

it("異常系: authorize は client_id が無いリクエストを 400 で拒否する", async () => {
  const app = createApp();
  const res = await app.request(authorizePath({ response_type: "code", redirect_uri: REDIRECT_URI }));
  expect(res.status).toBe(400);
  expect(((await res.json()) as Record<string, string>).error).toBe("invalid_request");
});

it("異常系: userinfo は sub が文字列でない Token を空 sub に正規化する", async () => {
  const config = loadConfig();
  const now = Math.floor(Date.now() / 1000);
  const { signingKey, kid } = keyStore.signing();
  // 発行側の Profile では作れない形（sub が数値）を、検証側の正規化を突くために直接署名して作る。
  // JWTPayload は sub を string と宣言するため、型を外して数値を載せる。
  const claims = {
    iss: config.issuer,
    sub: 12345,
    aud: config.audience,
    iat: now,
    nbf: now,
    exp: now + 300,
  } as unknown as JWTPayload;
  const token = await new SignJWT(claims).setProtectedHeader({ alg: ALG, kid, typ: "at+jwt" }).sign(signingKey);

  const res = await createApp().request("/oidc/userinfo", { headers: { authorization: `Bearer ${token}` } });
  expect(res.status).toBe(200);
  expect(await res.json()).toEqual({ sub: "" });
});

it("正常系: subject を取り出せない id_token_hint でも logout は成功する", async () => {
  const app = createApp();

  const res = await logoutRequest(app, { id_token_hint: "not-a-jwt" });

  expect(res.status).toBe(200);
});

it("正常系: state 無しの logout は redirect URI へそのまま 302 する", async () => {
  const app = createApp();

  const res = await logoutRequest(app, { post_logout_redirect_uri: "http://localhost:3000" });

  expect(res.status).toBe(302);
  expect(new URL(res.headers.get("location") ?? "").searchParams.get("state")).toBe(null);
});
