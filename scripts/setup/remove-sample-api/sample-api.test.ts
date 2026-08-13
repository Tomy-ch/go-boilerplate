import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

import { ROOT_DIR } from "../lib/runtime";
import {
  SETUP_SHARED_DIR,
  SETUP_SHARED_DIR_USERS,
  SETUP_VERIFIER_DIR,
  containsSampleMarker,
  isScanTarget,
  isWithinRoot,
  sharedModuleTargets,
  stripSampleMarkers,
} from "./sample-api";
import { EXCLUDED_PATH_PREFIXES, MARKER_LITERAL_FILES } from "./sample-manifest";

describe("stripSampleMarkers", () => {
  describe("正常系", () => {
    it("begin〜end のブロックをマーカーごと取り除く", () => {
      const content = [
        "package module",
        "// sample-api:begin",
        "var sample = 1",
        "// sample-api:end",
        "var core = 2",
      ].join("\n");

      expect(stripSampleMarkers(content)).toEqual({
        content: "package module\nvar core = 2",
        removed: 3,
      });
    });

    it("行末の sample-api:line を持つ行だけを取り除く", () => {
      const content = "keep\nsample() // sample-api:line\nkeep2";

      expect(stripSampleMarkers(content)).toEqual({ content: "keep\nkeep2", removed: 1 });
    });

    it("# と <!-- のコメント記号でもマーカーとして扱う", () => {
      const content = ["a: 1 # sample-api:line", "<!-- sample-api:begin -->", "b", "<!-- sample-api:end -->", "c"].join(
        "\n",
      );

      expect(stripSampleMarkers(content).content).toBe("c");
    });

    it("入れ子のブロックを深さで数えて取り除く", () => {
      const content = [
        "keep",
        "// sample-api:begin",
        "outer",
        "// sample-api:begin",
        "inner",
        "// sample-api:end",
        "still-inside",
        "// sample-api:end",
        "keep2",
      ].join("\n");

      expect(stripSampleMarkers(content).content).toBe("keep\nkeep2");
    });

    it("replace ブロックは有効側を落として退避側をアンコメントする", () => {
      const content = [
        "// sample-api:replace-begin",
        "resolver := userIdentityResolver()",
        "// sample-api:replace-with",
        "// = resolver := passthroughResolver()",
        "// sample-api:replace-end",
      ].join("\n");

      expect(stripSampleMarkers(content).content).toBe("resolver := passthroughResolver()");
    });

    // 退避コードは Go のインデント（タブ）を含む。剥がすのは `=` 直後の空白 1 つだけで、
    // それ以上剥がすと gofmt の効かないテストコードとして復元される。
    it("退避コード側のタブ字下げを保つ", () => {
      const content = [
        "\t// sample-api:replace-begin",
        "\tactive()",
        "\t// sample-api:replace-with",
        "\t// = \tt.Parallel()",
        "\t// sample-api:replace-end",
      ].join("\n");

      expect(stripSampleMarkers(content).content).toBe("\t\tt.Parallel()");
    });

    it("退避コメントのインデントを保ったままアンコメントする", () => {
      const content = [
        "  # sample-api:replace-begin",
        "  active: true",
        "  # sample-api:replace-with",
        "  # = active: false",
        "  # sample-api:replace-end",
      ].join("\n");

      expect(stripSampleMarkers(content).content).toBe("  active: false");
    });

    it("マーカーが無ければ本文をそのまま返す", () => {
      const content = "package module\nvar core = 1\n";

      expect(stripSampleMarkers(content)).toEqual({ content, removed: 0 });
    });
  });

  describe("異常系", () => {
    // コメント記号を必須にしないと、マーカー名に言及した文字列リテラルや
    // ドキュメント本文（この規約を説明している README）を境界と誤認して本文を落とす。
    it("コメント記号の無い同名トークンをマーカーと見なさない", () => {
      const content = 'const marker = "sample-api:begin"\nkeep';

      expect(stripSampleMarkers(content)).toEqual({ content, removed: 0 });
    });

    // 対応の取れないマーカーを黙って通すと、begin だけ残ったファイルが
    // 「除去済み」として扱われ、サンプルコードが本番に残る。
    it("閉じられていない begin を throw する", () => {
      expect(() => stripSampleMarkers("// sample-api:begin\nsample")).toThrow(
        "sample-api:end が見つかりません",
      );
    });

    it("対応する begin の無い end を throw する", () => {
      expect(() => stripSampleMarkers("keep\n// sample-api:end")).toThrow(
        "sample-api:begin が見つかりません",
      );
    });

    it("入れ子の replace ブロックを throw する", () => {
      const content = ["// sample-api:replace-begin", "// sample-api:replace-begin"].join("\n");

      expect(() => stripSampleMarkers(content)).toThrow("入れ子にできません");
    });

    it("replace-begin の無い replace-with を throw する", () => {
      expect(() => stripSampleMarkers("// sample-api:replace-with")).toThrow(
        "sample-api:replace-with に対応する",
      );
    });

    it("replace-begin の無い replace-end を throw する", () => {
      expect(() => stripSampleMarkers("// sample-api:replace-end")).toThrow(
        "sample-api:replace-end に対応する",
      );
    });

    it("閉じられていない replace-begin を throw する", () => {
      const content = ["// sample-api:replace-begin", "active", "// sample-api:replace-with"].join(
        "\n",
      );

      expect(() => stripSampleMarkers(content)).toThrow("replace-end が見つかりません");
    });

    // 退避コメントでない行をそのまま残すと、削除後にしか有効化しないはずの
    // コードと有効側のコードが両方残り、コンパイルが通らなくなる。
    it("退避コメントで始まらない差し替え行を throw する", () => {
      const content = [
        "// sample-api:replace-begin",
        "active",
        "// sample-api:replace-with",
        "resolver := passthroughResolver()",
        "// sample-api:replace-end",
      ].join("\n");

      expect(() => stripSampleMarkers(content)).toThrow("のいずれかで書いてください");
    });
  });
});

describe("isWithinRoot", () => {
  describe("正常系", () => {
    it("ROOT_DIR 配下のパスを内側と判定する", () => {
      expect(isWithinRoot("/repo/internal/user", "/repo", "/")).toBe(true);
    });

    it("ROOT_DIR が区切り文字で終わっていても同じ判定になる", () => {
      expect(isWithinRoot("/repo/internal", "/repo/", "/")).toBe(true);
    });

    it("直下のファイルも内側と判定する", () => {
      expect(isWithinRoot("/repo/a.md", "/repo", "/")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("ROOT_DIR 自体は削除対象として認めない", () => {
      expect(isWithinRoot("/repo", "/repo", "/")).toBe(false);
    });

    it("接頭辞が一致するだけの別ディレクトリを内側と誤判定しない", () => {
      expect(isWithinRoot("/repo-backup/internal", "/repo", "/")).toBe(false);
    });

    it("ROOT_DIR の外を指すパスを拒否する", () => {
      expect(isWithinRoot("/etc/passwd", "/repo", "/")).toBe(false);
    });

    it("親へ抜けたパスを拒否する", () => {
      expect(isWithinRoot("/", "/repo", "/")).toBe(false);
    });
  });
});

describe("sharedModuleTargets", () => {
  describe("正常系", () => {
    it("共有モジュールを使う他のツールが残っていれば消さない", () => {
      expect(sharedModuleTargets(true)).toEqual([]);
    });

    it("使う側が全て消えていれば共有モジュールを道連れにする", () => {
      expect(sharedModuleTargets(false)).toEqual([SETUP_SHARED_DIR]);
    });
  });
});

describe("SETUP_SHARED_DIR_USERS", () => {
  describe("正常系", () => {
    // 挙げ漏らしたツールは、先にサンプル削除を実行した利用者の手元で共有モジュールごと壊れる。
    // 壊れ方が「まだ実行していない手順が実行できない」なので、その手順に来るまで気づけない。
    // 初期化ツール群（Phase 5）は検証器と一緒に消えるため、検証器 1 つで代表できる。
    it("サンプル削除と独立して残りうるツールを挙げている", () => {
      expect(SETUP_SHARED_DIR_USERS).toContain(SETUP_VERIFIER_DIR);
    });

    it("挙げたツールが実在する", () => {
      for (const user of SETUP_SHARED_DIR_USERS) {
        const dir = path.join(ROOT_DIR, "scripts/setup", user);

        expect(fs.existsSync(dir), user).toBe(true);
      }
    });
  });
});

describe("isScanTarget", () => {
  describe("正常系", () => {
    it("除外に当たらないパスを対象にする", () => {
      expect(isScanTarget("internal/di/module/job.go")).toBe(true);
      expect(isScanTarget("docs/rules.md")).toBe(true);
    });

    it("Windows 形式の区切りでも除外を判定する", () => {
      expect(isScanTarget("docs\\portal\\guides\\index.md")).toBe(false);
    });
  });

  describe("異常系", () => {
    // 生成物はマーカーを持っていても再生成で戻るため、除去の対象にしてはいけない。
    it("生成物の接頭辞を対象から外す", () => {
      for (const prefix of EXCLUDED_PATH_PREFIXES) {
        expect(isScanTarget(`${prefix}anything.md`), prefix).toBe(false);
      }
    });

    it("接頭辞が途中まで一致するだけのパスは外さない", () => {
      expect(isScanTarget("docs/portal/manifest.yaml")).toBe(true);
    });
  });
});

describe("containsSampleMarker", () => {
  describe("正常系", () => {
    it("コメント記号を伴うマーカーを検出する", () => {
      expect(containsSampleMarker("<!-- sample-api:begin -->")).toBe(true);
      expect(containsSampleMarker("x // sample-api:line")).toBe(true);
    });
  });

  describe("異常系", () => {
    // コメント記号を必須にしないと、規約を説明している散文を境界と誤認する。
    it("コメント記号の無い同名トークンを検出しない", () => {
      expect(containsSampleMarker('const marker = "sample-api:begin"')).toBe(false);
    });
  });
});
