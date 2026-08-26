import { describe, expect, it } from "vitest";

import { hasPremisePhrase, inspect, isChecked, survivingText } from "./rules";

describe("survivingText", () => {
  describe("正常系", () => {
    it("boilerplate-only の領域をマーカーごと落とす", () => {
      const content = ["keep", "<!-- boilerplate-only:begin -->", "drop", "<!-- boilerplate-only:end -->", "keep2"].join("\n");

      expect(survivingText(content)).toBe("keep\nkeep2");
    });

    it("sample-api の領域も同じく落とす", () => {
      const content = ["keep", "// sample-api:begin", "drop", "// sample-api:end"].join("\n");

      expect(survivingText(content)).toBe("keep");
    });

    it("行マーカーの行だけを落とす", () => {
      expect(survivingText("keep\ndrop <!-- boilerplate-only:line -->\nkeep2")).toBe("keep\nkeep2");
    });

    it("入れ子の領域を深さで数える", () => {
      const content = [
        "keep",
        "# sample-api:begin",
        "# sample-api:begin",
        "drop",
        "# sample-api:end",
        "drop2",
        "# sample-api:end",
        "keep2",
      ].join("\n");

      expect(survivingText(content)).toBe("keep\nkeep2");
    });

    it("replace の上流側を落とし退避側をアンコメントして残す", () => {
      const content = [
        "<!-- boilerplate-only:replace-begin -->",
        "upstream only",
        "<!-- boilerplate-only:replace-with -->",
        "<!-- = general form -->",
        "<!-- boilerplate-only:replace-end -->",
      ].join("\n");

      expect(survivingText(content)).toBe("general form");
    });

    it("`//` と `#` の退避コメントもアンコメントする", () => {
      const content = [
        "# sample-api:replace-begin",
        "active()",
        "# sample-api:replace-with",
        "  // = substitute()",
        "# sample-api:replace-end",
      ].join("\n");

      expect(survivingText(content)).toBe("substitute()");
    });

    it("マーカーが無ければ本文をそのまま返す", () => {
      expect(survivingText("a\nb\nc")).toBe("a\nb\nc");
    });
  });

  describe("異常系", () => {
    // 落としすぎる側の失敗は無言である。検査は「0 件」と報告しながら何も見ていない状態になり、
    // この検査が塞ごうとしている失敗と同じ形になる。以下はその向きを塞ぐ。

    // 退避側は 作成先へ残る本文なので、ここに前提を書いたら検出できなければならない。
    it("退避側に書かれた前提を検査へ通す", () => {
      const content = [
        "<!-- boilerplate-only:replace-begin -->",
        "upstream",
        "<!-- boilerplate-only:replace-with -->",
        "<!-- = This template is maintained by its own team. -->",
        "<!-- boilerplate-only:replace-end -->",
      ].join("\n");

      expect(hasPremisePhrase(survivingText(content))).toBe(true);
    });

    it("退避側の形をしていない行も落とさない", () => {
      const content = [
        "# sample-api:replace-begin",
        "# sample-api:replace-with",
        "raw line",
        "# sample-api:replace-end",
      ].join("\n");

      expect(survivingText(content)).toBe("raw line");
    });

    // 対応の取れない end で depth が負に振れると、以降の本文がすべて落ちる。
    it("対応の取れない end で以降を巻き込まない", () => {
      expect(survivingText("<!-- boilerplate-only:end -->\nkeep")).toBe("keep");
    });

    // コメント記号を必須にしないと、マーカー規約を説明している散文を境界と誤認して本文を落とす。
    it("コメント記号の無い同名トークンを境界と見なさない", () => {
      const content = 'grep -rn "boilerplate-only:begin" docs/\nkeep';

      expect(survivingText(content)).toBe(content);
    });

    it("別名のマーカーには反応しない", () => {
      const content = "keep\n<!-- setup-localize:begin -->\nkeep2\n<!-- setup-localize:end -->";

      expect(survivingText(content)).toBe(content);
    });
  });
});

describe("isChecked", () => {
  describe("正常系", () => {
    it("テンプレート作成後も残る文書を対象にする", () => {
      expect(isChecked("docs/adr/0001-avoid-lock-in.md")).toBe(true);
      expect(isChecked("docs/rules.md")).toBe(true);
      expect(isChecked("docs/design/worker.md")).toBe(true);
      expect(isChecked(".makefiles/README.md")).toBe(true);
    });

    it("層 README を対象にする", () => {
      expect(isChecked("internal/domain/README.md")).toBe(true);
      expect(isChecked("pkg/uuid/README.ja.md")).toBe(true);
    });

    // ミラーは正本と同じ内容を運ぶので、正本が対象ならミラーも対象。
    it("日本語ミラーを正本と同じ扱いにする", () => {
      expect(isChecked("docs/adr/0001-avoid-lock-in.ja.md")).toBe(true);
      expect(isChecked("docs/rules.ja.md")).toBe(true);
    });
  });

  describe("異常系", () => {
    // 許可域は前提と一緒に消えるので、前提が残っても嘘にならない。
    it("許可域を対象から外す", () => {
      expect(isChecked("README.md")).toBe(false);
      expect(isChecked("README.ja.md")).toBe(false);
      expect(isChecked("docs/get-started/setup-repository.md")).toBe(false);
      expect(isChecked("docs/get-started/setup-repository.ja.md")).toBe(false);
    });

    it("Markdown 以外を対象にしない", () => {
      expect(isChecked("internal/domain/user/user_domain.go")).toBe(false);
      expect(isChecked("docs/portal/manifest.yaml")).toBe(false);
    });

    it("挙げていない領域を対象にしない", () => {
      expect(isChecked("docs/deployment/github-page.md")).toBe(false);
      expect(isChecked("scripts/README.md")).toBe(false);
    });
  });
});

describe("hasPremisePhrase", () => {
  describe("正常系", () => {
    it("自己参照の言い回しを検出する", () => {
      expect(hasPremisePhrase("This template targets backend services.")).toBe(true);
      expect(hasPremisePhrase("The scaffold provides the runner.")).toBe(true);
      expect(hasPremisePhrase("Adopters must configure rate limiting.")).toBe(true);
      expect(hasPremisePhrase("A template-derived repository starts disabled.")).toBe(true);
      expect(hasPremisePhrase("このテンプレートは Onion Architecture を前提とする")).toBe(true);
      expect(hasPremisePhrase("本テンプレートは `http.Server` を自前で持つ")).toBe(true);
    });
  });

  describe("異常系", () => {
    // 裸の名詞で落とすと 381 件が当たり、その大半が別語義だった。判定は自己参照の形に限る。
    it("裸の名詞だけでは検出しない", () => {
      expect(hasPremisePhrase("More boilerplate than an ORM's convenience methods.")).toBe(false);
      expect(hasPremisePhrase("Downstream consumers observe eventual consistency.")).toBe(false);
      expect(hasPremisePhrase("Generated by the `scaffold-domain` skill.")).toBe(false);
      expect(hasPremisePhrase("Rendered with Go `text/template`.")).toBe(false);
    });

    it("上流サービスの意味の upstream を検出しない", () => {
      expect(hasPremisePhrase("The upstream HTTP service returns 503.")).toBe(false);
    });
  });
});

describe("inspect", () => {
  describe("正常系", () => {
    it("該当行を行番号付きで返す", () => {
      const content = ["# Title", "", "This template targets X.", "Normal line."].join("\n");

      expect(inspect("docs/rules.md", content, [])).toEqual([
        { file: "docs/rules.md", line: 3, text: "This template targets X." },
      ]);
    });

    it("該当が無ければ空を返す", () => {
      expect(inspect("docs/rules.md", "# Title\n\nNormal.", [])).toEqual([]);
    });

    it("宣言された行を除外する", () => {
      const content = "CREATE DATABASE ... TEMPLATE template1 while the template is stale";
      const allowances = [
        { file: "a/README.md", contains: "TEMPLATE template1", reason: "PostgreSQL の template DB" },
      ];

      expect(inspect("a/README.md", content, allowances)).toEqual([]);
    });
  });

  describe("異常系", () => {
    // 宣言はファイル単位で効く。別ファイルの同文を巻き添えで許すと、検査に穴が開く。
    it("別ファイルの宣言では除外しない", () => {
      const content = "This template targets X.";
      const allowances = [{ file: "other.md", contains: "This template", reason: "別物" }];

      expect(inspect("docs/rules.md", content, allowances)).toHaveLength(1);
    });
  });
});
