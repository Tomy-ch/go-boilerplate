import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

import { ROOT_DIR } from "../lib/runtime";
import {
  ACTION_PIN_LOCKFILE,
  DAST_ACTION_PIN_KEY,
  DAST_MARKER,
  DAST_MARKER_FILES,
  DAST_PATHS,
  isRemovablePath,
  stripActionPin,
} from "./dast-targets";

/** 宣言したファイルの実際の中身。 */
function read(relativePath: string): string {
  return fs.readFileSync(path.join(ROOT_DIR, relativePath), "utf8");
}

describe("DAST_PATHS", () => {
  describe("正常系", () => {
    it("挙げた対象がすべて実在する", () => {
      for (const target of DAST_PATHS) {
        expect(fs.existsSync(path.join(ROOT_DIR, target)), target).toBe(true);
      }
    });

    // ツールが残ったままなら二度目の実行は何も消せず、利用者のリポジトリに居座るだけになる。
    it("撤去ツール自身を対象に含む", () => {
      expect(DAST_PATHS).toContain("scripts/setup/remove-dast-setting");
    });

    // ワークフローが残れば毎週スキャンが動き続ける。撤去の本体はここ。
    it("ワークフロー本体と ZAP のルールファイルを対象に含む", () => {
      expect(DAST_PATHS).toContain(".github/workflows/dast.yaml");
      expect(DAST_PATHS).toContain(".github/zap");
    });
  });

  describe("異常系", () => {
    it("受け付けられないパスを含まない", () => {
      for (const target of DAST_PATHS) {
        expect(isRemovablePath(target), target).toBe(true);
      }
    });
  });
});

describe("DAST_MARKER_FILES", () => {
  describe("正常系", () => {
    it("挙げた対象がすべて実在する", () => {
      for (const target of DAST_MARKER_FILES) {
        expect(fs.existsSync(path.join(ROOT_DIR, target)), target).toBe(true);
      }
    });

    // 対象に挙がっていてもマーカーが 1 つも無ければ、除去は何もせず静かに終わる。
    // ファイルを移した・マーカーを消したときに気づけるのはここだけ。
    it("挙げた対象がすべてマーカーを持つ", () => {
      for (const target of DAST_MARKER_FILES) {
        expect(read(target).includes(`${DAST_MARKER}:`), target).toBe(true);
      }
    });

    it("ワークフロー一覧とセットアップ手順を英日どちらも対象に含む", () => {
      expect(DAST_MARKER_FILES).toContain(".github/workflows/README.md");
      expect(DAST_MARKER_FILES).toContain(".github/workflows/README.ja.md");
      expect(DAST_MARKER_FILES).toContain("docs/get-started/setup-repository.md");
      expect(DAST_MARKER_FILES).toContain("docs/ja/get-started/setup-repository.ja.md");
    });

    // ツール自身が消える以上、それを叩く CI 定義も一緒に落ちなければ、実行できないものを
    // 実行しようとする検査だけが残る。
    it("自分を叩く CI 定義を対象に含む", () => {
      expect(DAST_MARKER_FILES).toContain(".github/workflows/setup-scripts-check.yaml");
    });
  });

  describe("異常系", () => {
    // 削除するファイル自身にマーカー除去をかけても意味が無く、順序に依存した二重管理になる。
    it("丸ごと削除するパスと重複しない", () => {
      for (const target of DAST_MARKER_FILES) {
        expect(DAST_PATHS.some((removed) => target.startsWith(removed)), target).toBe(false);
      }
    });
  });
});

describe("DAST_MARKER", () => {
  describe("正常系", () => {
    it("他の除去マーカーと衝突しない名前である", () => {
      expect(DAST_MARKER).not.toBe("sample-api");
      expect(DAST_MARKER).not.toBe("boilerplate");
      expect(DAST_MARKER).not.toBe("setup-localize");
    });
  });
});

describe("stripActionPin", () => {
  describe("正常系", () => {
    it("指定 action のエントリ行だけを落とす", () => {
      const content = [
        '"actions/checkout@v7.0.0" = "aaa"',
        '"zaproxy/action-api-scan@v0.10.0" = "bbb"',
        '"step-security/harden-runner@v2.20.0" = "ccc"',
      ].join("\n");

      expect(stripActionPin(content, "zaproxy/action-api-scan")).toBe(
        '"actions/checkout@v7.0.0" = "aaa"\n"step-security/harden-runner@v2.20.0" = "ccc"',
      );
    });

    // 撤去後に参照の消えたエントリが残ると pin-actions-check が落ちる。実在するキーを
    // 相手にしていることは、lockfile の実物と突き合わせて初めて確かめられる。
    it("宣言したキーが実際の lockfile に存在する", () => {
      const lockfile = read(ACTION_PIN_LOCKFILE);

      expect(stripActionPin(lockfile, DAST_ACTION_PIN_KEY)).not.toBeNull();
    });
  });

  describe("異常系", () => {
    it("該当が無ければ書き換え不要として null を返す", () => {
      expect(stripActionPin('"actions/checkout@v7.0.0" = "aaa"', "zaproxy/action-api-scan")).toBeNull();
    });

    // 接頭辞が一致するだけの別 action を巻き込むと、無関係な参照が固定を失う。
    it("キーが接頭辞として一致するだけの別エントリを落とさない", () => {
      const content = '"zaproxy/action-api-scan-extra@v1" = "aaa"';

      expect(stripActionPin(content, "zaproxy/action-api-scan")).toBeNull();
    });
  });
});

describe("isRemovablePath", () => {
  describe("正常系", () => {
    it("リポジトリ相対のパスを受け付ける", () => {
      expect(isRemovablePath(".github/workflows/dast.yaml")).toBe(true);
      expect(isRemovablePath(".github/zap")).toBe(true);
    });

    // `..` を名前の一部に含むだけのファイルは、親を辿らないので受け付けてよい。
    it("親を辿らない `..` を含む名前は受け付ける", () => {
      expect(isRemovablePath("docs/a..b.md")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("空文字を受け付けない", () => {
      expect(isRemovablePath("")).toBe(false);
    });

    it("絶対パスを受け付けない", () => {
      expect(isRemovablePath("/etc/passwd")).toBe(false);
    });

    it("親を辿るパスを受け付けない", () => {
      expect(isRemovablePath("../outside")).toBe(false);
      expect(isRemovablePath(".github/../../outside")).toBe(false);
    });
  });
});
