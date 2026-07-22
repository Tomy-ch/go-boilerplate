// middleware.ts は Hono のミドルウェアを提供する。
import type { MiddlewareHandler } from "hono";

// devGate は dev endpoint（/bypass ・ /admin）を保護する。無効時は 404 を返し存在を秘匿する。
export function devGate(enabled: boolean): MiddlewareHandler {
  return async (c, next) => {
    if (!enabled) {
      return c.json({ error: "not_found" }, 404);
    }
    await next();
  };
}
