// admin.ts は運用補助（/admin/*）を提供する。dev-gate 配下でのみ有効。
import { Hono } from "hono";
import { users } from "../users.ts";
import { resetAll } from "../store.ts";
import { keyStore, RotationError } from "../keys.ts";
import { adminRotateKeysBody } from "../generated/schemas.ts";
import { logEvent } from "../log.ts";

export const adminRoutes = new Hono();

// GET /admin/users は固定 User Fixture の一覧を返す。
adminRoutes.get("/admin/users", (c) => c.json({ users }));

// POST /admin/reset は揮発ストア（code / session）と鍵ストア（Phase1）を初期化する。
// fixture は対象外（再起動で初期化）。鍵素材は固定のまま、公開集合・署名鍵の状態のみ Phase1 へ戻す。
adminRoutes.post("/admin/reset", (c) => {
  resetAll();
  keyStore.reset();
  logEvent("admin_reset", {});
  return c.json({ status: "reset" });
});

// POST /admin/keys/rotate は宣言的操作（add-key / promote / retire）で署名鍵状態を遷移する。
// 操作後の鍵状態を返す。不正な操作（未知 / 退役不能 kid・不正遷移）は 400。
adminRoutes.post("/admin/keys/rotate", async (c) => {
  const raw = await c.req.text();
  let json: unknown = {};
  if (raw.trim() !== "") {
    try {
      json = JSON.parse(raw);
    } catch {
      return c.json({ error: "invalid_request", error_description: "body must be valid JSON" }, 400);
    }
  }
  const parsed = adminRotateKeysBody.safeParse(json);
  if (!parsed.success) {
    return c.json({ error: "invalid_request", error_description: "action and kid are required" }, 400);
  }
  const { action, kid } = parsed.data;

  try {
    switch (action) {
      case "add-key":
        keyStore.addKey(kid);
        break;
      case "promote":
        keyStore.promote(kid);
        break;
      case "retire":
        keyStore.retire(kid);
        break;
    }
  } catch (err) {
    if (err instanceof RotationError) {
      return c.json({ error: "invalid_request", error_description: err.message }, 400);
    }
    throw err;
  }

  const state = keyStore.state();
  logEvent("key_rotation", { action, kid, ...state });
  return c.json(state);
});
