// admin.ts は運用補助（/admin/*）を提供する。dev-gate 配下でのみ有効。
import { Hono } from "hono";
import { users } from "../users.ts";

export const adminRoutes = new Hono();

// GET /admin/users は固定 User Fixture の一覧を返す。
adminRoutes.get("/admin/users", (c) => c.json({ users }));

// POST /admin/reset ・ POST /admin/keys/rotate は後続 Increment で実装する（契約は openapi/ に定義済み）。
adminRoutes.post("/admin/reset", (c) => c.json({ error: "not_implemented" }, 501));
adminRoutes.post("/admin/keys/rotate", (c) => c.json({ error: "not_implemented" }, 501));
