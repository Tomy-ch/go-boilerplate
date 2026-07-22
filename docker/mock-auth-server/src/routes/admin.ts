// admin.ts は運用補助（/admin/*）を提供する。dev-gate 配下でのみ有効。
import { Hono } from "hono";
import { users } from "../users.ts";
import { resetAll } from "../store.ts";
import { logEvent } from "../log.ts";

export const adminRoutes = new Hono();

// GET /admin/users は固定 User Fixture の一覧を返す。
adminRoutes.get("/admin/users", (c) => c.json({ users }));

// POST /admin/reset は揮発ストア（code / session）を初期化する。fixture は対象外（再起動で初期化）。
adminRoutes.post("/admin/reset", (c) => {
  resetAll();
  logEvent("admin_reset", {});
  return c.json({ status: "reset" });
});

// POST /admin/keys/rotate は契約のみ（実ローテは後続 PBI）。定義済み・未実装として 501 を返す。
adminRoutes.post("/admin/keys/rotate", (c) =>
  c.json({ error: "not_implemented", error_description: "key rotation is implemented in a later PBI" }, 501),
);
