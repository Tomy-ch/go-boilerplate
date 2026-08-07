import { describe, expect, it } from "vitest";

import { stripSampleMarkers, isWithinRoot } from "./sample-api";

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

      expect(() => stripSampleMarkers(content)).toThrow("//= または #= で始めてください");
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
