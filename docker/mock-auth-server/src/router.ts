// router.ts は Hono アプリを組み立てる。ルートは routes/* に分割し、/bypass ・ /admin は dev-gate 配下に置く。
import { Hono } from "hono";
import { config } from "./config.ts";
import { devGate } from "./middleware.ts";
import { healthRoutes } from "./routes/health.ts";
import { wellKnownRoutes } from "./routes/wellKnown.ts";
import { oidcRoutes } from "./routes/oidc.ts";
import { bypassRoutes } from "./routes/bypass.ts";
import { adminRoutes } from "./routes/admin.ts";

// createApp は Hono アプリを構築する。devEndpointsEnabled はテストで上書きできる（既定は config 値）。
export function createApp(opts: { devEndpointsEnabled?: boolean } = {}): Hono {
  const devEnabled = opts.devEndpointsEnabled ?? config.devEndpointsEnabled;
  const app = new Hono();

  app.route("/", healthRoutes);
  app.route("/", wellKnownRoutes);
  app.route("/", oidcRoutes);

  // /bypass ・ /admin は dev-gate 配下（無効時は 404 で存在を秘匿）。
  // Hono は use をルート登録より前に置かないとそのルートに適用されないため、route の前に use する。
  app.use("/bypass/*", devGate(devEnabled));
  app.use("/admin/*", devGate(devEnabled));
  app.route("/", bypassRoutes);
  app.route("/", adminRoutes);

  app.notFound((c) => c.json({ error: "not_found" }, 404));
  return app;
}
