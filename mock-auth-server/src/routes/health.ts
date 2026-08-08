// health.ts は /health（liveness）を提供する。
import { Hono } from "hono";

export const healthRoutes = new Hono();

healthRoutes.get("/health", (c) => c.json({ status: "ok" }));
