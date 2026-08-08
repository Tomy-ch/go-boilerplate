// @vitest-environment jsdom

import { screen, waitFor } from "@testing-library/react";
import { HttpResponse, http } from "msw";
import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { mountPortal, PORTAL_LOAD_ERROR_MESSAGE } from "./mount-portal";

const docs = {
  title: "Documentation",
  subtitle: "boilerplate",
  groups: [
    {
      title: "Architecture",
      slug: "architecture",
      sections: [
        {
          id: "adr",
          slug: "adr",
          title: "ADR",
          items: [{ name: "ADR 0001", path: "./guides/0001.md", lang: "all" }],
        },
      ],
    },
  ],
};

const server = setupServer();

function createContainer(): HTMLElement {
  const container = document.createElement("div");

  document.body.append(container);

  return container;
}

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));

afterEach(() => {
  server.resetHandlers();
  vi.unstubAllGlobals();
  document.body.replaceChildren();
});

afterAll(() => server.close());

describe("mountPortal", () => {
  describe("正常系", () => {
    it("生成物を取得してビューアーを描画する", async () => {
      server.use(http.get("*/docs.json", () => HttpResponse.json(docs)));

      await mountPortal(createContainer());

      await waitFor(() =>
        expect(
          screen.getByRole("heading", { level: 1, name: "Documentation" }),
        ).toBeInTheDocument(),
      );
    });
  });

  describe("異常系", () => {
    it("取得に失敗したら読み込めなかったことと応答の状態を伝える", async () => {
      server.use(http.get("*/docs.json", () => new HttpResponse(null, { status: 500 })));

      const container = createContainer();

      await mountPortal(container);

      expect(container.textContent).toContain(PORTAL_LOAD_ERROR_MESSAGE);
      expect(container.textContent).toContain("500");
    });
    it("生成物の形が違えば読み込めなかったことを伝える", async () => {
      server.use(http.get("*/docs.json", () => HttpResponse.json({ title: "x" })));

      const container = createContainer();

      await mountPortal(container);

      expect(container.textContent).toContain(PORTAL_LOAD_ERROR_MESSAGE);
    });
    it("取得そのものが失敗した場合も原因を画面へ残す", async () => {
      server.use(http.get("*/docs.json", () => HttpResponse.error()));

      const container = createContainer();

      await mountPortal(container);

      expect(container.textContent).toContain(PORTAL_LOAD_ERROR_MESSAGE);
    });
    it("Error ではない値が投げられても原因を画面へ残す", async () => {
      // ここだけ fetch を直接差し替える。Error でない値が投げられる状況は HTTP の応答では
      // 作れず、MSW では再現できない。防御的な分岐そのものを行使するための例外。
      vi.stubGlobal(
        "fetch",
        vi.fn(() => Promise.reject("想定外")),
      );

      const container = createContainer();

      await mountPortal(container);

      expect(container.textContent).toContain("想定外");
    });
  });
});
