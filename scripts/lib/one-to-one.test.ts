import { describe, expect, it } from "vitest";
import {
  checkFile,
  collectDescribeTree,
  collectTestableExports,
  formatViolations,
  type DescribeNode,
} from "./one-to-one";

const anyCallable = () => true;
const noneCallable = () => false;

/** node を「名前と行だけ」に潰して比較しやすくする。 */
const shape = (nodes: readonly DescribeNode[]): unknown =>
  nodes.map((n) => ({ name: n.name, hasDirectIt: n.hasDirectIt, describes: shape(n.describes) }));

describe("collectDescribeTree", () => {
  describe("正常系", () => {
    it("最上位の describe を名前と行で拾う", () => {
      const tree = collectDescribeTree(`describe("foo", () => {});`, "a.test.ts");
      expect(tree).toEqual([{ name: "foo", line: 1, describes: [], hasDirectIt: false }]);
    });

    it("入れ子の describe を親の子として読む", () => {
      const tree = collectDescribeTree(
        `describe("foo", () => { describe("正常系", () => {}); });`,
        "a.test.ts",
      );
      expect(shape(tree)).toEqual([
        { name: "foo", hasDirectIt: false, describes: [{ name: "正常系", hasDirectIt: false, describes: [] }] },
      ]);
    });

    it("直下の it を hasDirectIt として記録する", () => {
      const tree = collectDescribeTree(`describe("foo", () => { it("x", () => {}); });`, "a.test.ts");
      expect(tree[0].hasDirectIt).toBe(true);
    });

    it("孫の it を親の hasDirectIt に数えない", () => {
      const tree = collectDescribeTree(
        `describe("foo", () => { describe("正常系", () => { it("x", () => {}); }); });`,
        "a.test.ts",
      );
      expect(tree[0].hasDirectIt).toBe(false);
      expect(tree[0].describes[0].hasDirectIt).toBe(true);
    });

    it("node:test 流のフラットな test も it として数える", () => {
      const tree = collectDescribeTree(`describe("foo", () => { test("x", () => {}); });`, "a.test.ts");
      expect(tree[0].hasDirectIt).toBe(true);
    });

    it("describe.only のような修飾付きの呼び出しも describe として読む", () => {
      const tree = collectDescribeTree(`describe.only("foo", () => {});`, "a.test.ts");
      expect(tree[0].name).toBe("foo");
    });
  });

  describe("異常系", () => {
    it("題名が文字列リテラルでない describe は木に載せない", () => {
      const tree = collectDescribeTree(`describe(name, () => { describe("foo", () => {}); });`, "a.test.ts");
      expect(shape(tree)).toEqual([{ name: "foo", hasDirectIt: false, describes: [] }]);
    });

    it("本体が関数でない describe は子を持たないものとして扱う", () => {
      const tree = collectDescribeTree(`describe("foo", 1 as never);`, "a.test.ts");
      expect(tree).toEqual([{ name: "foo", line: 1, describes: [], hasDirectIt: false }]);
    });

    it("引数の無い describe は木に載せない", () => {
      expect(collectDescribeTree(`describe();`, "a.test.ts")).toEqual([]);
    });
  });
});

describe("collectTestableExports", () => {
  describe("正常系", () => {
    it("export された関数宣言を拾う", () => {
      const got = collectTestableExports(`export function f() {}`, "a.ts", anyCallable);
      expect(got).toEqual([{ name: "f", line: 1, testable: true }]);
    });

    it("export されたクラス宣言を拾う", () => {
      const got = collectTestableExports(`export class C {}`, "a.ts", anyCallable);
      expect(got).toEqual([{ name: "C", line: 1, testable: true }]);
    });

    it("export された変数宣言を拾う", () => {
      const got = collectTestableExports(`export const f = () => {};`, "a.ts", anyCallable);
      expect(got).toEqual([{ name: "f", line: 1, testable: true }]);
    });

    it("末尾でまとめて出す export 句の名前を拾う", () => {
      const got = collectTestableExports(
        `function a() {}\nfunction b() {}\nexport { a, b };`,
        "a.ts",
        anyCallable,
      );
      expect(got.map((s) => s.name)).toEqual(["a", "b"]);
    });

    it("別名で出した export は公開される側の名前で拾う", () => {
      const got = collectTestableExports(`function a() {}\nexport { a as publicName };`, "a.ts", anyCallable);
      expect(got.map((s) => s.name)).toEqual(["publicName"]);
    });

    it("1 つの文で宣言された複数の変数をそれぞれ拾う", () => {
      const got = collectTestableExports(`export const a = () => {}, b = () => {};`, "a.ts", anyCallable);
      expect(got.map((s) => s.name)).toEqual(["a", "b"]);
    });
  });

  describe("異常系", () => {
    it("呼べない値は describe を要求しない印を付けて返す", () => {
      expect(collectTestableExports(`export const X = 1;`, "a.ts", noneCallable)).toEqual([
        { name: "X", line: 1, testable: false },
      ]);
    });

    it("export されていない宣言は対象にしない", () => {
      expect(collectTestableExports(`function f() {}`, "a.ts", anyCallable)).toEqual([]);
    });

    it("型だけの export は対象にしない", () => {
      expect(collectTestableExports(`export type T = () => void;`, "a.ts", anyCallable)).toEqual([]);
    });

    it("名前を持たない export default は対象にしない", () => {
      expect(collectTestableExports(`export default function () {}`, "a.ts", anyCallable)).toEqual([]);
    });

    it("分割代入で export された変数は名前を取れないので対象にしない", () => {
      expect(collectTestableExports(`export const { a } = obj;`, "a.ts", anyCallable)).toEqual([]);
    });

    it("型だけの export 句は対象にしない", () => {
      expect(collectTestableExports(`export type { T };`, "a.ts", anyCallable)).toEqual([]);
    });

    it("export 句の中で型と印された名前は対象にしない", () => {
      expect(collectTestableExports(`export { type T, a };`, "a.ts", anyCallable).map((s) => s.name)).toEqual([
        "a",
      ]);
    });

    it("再 export の * は名前を持たないので対象にしない", () => {
      expect(collectTestableExports(`export * from "./other";`, "a.ts", anyCallable)).toEqual([]);
    });

    it("名前空間としての再 export は個々の名前を持たないので対象にしない", () => {
      expect(collectTestableExports(`export * as ns from "./other";`, "a.ts", anyCallable)).toEqual([]);
    });
  });
});

describe("checkFile", () => {
  const symbol = { name: "f", line: 3, testable: true };
  const group = (name: string): DescribeNode => ({
    name,
    line: 2,
    describes: [],
    hasDirectIt: true,
  });
  const wellFormed: DescribeNode = {
    name: "f",
    line: 1,
    describes: [group("正常系"), group("異常系")],
    hasDirectIt: false,
  };

  describe("正常系", () => {
    it("export 名の describe に 正常系 と 異常系 が居れば違反にしない", () => {
      expect(
        checkFile({ file: "a.ts", testFile: "a.test.ts", exports: [symbol], describes: [wellFormed] }),
      ).toEqual([]);
    });

    it("正常系 だけでもグループが在れば違反にしない", () => {
      expect(
        checkFile({
          file: "a.ts",
          testFile: "a.test.ts",
          exports: [symbol],
          describes: [{ ...wellFormed, describes: [group("正常系")] }],
        }),
      ).toEqual([]);
    });

    it("対象 export が無いファイルはテストが無くても違反にしない", () => {
      expect(checkFile({ file: "a.ts", testFile: null, exports: [], describes: [] })).toEqual([]);
    });

    it("呼べない export には describe を要求しない", () => {
      expect(
        checkFile({
          file: "a.ts",
          testFile: "a.test.ts",
          exports: [symbol, { name: "CONST", line: 5, testable: false }],
          describes: [wellFormed],
        }),
      ).toEqual([]);
    });

    it("呼べない export に対する契約テストの describe は許す", () => {
      expect(
        checkFile({
          file: "a.ts",
          testFile: "a.test.ts",
          exports: [symbol, { name: "CONST", line: 5, testable: false }],
          describes: [wellFormed, { ...wellFormed, name: "CONST", line: 9 }],
        }),
      ).toEqual([]);
    });

    it("呼べる export が 1 つも無ければテストが無くても違反にしない", () => {
      expect(
        checkFile({
          file: "a.ts",
          testFile: null,
          exports: [{ name: "CONST", line: 5, testable: false }],
          describes: [],
        }),
      ).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("テストファイルが無ければ export ごとに違反にする", () => {
      const got = checkFile({ file: "a.ts", testFile: null, exports: [symbol], describes: [] });
      expect(got).toHaveLength(1);
      expect(got[0].kind).toBe("missing-test-file");
      expect(got[0].line).toBe(3);
    });

    it("対応する describe が無ければ違反にする", () => {
      const got = checkFile({ file: "a.ts", testFile: "a.test.ts", exports: [symbol], describes: [] });
      expect(got.map((v) => v.kind)).toEqual(["missing-describe"]);
    });

    it("同じ export に describe が 2 つあれば違反にする", () => {
      const got = checkFile({
        file: "a.ts",
        testFile: "a.test.ts",
        exports: [symbol],
        describes: [wellFormed, { ...wellFormed, line: 9 }],
      });
      expect(got.map((v) => v.kind)).toEqual(["duplicate-describe"]);
      expect(got[0].line).toBe(9);
    });

    it("最上位が 正常系 になっていれば入れ子の順序違反として報告する", () => {
      const got = checkFile({
        file: "a.ts",
        testFile: "a.test.ts",
        exports: [symbol],
        describes: [{ name: "正常系", line: 1, describes: [wellFormed], hasDirectIt: false }],
      });
      expect(got.map((v) => v.kind)).toEqual(["group-at-top", "missing-describe"]);
    });

    it("どの export にも対応しない describe を違反にする", () => {
      const got = checkFile({
        file: "a.ts",
        testFile: "a.test.ts",
        exports: [symbol],
        describes: [{ name: "f / g", line: 1, describes: [], hasDirectIt: false }, wellFormed],
      });
      expect(got.map((v) => v.kind)).toEqual(["unknown-describe"]);
    });

    it("export 名 describe の直下に it があれば違反にする", () => {
      const got = checkFile({
        file: "a.ts",
        testFile: "a.test.ts",
        exports: [symbol],
        describes: [{ ...wellFormed, hasDirectIt: true }],
      });
      expect(got.map((v) => v.kind)).toEqual(["case-outside-group"]);
    });

    it("正常系 も 異常系 も無ければ違反にする", () => {
      const got = checkFile({
        file: "a.ts",
        testFile: "a.test.ts",
        exports: [symbol],
        describes: [{ ...wellFormed, describes: [group("境界値")] }],
      });
      expect(got.map((v) => v.kind)).toEqual(["missing-group"]);
    });
  });
});

describe("formatViolations", () => {
  describe("正常系", () => {
    it("file:line: [kind] message の形で 1 行にする", () => {
      expect(
        formatViolations([{ kind: "missing-describe", file: "a.ts", line: 3, message: "no describe" }]),
      ).toBe("a.ts:3: [missing-describe] no describe");
    });

    it("複数件を安定した順序で並べる", () => {
      const got = formatViolations([
        { kind: "missing-describe", file: "b.ts", line: 1, message: "b" },
        { kind: "missing-describe", file: "a.ts", line: 1, message: "a" },
      ]);
      expect(got).toBe("a.ts:1: [missing-describe] a\nb.ts:1: [missing-describe] b");
    });

    it("既に順序どおりなら並べ替えない", () => {
      const got = formatViolations([
        { kind: "missing-describe", file: "a.ts", line: 1, message: "a" },
        { kind: "missing-describe", file: "b.ts", line: 1, message: "b" },
      ]);
      expect(got).toBe("a.ts:1: [missing-describe] a\nb.ts:1: [missing-describe] b");
    });

    it("違反が無ければ空文字を返す", () => {
      expect(formatViolations([])).toBe("");
    });
  });
});
