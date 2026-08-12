import ts from "typescript";

/**
 * TypeScript 側の 1:1 テスト対応原則を機械判定する。
 *
 * @remarks
 * Go 側は `internal/architest` の `TestUnitTestMappingCompleteness` が同じ原則を強制している。
 * 判定を TypeScript にも置くのは、片方だけが無検査だと「テストが在る」ことの意味が言語で
 * 変わってしまうため。入れ子の順序も Go に揃える（`TestXxx` の中に 正常系 / 異常系 が居るのと
 * 同じく、export 名の describe の中に 正常系 / 異常系 を置く）。
 *
 * このモジュールは fs も Program も持たない。走査と型解決は各パッケージのゲートが担い、
 * ここは「読み取り済みの構文木と、呼べるかどうかの判定」から違反を導くところだけを持つ。
 */

/** グループ名。export 名 describe の直下にこの名前が 1 つも無いと missing-group 違反になる。 */
export const GROUP_NAMES = ["正常系", "異常系"] as const;

export type GroupName = (typeof GROUP_NAMES)[number];

/** export 1 件。 */
export type ExportedSymbol = {
  readonly name: string;
  readonly line: number;
  /**
   * 専用の describe を要求する対象か。
   *
   * @remarks
   * 呼べる値（関数・クラス・`cva()` の戻り値など）だけが true になります。定数や zod スキーマは
   * false で、describe を要求されません。ただし describe を書くこと自体は許されます
   * （production シンボルに対応しない契約テストは 1:1 違反ではない、という規約に従う）。
   */
  readonly testable: boolean;
};

/** テストファイル中の describe 1 件。 */
export type DescribeNode = {
  readonly name: string;
  readonly line: number;
  readonly describes: readonly DescribeNode[];
  /** 直下に it を持つか。export 名 describe の直下に it が居る形を検出するために要る。 */
  readonly hasDirectIt: boolean;
};

export type ViolationKind =
  /** export に対応する describe が無い。 */
  | "missing-describe"
  /** テストファイルそのものが無い。 */
  | "missing-test-file"
  /** 同じ export に describe が 2 つ以上ある。 */
  | "duplicate-describe"
  /** 最上位 describe の名前がどの export とも対応しない（束ね describe を含む）。 */
  | "unknown-describe"
  /** export 名 describe の直下に 正常系 / 異常系 が無い。 */
  | "missing-group"
  /** export 名 describe の直下に it が居る（グループを飛ばしている）。 */
  | "case-outside-group"
  /** 最上位が 正常系 / 異常系 になっている（Go と入れ子順が逆）。 */
  | "group-at-top";

export type Violation = {
  readonly kind: ViolationKind;
  readonly file: string;
  readonly line: number;
  readonly message: string;
};

/** 1 ファイル分の検査入力。 */
export type FileInput = {
  /** 検査対象の production ファイル（リポジトリ相対）。 */
  readonly file: string;
  /** 対応するテストファイル（リポジトリ相対）。存在しなければ null。 */
  readonly testFile: string | null;
  readonly exports: readonly ExportedSymbol[];
  readonly describes: readonly DescribeNode[];
};

const isGroupName = (name: string): name is GroupName =>
  (GROUP_NAMES as readonly string[]).includes(name);

/** ノードの開始行（1 始まり）を返す。 */
function lineOf(source: ts.SourceFile, node: ts.Node): number {
  return source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1;
}

/** `describe` / `it` の第 1 引数がリテラル文字列なら取り出す。 */
function literalTitle(node: ts.CallExpression): string | null {
  const [first] = node.arguments;

  if (first === undefined) {
    return null;
  }

  return ts.isStringLiteralLike(first) ? first.text : null;
}

/** 呼び出しが `describe(...)` / `describe.only(...)` などであれば true。 */
function isCallOf(node: ts.CallExpression, name: string): boolean {
  const callee = node.expression;

  if (ts.isIdentifier(callee)) {
    return callee.text === name;
  }

  return (
    ts.isPropertyAccessExpression(callee) &&
    ts.isIdentifier(callee.expression) &&
    callee.expression.text === name
  );
}

/**
 * describe / it の本体（第 2 引数の関数）を返す。
 *
 * @remarks
 * 受け付けるのはアロー関数と関数式だけです。この 2 つは本体を必ず持つため、本体の有無で
 * 分岐する必要がありません。`ts.isFunctionLike` は本体を持たない宣言形（メソッドシグネチャ等）
 * まで通しますが、それらは引数の位置に書けません。
 */
function callbackBody(node: ts.CallExpression): ts.Node | null {
  const [, second] = node.arguments;

  if (second === undefined || (!ts.isArrowFunction(second) && !ts.isFunctionExpression(second))) {
    return null;
  }

  return second.body;
}

/**
 * テストファイルの describe 入れ子を読む。
 *
 * @remarks
 * 型は見ない。`describe` は import 名で判別できる呼び出しであり、構文だけで木になる。
 */
export function collectDescribeTree(sourceText: string, fileName: string): DescribeNode[] {
  const source = ts.createSourceFile(fileName, sourceText, ts.ScriptTarget.Latest, true);

  const walk = (container: ts.Node): { describes: DescribeNode[]; hasDirectIt: boolean } => {
    const describes: DescribeNode[] = [];
    let hasDirectIt = false;

    const visit = (node: ts.Node): void => {
      if (ts.isCallExpression(node)) {
        if (isCallOf(node, "describe")) {
          const title = literalTitle(node);
          const body = callbackBody(node);

          if (title !== null) {
            const inner = body === null ? { describes: [], hasDirectIt: false } : walk(body);
            describes.push({
              name: title,
              line: lineOf(source, node),
              describes: inner.describes,
              hasDirectIt: inner.hasDirectIt,
            });

            return;
          }
        }

        if (isCallOf(node, "it") || isCallOf(node, "test")) {
          hasDirectIt = true;

          return;
        }
      }

      ts.forEachChild(node, visit);
    };

    ts.forEachChild(container, visit);

    return { describes, hasDirectIt };
  };

  return walk(source).describes;
}

/**
 * テスト対象になりうる export を読む。
 *
 * @remarks
 * `isCallable` に「その名前が呼べる値か」を渡す。呼べないもの（定数・配列・zod スキーマ）は
 * Go 側のゲートが func / method だけを見るのと同じ粒度で対象外にする。呼べるかどうかは構文では
 * 決まらない（`cva(...)` は関数を返し `z.record(...)` は返さない）ため、判定は型を持つ側に委ねる。
 */
export function collectTestableExports(
  sourceText: string,
  fileName: string,
  isCallable: (name: string) => boolean,
): ExportedSymbol[] {
  const source = ts.createSourceFile(fileName, sourceText, ts.ScriptTarget.Latest, true);
  const found: ExportedSymbol[] = [];

  const add = (name: string, node: ts.Node): void => {
    found.push({ name, line: lineOf(source, node), testable: isCallable(name) });
  };

  const exported = (node: ts.Node): boolean =>
    ts.canHaveModifiers(node) &&
    (ts.getModifiers(node) ?? []).some((m) => m.kind === ts.SyntaxKind.ExportKeyword);

  ts.forEachChild(source, (node) => {
    if (ts.isFunctionDeclaration(node) && exported(node) && node.name !== undefined) {
      add(node.name.text, node);

      return;
    }

    if (ts.isClassDeclaration(node) && exported(node) && node.name !== undefined) {
      add(node.name.text, node);

      return;
    }

    if (ts.isVariableStatement(node) && exported(node)) {
      for (const decl of node.declarationList.declarations) {
        if (ts.isIdentifier(decl.name)) {
          add(decl.name.text, decl);
        }
      }

      return;
    }

    // `export { A, B as C }`。宣言に export 修飾子を付けず末尾でまとめて出す形は珍しくなく、
    // 見落とすと 1 ファイル分の export がまるごと検査から外れる。公開される名前は別名の側。
    if (ts.isExportDeclaration(node) && node.exportClause !== undefined) {
      const clause = node.exportClause;

      if (ts.isNamedExports(clause)) {
        for (const element of clause.elements) {
          if (!element.isTypeOnly && !node.isTypeOnly) {
            add(element.name.text, element);
          }
        }
      }
    }
  });

  return found;
}

/** export 名 describe 1 つ分の内側の形を検査する。 */
function checkGroups(testFile: string, node: DescribeNode): Violation[] {
  const violations: Violation[] = [];

  if (node.hasDirectIt) {
    violations.push({
      kind: "case-outside-group",
      file: testFile,
      line: node.line,
      message: `describe("${node.name}") の直下に it があります。正常系 / 異常系 のどちらかへ入れてください`,
    });
  }

  const groups = node.describes.filter((d) => isGroupName(d.name));

  if (groups.length === 0) {
    violations.push({
      kind: "missing-group",
      file: testFile,
      line: node.line,
      message: `describe("${node.name}") の直下に 正常系 / 異常系 がありません`,
    });
  }

  return violations;
}

/**
 * 1 ファイル分の 1:1 対応を検査する。
 *
 * @remarks
 * 「export に describe が無い」と「describe に対応する export が無い」の両方向を見る。
 * 片方向だけだと、export を消してテストだけ残った状態や、テストを別名へ改名した状態が
 * 検査をすり抜ける。
 */
export function checkFile(input: FileInput): Violation[] {
  const required = input.exports.filter((s) => s.testable);

  if (required.length === 0) {
    return [];
  }

  if (input.testFile === null) {
    return required.map((symbol) => ({
      kind: "missing-test-file" as const,
      file: input.file,
      line: symbol.line,
      message: `${symbol.name} に対応するテストファイルがありません`,
    }));
  }

  const testFile = input.testFile;
  const violations: Violation[] = [];
  const byName = new Map<string, DescribeNode[]>();

  for (const node of input.describes) {
    byName.set(node.name, [...(byName.get(node.name) ?? []), node]);
  }

  // describe を「要求する」のは呼べる export だけですが、「許す」のは export 全体です。
  // 定数の契約テストは production の関数に対応しませんが、退行を単独で捕まえるので違反ではない。
  const exportNames = new Set(input.exports.map((s) => s.name));

  for (const node of input.describes) {
    if (exportNames.has(node.name)) {
      continue;
    }

    violations.push({
      kind: isGroupName(node.name) ? "group-at-top" : "unknown-describe",
      file: testFile,
      line: node.line,
      message: isGroupName(node.name)
        ? `最上位が describe("${node.name}") になっています。export 名の describe を最上位に置き、その内側へ入れてください`
        : `describe("${node.name}") はどの export にも対応しません。1 つの export に 1 つの describe を対応させてください`,
    });
  }

  for (const symbol of required) {
    const nodes = byName.get(symbol.name) ?? [];

    if (nodes.length === 0) {
      violations.push({
        kind: "missing-describe",
        file: input.file,
        line: symbol.line,
        message: `${symbol.name} に対応する describe("${symbol.name}") が ${testFile} にありません`,
      });

      continue;
    }

    if (nodes.length > 1) {
      violations.push({
        kind: "duplicate-describe",
        file: testFile,
        line: nodes[1].line,
        message: `describe("${symbol.name}") が ${nodes.length} つあります。1 つにまとめてください`,
      });
    }

    violations.push(...checkGroups(testFile, nodes[0]));
  }

  return violations;
}

/** 違反一覧を、失敗メッセージとして読める 1 つの文字列へ整形する。 */
export function formatViolations(violations: readonly Violation[]): string {
  return violations
    .map((v) => `${v.file}:${v.line}: [${v.kind}] ${v.message}`)
    .sort((a, b) => (a < b ? -1 : 1))
    .join("\n");
}
