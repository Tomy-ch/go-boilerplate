// bypass.ts はテスト用ショートカット（/bypass/*）を提供する。dev-gate 配下でのみ有効。
import { Hono } from "hono";
import { config } from "../config.ts";
import { defaultSubject } from "../users.ts";
import { issueToken, PROFILES, ACCESS_TTL_SECONDS } from "../tokens.ts";

export const bypassRoutes = new Hono();

// POST /bypass/token は subject と固定 Profile から access token を発行する。
// subject 省略時は defaultSubject、profile 省略時は "valid"。未知の profile は 400。
bypassRoutes.post("/bypass/token", async (c) => {
  const raw = await c.req.text();
  let body: Record<string, unknown> = {};
  if (raw.trim() !== "") {
    try {
      body = JSON.parse(raw) as Record<string, unknown>;
    } catch {
      return c.json({ error: "invalid_request", error_description: "body must be valid JSON" }, 400);
    }
  }

  const subject = typeof body.subject === "string" ? body.subject : defaultSubject;
  const profile = typeof body.profile === "string" ? body.profile : "valid";
  if (!PROFILES.includes(profile)) {
    return c.json({ error: "invalid_request", error_description: `unknown profile: ${profile}` }, 400);
  }

  const accessToken = await issueToken(config, subject, profile);
  return c.json({ access_token: accessToken, token_type: "Bearer", expires_in: ACCESS_TTL_SECONDS });
});

// POST /bypass/session は後続 Increment で実装する（契約は openapi/ に定義済み）。
bypassRoutes.post("/bypass/session", (c) => c.json({ error: "not_implemented" }, 501));
