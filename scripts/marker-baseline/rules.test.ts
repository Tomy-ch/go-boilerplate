import { describe, expect, it } from "vitest";

import {
  EXCLUDED_PATH_PREFIXES,
  countMarkerLines,
  diffBaseline,
  isBaselineTarget,
} from "./rules";

describe("countMarkerLines", () => {
  describe("正常系", () => {
    it("両名前空間の全接尾辞を数える", () => {
      const content = [
        "<!-- boilerplate-only:begin -->",
        "<!-- boilerplate-only:end -->",
        "x // sample-api:line",
        "# sample-api:replace-begin",
        "# sample-api:replace-with",
        "# sample-api:replace-end",
      ].join("\n");

      expect(countMarkerLines(content)).toBe(6);
    });

    it("コードフェンスの中でも数える", () => {
      const content = ["```go", "usercount.New, // sample-api:line", "```"].join("\n");

      expect(countMarkerLines(content)).toBe(1);
    });

    it("マーカーが無ければ 0", () => {
      expect(countMarkerLines("# Title\n\nNormal prose.")).toBe(0);
    });
  });

  describe("異常系", () => {
    // コメント記号を必須にしないと、規約を説明している散文を数えて差分が揺れる。
    it("コメント記号の無い同名トークンを数えない", () => {
      expect(countMarkerLines('grep -rn "sample-api:begin" docs/')).toBe(0);
      expect(countMarkerLines("| `boilerplate-only:line` | 行末コメント |")).toBe(0);
    });

    it("接尾辞の無いマーカー名を数えない", () => {
      expect(countMarkerLines("<!-- sample-api -->")).toBe(0);
    });

    it("別名のマーカーを数えない", () => {
      expect(countMarkerLines("# setup-localize:begin")).toBe(0);
    });
  });
});

describe("isBaselineTarget", () => {
  describe("正常系", () => {
    it("通常の本文を対象にする", () => {
      expect(isBaselineTarget("docs/adr/README.md")).toBe(true);
      expect(isBaselineTarget("internal/di/module/job.go")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("生成物の接頭辞を対象から外す", () => {
      for (const prefix of EXCLUDED_PATH_PREFIXES) {
        expect(isBaselineTarget(`${prefix}anything.md`), prefix).toBe(false);
      }
    });

    it("自分自身のディレクトリを対象から外す", () => {
      expect(isBaselineTarget("scripts/marker-baseline/rules.test.ts")).toBe(false);
    });

    it("接頭辞が途中まで一致するだけのパスは外さない", () => {
      expect(isBaselineTarget("docs/portal/manifest.yaml")).toBe(true);
    });
  });
});

describe("diffBaseline", () => {
  describe("正常系", () => {
    it("一致していれば空を返す", () => {
      expect(diffBaseline({ "a.md": 2 }, { "a.md": 2 })).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("マーカーを持つファイルが増えたら落とす", () => {
      const failures = diffBaseline({ "a.md": 2, "new.md": 2 }, { "a.md": 2 });

      expect(failures).toHaveLength(1);
      expect(failures[0]).toContain("new.md");
      expect(failures[0]).toContain("MARKER_LITERAL_FILES");
    });

    // 既にマーカーを持つファイルへ例示を足す経路。ファイルの集合だけを見ていると漏れる。
    it("既存ファイルで行数が変わったら落とす", () => {
      const failures = diffBaseline({ "a.md": 4 }, { "a.md": 2 });

      expect(failures).toHaveLength(1);
      expect(failures[0]).toContain("2 → 4");
    });

    it("マーカーが無くなったら落とす", () => {
      const failures = diffBaseline({}, { "a.md": 2 });

      expect(failures).toHaveLength(1);
      expect(failures[0]).toContain("ベースラインのほうが古い");
    });
  });
});
