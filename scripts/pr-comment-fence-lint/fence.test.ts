import { describe, expect, it } from "vitest";

import {
  compareImplementations,
  extractFenceFor,
  fencesBody,
  findFixedFences,
  findInterpolatedSpans,
  hasPassThroughCall,
  scanWorkflow,
} from "./fence";

const USES = "        uses: ./.github/actions/upsert-pr-comment";

describe("extractFenceFor", () => {
  describe("正常系", () => {
    it("関数本文を字下げを均して取り出す", () => {
      const lines = ["          fence_for() {", "            local x=1", "          }", "          echo done"];

      expect(extractFenceFor(lines)).toBe("fence_for() {\nlocal x=1\n}");
    });

    it("字下げだけが違う複製を同じ本文として扱う", () => {
      const shallow = ["  fence_for() {", "    local x=1", "  }"];
      const deep = ["      fence_for() {", "        local x=1", "      }"];

      expect(extractFenceFor(shallow)).toBe(extractFenceFor(deep));
    });
  });

  describe("異常系", () => {
    it("fence_for を持たない定義は null を返す", () => {
      expect(extractFenceFor(["  echo hi"])).toBeNull();
    });

    it("閉じられていない fence_for は null を返す（半端な本文で突き合わせない）", () => {
      expect(extractFenceFor(["  fence_for() {", "    local x=1"])).toBeNull();
    });
  });
});

describe("findFixedFences", () => {
  describe("正常系", () => {
    it("固定長のフェンスを出す echo 行を行番号付きで返す", () => {
      expect(findFixedFences(["          echo '```'"])).toEqual([{ line: 1, text: "echo '```'" }]);
    });

    it("言語指定付きのフェンスも拾う", () => {
      expect(findFixedFences(['          echo "```log"'])).toHaveLength(1);
    });

    it("変数でフェンス長を組む行は対象外にする", () => {
      expect(findFixedFences(['          echo "${fence}log"'])).toEqual([]);
    });

    it("echo 以外の行は見ない", () => {
      expect(findFixedFences(["          printf '```'"])).toEqual([]);
    });

    it("フェンスを出さない echo 行を違反にしない", () => {
      expect(findFixedFences(["          echo 'md-lint に違反はありません'"])).toEqual([]);
    });
  });
});

describe("fencesBody", () => {
  describe("正常系", () => {
    it("値のある details-summary はアクションにフェンスさせる", () => {
      expect(fencesBody("          details-summary: 'md-lint log'")).toBe(true);
    });
  });

  describe("異常系", () => {
    it("details-summary を持たない行は false", () => {
      expect(fencesBody("          marker: '<!-- x -->'")).toBe(false);
    });

    it("空文字の details-summary をフェンス済みと見なさない", () => {
      expect(fencesBody("          details-summary: ''")).toBe(false);
      expect(fencesBody('          details-summary: ""')).toBe(false);
      expect(fencesBody("          details-summary:")).toBe(false);
    });

    it("式の details-summary は静的に空か判定できないためフェンスされない側へ倒す", () => {
      expect(fencesBody("          details-summary: ${{ steps.x.outputs.summary }}")).toBe(false);
    });
  });
});

describe("hasPassThroughCall", () => {
  describe("正常系", () => {
    it("details-summary の無い呼び出しを素通しと判定する", () => {
      expect(hasPassThroughCall(["      - name: comment", USES, "        with:", "          marker: x"])).toBe(
        true,
      );
    });

    it("details-summary のある呼び出しを素通しと判定しない", () => {
      expect(
        hasPassThroughCall(["      - name: comment", USES, "        with:", "          details-summary: 'log'"]),
      ).toBe(false);
    });

    it("空行を挟んでも同じステップの details-summary を見つける", () => {
      expect(hasPassThroughCall(["      - name: comment", USES, "", "        with:", "          details-summary: 'log'"])).toBe(
        false,
      );
    });
  });

  describe("異常系", () => {
    it("後続ステップの details-summary を自分のものとして拾わない", () => {
      const lines = [
        "      - name: pass through",
        USES,
        "        with:",
        "          marker: x",
        "      - name: fenced",
        USES,
        "        with:",
        "          details-summary: 'log'",
      ];

      expect(hasPassThroughCall(lines)).toBe(true);
    });

    it("`- uses:` から書き始めたステップでも次のステップで区切る", () => {
      const lines = [
        "      - uses: ./.github/actions/upsert-pr-comment",
        "        with:",
        "          marker: x",
        "      - uses: ./.github/actions/upsert-pr-comment",
        "        with:",
        "          details-summary: 'log'",
      ];

      expect(hasPassThroughCall(lines)).toBe(true);
    });

    it("ステップを開く `-` が上に無い呼び出しでも、後続の details-summary を自分のものにしない", () => {
      const lines = [
        "uses: ./.github/actions/upsert-pr-comment",
        "with:",
        "- uses: ./.github/actions/upsert-pr-comment",
        "  with:",
        "    details-summary: 'log'",
      ];

      expect(hasPassThroughCall(lines)).toBe(true);
    });

    it("呼び出しが無ければ素通しも無い", () => {
      expect(hasPassThroughCall(["      - uses: actions/checkout@v7"])).toBe(false);
    });
  });
});

describe("findInterpolatedSpans", () => {
  describe("正常系", () => {
    it("シェル変数展開を含む span を行番号付きで返す", () => {
      expect(findInterpolatedSpans(["          echo \"file: `${name}`\""])).toEqual([
        { line: 1, text: 'echo "file: `${name}`"' },
      ]);
    });

    it("printf の変換指定を含む span も拾う", () => {
      expect(findInterpolatedSpans(["          printf 'file: `%s`\\n' \"$f\""])).toHaveLength(1);
    });
  });

  describe("異常系", () => {
    it("規約そのものを説明するコメント行に反応しない", () => {
      expect(findInterpolatedSpans(["          # `${name}` のように書いてはいけない"])).toEqual([]);
    });

    it("補間を含まない span を違反にしない", () => {
      expect(findInterpolatedSpans(["          echo 'see `docs/rules.md`'"])).toEqual([]);
    });

    it("改行を跨いだ 2 つのバッククォートを 1 つの span と見なさない", () => {
      expect(findInterpolatedSpans(["          echo '`'", "          echo '${x}`'"])).toEqual([]);
    });
  });
});

const PASS_THROUGH = ["      - name: comment", USES, "        with:", "          marker: x"];

describe("scanWorkflow", () => {
  describe("正常系", () => {
    it("違反が無ければ空の一覧を返す", () => {
      const scan = scanWorkflow("w.yaml", ["          echo hello"]);

      expect(scan.violations).toEqual([]);
      expect(scan.implementation).toBeNull();
    });

    it("そのファイルが持つ fence_for の実装を返す", () => {
      const scan = scanWorkflow("w.yaml", ["          fence_for() {", "            local x=1", "          }"]);

      expect(scan.implementation).toBe("fence_for() {\nlocal x=1\n}");
    });

    it("素通し呼び出しが無ければ span を検査しない", () => {
      const scan = scanWorkflow("w.yaml", ['          echo "`${value}`"']);

      expect(scan.violations).toEqual([]);
    });

    it("除外に載ったファイルは span を検査しない", () => {
      const lines = [...PASS_THROUGH, '          echo "`${value}`"'];

      const scan = scanWorkflow(
        ".github/workflows/w.yaml",
        lines,
        new Map([["w.yaml", "#123"]]),
      );

      expect(scan.violations).toEqual([]);
    });

    it("除外はファイル名で当てるのでディレクトリの違いに影響されない", () => {
      const lines = [...PASS_THROUGH, '          echo "`${value}`"'];

      const bare = scanWorkflow("w.yaml", lines, new Map([["w.yaml", "#123"]]));

      expect(bare.violations).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("固定長のフェンスをファイル名と行番号付きで挙げる", () => {
      const scan = scanWorkflow("w.yaml", ["          echo '```'"]);

      expect(scan.violations).toHaveLength(1);
      expect(scan.violations[0]).toContain("w.yaml:1:");
      expect(scan.violations[0]).toContain("固定長のフェンス");
    });

    it("素通し呼び出しのあるファイルの span 補間を挙げる", () => {
      const scan = scanWorkflow("w.yaml", [...PASS_THROUGH, '          echo "`${value}`"']);

      expect(scan.violations).toHaveLength(1);
      expect(scan.violations[0]).toContain("inline code span");
    });

    it("除外に載っていないファイルは span を検査する", () => {
      const scan = scanWorkflow(
        ".github/workflows/other.yaml",
        [...PASS_THROUGH, '          echo "`${value}`"'],
        new Map([["w.yaml", "#123"]]),
      );

      expect(scan.violations).toHaveLength(1);
    });
  });
});

describe("compareImplementations", () => {
  describe("正常系", () => {
    it("実装が全て一致すれば違反にしない", () => {
      expect(
        compareImplementations([
          ["a.yaml", "body"],
          ["b.yaml", "body"],
        ]),
      ).toEqual([]);
    });

    it("実装が 1 件だけなら突き合わせる相手が無く違反にしない", () => {
      expect(compareImplementations([["a.yaml", "body"]])).toEqual([]);
    });

    it("実装が 1 件も無ければ違反にしない", () => {
      expect(compareImplementations([])).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("先頭と食い違う実装を、ずれた側のファイル名で挙げる", () => {
      const violations = compareImplementations([
        ["a.yaml", "body"],
        ["b.yaml", "other"],
      ]);

      expect(violations).toHaveLength(1);
      expect(violations[0]).toContain("b.yaml");
      expect(violations[0]).toContain("a.yaml");
    });

    it("食い違う実装が複数あればすべて挙げる", () => {
      const violations = compareImplementations([
        ["a.yaml", "body"],
        ["b.yaml", "other"],
        ["c.yaml", "another"],
      ]);

      expect(violations).toHaveLength(2);
    });
  });
});
