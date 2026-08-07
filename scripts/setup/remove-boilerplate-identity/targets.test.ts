import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

import { ROOT_DIR } from "../lib/runtime";
import {
  BOILERPLATE_MARKER,
  BOILERPLATE_MARKER_FILES,
  BOILERPLATE_PROSE_MARKERS,
} from "./targets";

/** 宣言したファイルの実際の中身。 */
function read(relativePath: string): string {
  return fs.readFileSync(path.join(ROOT_DIR, relativePath), "utf8");
}

describe("BOILERPLATE_MARKER_FILES", () => {
  describe("正常系", () => {
    it("挙げた対象がすべて実在する", () => {
      for (const target of BOILERPLATE_MARKER_FILES) {
        expect(fs.existsSync(path.join(ROOT_DIR, target)), target).toBe(true);
      }
    });

    // 対象に挙がっていてもマーカーが 1 つも無ければ、除去は何もせず静かに終わる。
    // ファイルを移した・マーカーを消したときに気づけるのはここだけ。
    it("挙げた対象がすべてマーカーを持つ", () => {
      for (const target of BOILERPLATE_MARKER_FILES) {
        expect(read(target).includes(BOILERPLATE_MARKER), target).toBe(true);
      }
    });

    // ツール自身が消える以上、それを呼ぶ make ターゲットの宣言も一緒に落ちなければ、
    // make help に残って叩けば失敗するだけのものになる。
    it("自分を呼ぶ make ターゲットの定義元を対象に含む", () => {
      expect(BOILERPLATE_MARKER_FILES).toContain(
        ".makefiles/github/operation/setup-repository.mk",
      );
    });

    it("README の英日をどちらも対象に含む", () => {
      expect(BOILERPLATE_MARKER_FILES).toContain("README.md");
      expect(BOILERPLATE_MARKER_FILES).toContain("README.ja.md");
    });
  });

  describe("異常系", () => {
    // 生成物はマーカーを持っていても再生成で戻るため、除去の対象にしてはいけない。
    it("生成物を対象に含まない", () => {
      for (const target of BOILERPLATE_MARKER_FILES) {
        expect(target.startsWith("docs/portal/guides/"), target).toBe(false);
      }
    });
  });
});

describe("BOILERPLATE_MARKER", () => {
  describe("正常系", () => {
    it("他の除去マーカーと衝突しない名前である", () => {
      expect(BOILERPLATE_MARKER).not.toBe("sample-api");
      expect(BOILERPLATE_MARKER).not.toBe("setup-localize");
    });
  });
});

describe("BOILERPLATE_PROSE_MARKERS", () => {
  describe("正常系", () => {
    it("英日どちらの言い回しも挙げている", () => {
      expect(BOILERPLATE_PROSE_MARKERS).toContain("boilerplate");
      expect(BOILERPLATE_PROSE_MARKERS).toContain("ボイラープレート");
    });

    // 語が README に一度も現れないなら、消すものが無いか、語が変わったかのどちらか。
    it("挙げた語が実際に README へ現れる", () => {
      const readme = read("README.md") + read("README.ja.md");

      expect(BOILERPLATE_PROSE_MARKERS.some((word) => readme.includes(word))).toBe(true);
    });
  });
});
