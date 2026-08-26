import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { parseDocsJson } from "../docs-json/docs-json";
import { PortalApp } from "../portal-app/portal-app";

/** 生成物が届かない・形が違う場合に画面へ残す文言。 */
export const PORTAL_LOAD_ERROR_MESSAGE = "ドキュメントを読み込めませんでした。";

function describe(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * 生成物 `docs.json` を取得してビューアーをマウントする。
 *
 * @remarks
 * 取得と検証の失敗は配信事故であり、利用者が操作で回復できる状態ではありません。原因は画面へ
 * 出します。静的配信されるためログの送り先を持たず、壊れた画面を見ている人がそのまま原因を
 * 追える形にしておかないと、失敗が誰にも届かないためです。
 */
export async function mountPortal(container: HTMLElement): Promise<void> {
  try {
    const response = await fetch("./docs.json");

    if (!response.ok) {
      throw new Error(`docs.json を取得できませんでした: ${response.status}`);
    }

    const docs = parseDocsJson(await response.json());

    createRoot(container).render(
      <StrictMode>
        <PortalApp docs={docs} />
      </StrictMode>,
    );
  } catch (error) {
    container.textContent = `${PORTAL_LOAD_ERROR_MESSAGE} ${describe(error)}`;
  }
}
