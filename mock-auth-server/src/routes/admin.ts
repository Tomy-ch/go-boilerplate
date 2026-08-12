// admin.ts は運用補助（/admin/*）を提供する。dev-gate 配下でのみ有効。
import { Hono } from "hono";
import { users } from "../users.ts";
import { resetAll } from "../store.ts";
import { keyStore, RotationError } from "../keys.ts";
import { adminRotateKeysBody } from "../generated/schemas.ts";
import { logEvent } from "../log.ts";

export const adminRoutes = new Hono();

// rotationFailure は RotationError を 400 応答ボディへ写像し、それ以外は握り潰さず再送出する。
export function rotationFailure(err: unknown): { error: string; error_description: string } {
  if (err instanceof RotationError) {
    return { error: "invalid_request", error_description: err.message };
  }
  throw err;
}

adminRoutes.get("/admin/users", (c) => c.json({ users }));

// POST /admin/reset は揮発ストアと鍵ストアを Phase1 へ戻す。fixture は対象外（詳細は README）。
adminRoutes.post("/admin/reset", (c) => {
  resetAll();
  keyStore.reset();
  logEvent("admin_reset", {});
  return c.json({ status: "reset" });
});

// POST /admin/keys/rotate は宣言的操作で鍵状態を遷移し、結果を返す。不正な操作は 400（詳細は README）。
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
    return c.json(rotationFailure(err), 400);
  }

  const state = keyStore.state();
  logEvent("key_rotation", { action, kid, ...state });
  return c.json(state);
});
