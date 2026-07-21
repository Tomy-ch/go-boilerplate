// oidc.ts は OAuth2 / OIDC 標準エンドポイント（/oidc/*）を提供する。
// authorize / token（PKCE 中核）・userinfo / logout は後続 Increment で実装する。
// 契約は openapi/ に定義済みのため、未実装の間は 501 を返す（404 ではなく「定義済み・未実装」を表す）。
import { Hono } from "hono";

export const oidcRoutes = new Hono();

oidcRoutes.get("/oidc/authorize", (c) => c.json({ error: "not_implemented" }, 501));
oidcRoutes.post("/oidc/token", (c) => c.json({ error: "not_implemented" }, 501));
oidcRoutes.get("/oidc/userinfo", (c) => c.json({ error: "not_implemented" }, 501));
oidcRoutes.post("/oidc/logout", (c) => c.json({ error: "not_implemented" }, 501));
