// server.ts は疑似 OIDC 認証サーバーの HTTP エントリポイント（Hono を @hono/node-server で起動する薄い層）。
// ルーティングは router.ts / routes/* に、状態は store.ts に分離している。
import { serve } from "@hono/node-server";
import { createApp } from "./router.ts";
import { config } from "./config.ts";
import { users } from "./users.ts";
import { sweepAll } from "./store.ts";

const app = createApp();

serve({ fetch: app.fetch, port: config.port }, (info) => {
  console.log(
    JSON.stringify({
      msg: "mock-auth-server started",
      issuer: config.issuer,
      audience: config.audience,
      port: info.port,
      users: users.length,
      dev_endpoints: config.devEndpointsEnabled ? "enabled" : "disabled",
    }),
  );
});

// 期限切れエントリの定期回収（参照時の lazy 失効に加えたメモリ回収）。プロセス終了は妨げない。
setInterval(sweepAll, 60_000).unref();
