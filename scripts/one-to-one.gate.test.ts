import { existsSync, readFileSync } from "node:fs";
import { relative, resolve } from "node:path";
import yaml from "js-yaml";
import ts from "typescript";
import { describe, expect, it } from "vitest";

import {
  checkFile,
  collectDescribeTree,
  collectTestableExports,
  formatViolations,
  type Violation,
} from "./lib/one-to-one";
import { EXCLUDED_FROM_CHECKS } from "./lib/untested-modules";

/**
 * TypeScript 全パッケージの 1:1 テスト対応ゲート。
 *
 * @remarks
 * Go 側の `internal/architest` が `internal/` `pkg/` `scripts/` に対して同じ原則を強制している。
 * TypeScript 側だけ無検査だと「テストが在る」の意味が言語で変わるため、同じ検査をここへ置く。
 * 検査の中身は `lib/one-to-one.ts` が持ち、ここはツリーの走査と型解決だけを担う。
 *
 * ゲートを各パッケージへ置かず `scripts/` へ集約しているのは、判定が `typescript` に依存するため。
 * Node の解決は import 元のファイル位置から始まるので、他パッケージから
 * `../../scripts/lib/one-to-one` を読むと `typescript` を `scripts/node_modules` に探しに行き、
 * 自分の依存しか入れない CI では解決できない。走査する側をここへ置けばその問題自体が消える。
 *
 * 走査対象は `PACKAGES` が持つ一方、このゲートを起こす条件は CI の `scripts-check.yaml` と
 * 手元の `.lefthook.yaml` が別々に持つ。走査対象だけが増えるとゲートは起動しないまま緑を返すため、
 * 末尾の describe が両者の整合も検査する。
 */

const REPOSITORY_ROOT = resolve(import.meta.dirname, "..");

const WORKFLOW_PATH = resolve(REPOSITORY_ROOT, ".github/workflows/scripts-check.yaml");
const LEFTHOOK_PATH = resolve(REPOSITORY_ROOT, ".lefthook.yaml");

/** 検査対象のパッケージ 1 件。 */
type TargetPackage = {
  name: string;
  /** リポジトリルートからのパッケージディレクトリ。 */
  dir: string;
  /**
   * 検査する production ファイルの、パッケージからの相対接頭辞。
   *
   * @remarks
   * `scripts` はパッケージ全体がツールの集合なので空。他の 2 つは `src/` だけがアプリ本体で、
   * ビルド設定やスクラッチは対象外。
   */
  sourcePrefix: string;
  /** 検査から外すファイル / 接頭辞（パッケージからの相対）。 */
  excluded: readonly string[];
};

const PACKAGES: readonly TargetPackage[] = [
  {
    name: "scripts",
    dir: "scripts",
    sourcePrefix: "",
    excluded: EXCLUDED_FROM_CHECKS,
  },
  {
    name: "docs-viewer",
    dir: "docs-viewer",
    sourcePrefix: "src/",
    // ビューアーの entry。読み込まれた時点で DOM を触るため、判断はすべて mount 側に置いてある。
    excluded: ["src/main.tsx"],
  },
  {
    name: "mock-auth-server",
    dir: "mock-auth-server",
    sourcePrefix: "src/",
    // 起動の入口（読み込んだ時点で listen する）と orval の生成物。
    excluded: ["src/server.ts", "src/generated/"],
  },
];

/** `portal/gen-*.ts` のような宣言を、パッケージ相対パスに当てる正規表現へ変える。 */
function toMatcher(pattern: string): RegExp {
  const escaped = pattern.replace(/[.+^${}()|[\]\\]/g, "\\$&").replace(/\*/g, "[^/]*");

  return new RegExp(`^${escaped}$`);
}

/** 除外宣言に当たるか。末尾が `/` の宣言はディレクトリ接頭辞として扱う。 */
function isExcluded(relativePath: string, excluded: readonly string[]): boolean {
  return excluded.some((pattern) =>
    pattern.endsWith("/")
      ? relativePath.startsWith(pattern)
      : toMatcher(pattern).test(relativePath),
  );
}

/** module symbol の各 export が「呼べる値」かを型から判定する述語を作る。 */
function callablePredicate(
  checker: ts.TypeChecker,
  source: ts.SourceFile,
): (name: string) => boolean {
  const moduleSymbol = checker.getSymbolAtLocation(source);
  const exports = moduleSymbol === undefined ? [] : checker.getExportsOfModule(moduleSymbol);
  const byName = new Map(exports.map((symbol) => [symbol.getName(), symbol]));

  return (name) => {
    const symbol = byName.get(name);
    const declaration = symbol?.valueDeclaration ?? symbol?.declarations?.[0];

    if (symbol === undefined || declaration === undefined) {
      return false;
    }

    const type = checker.getTypeOfSymbolAtLocation(symbol, declaration);

    return type.getCallSignatures().length > 0 || type.getConstructSignatures().length > 0;
  };
}

function createProgram(packageRoot: string): ts.Program {
  const configPath = ts.findConfigFile(packageRoot, ts.sys.fileExists, "tsconfig.json");

  if (configPath === undefined) {
    throw new Error(`${packageRoot}/tsconfig.json が見つかりません`);
  }

  const parsed = ts.parseJsonConfigFileContent(
    ts.readConfigFile(configPath, ts.sys.readFile).config,
    ts.sys,
    packageRoot,
  );

  return ts.createProgram(parsed.fileNames, parsed.options);
}

/** 1 パッケージ分の走査結果。 */
type PackageScan = {
  violations: Violation[];
  /**
   * 型を解決できなかった import。
   *
   * @remarks
   * 「呼べる値か」は型で決まるため、依存が解決できないと `any` になり、呼べる export が
   * 呼べないものとして扱われます。ゲートは違反ゼロを報告したまま黙るので、別に検出します。
   */
  unresolved: string[];
  /** 検査した production ファイル数。0 なら走査が的を外している。 */
  checkedFiles: number;
};

function scanPackage(target: TargetPackage): PackageScan {
  const packageRoot = resolve(REPOSITORY_ROOT, target.dir);
  const program = createProgram(packageRoot);
  const checker = program.getTypeChecker();
  const scan: PackageScan = { violations: [], unresolved: [], checkedFiles: 0 };
  const sourceRoot = `${packageRoot}/${target.sourcePrefix}`;

  for (const source of program.getSourceFiles()) {
    const absolute = source.fileName;

    if (source.isDeclarationFile || absolute.includes("node_modules")) {
      continue;
    }
    if (!absolute.startsWith(sourceRoot) || /\.test\.tsx?$/.test(absolute)) {
      continue;
    }

    const inPackage = relative(packageRoot, absolute);

    if (isExcluded(inPackage, target.excluded)) {
      continue;
    }

    for (const diagnostic of program.getSemanticDiagnostics(source)) {
      // TS2307: モジュールを解決できない。依存未導入で型が any へ落ちる唯一の入口。
      if (diagnostic.code === 2307) {
        scan.unresolved.push(
          `${relative(REPOSITORY_ROOT, absolute)}: ${ts.flattenDiagnosticMessageText(diagnostic.messageText, " ")}`,
        );
      }
    }

    const testPath = absolute.replace(/\.(tsx?)$/, ".test.$1");
    const hasTest = existsSync(testPath);

    scan.checkedFiles += 1;
    scan.violations.push(
      ...checkFile({
        file: relative(REPOSITORY_ROOT, absolute),
        testFile: hasTest ? relative(REPOSITORY_ROOT, testPath) : null,
        exports: collectTestableExports(
          source.getFullText(),
          absolute,
          callablePredicate(checker, source),
        ),
        describes: hasTest ? collectDescribeTree(readFileSync(testPath, "utf8"), testPath) : [],
      }),
    );
  }

  return scan;
}

// 3 パッケージ分の TypeScript プログラムを構築するため、既定の 5 秒では足りない。
const TIMEOUT_MS = 120_000;

describe.each(PACKAGES)("$name の 1:1 テスト対応", (target) => {
  describe("正常系", () => {
    it(
      "呼べる export はすべて専用の describe を持ち、その直下は 正常系 / 異常系 になっている",
      () => {
        const scan = scanPackage(target);

        // 依存を解決できないと型が any になり、呼べる export を取りこぼしたまま違反ゼロを
        // 報告する。違反より先にこちらを主張して、ゲートが黙った状態を緑にしない。
        expect(scan.unresolved.join("\n")).toBe("");
        expect(scan.checkedFiles).toBeGreaterThan(0);
        expect(formatViolations(scan.violations)).toBe("");
      },
      TIMEOUT_MS,
    );
  });
});

/**
 * `scripts-check.yaml` が `pull_request` で監視しているパス。
 *
 * @remarks
 * `on.pull_request.paths` の形が変わった場合は空配列になり、呼び出し側が全パッケージを未カバーと
 * して挙げます。ファイルを読めない場合は例外になり、どちらにしても緑では終わりません。
 */
function triggerPaths(): readonly string[] {
  const workflow = yaml.load(readFileSync(WORKFLOW_PATH, "utf8")) as {
    on?: { pull_request?: { paths?: string[] } };
  };

  return workflow.on?.pull_request?.paths ?? [];
}

/** `.lefthook.yaml` の `pre-push` でこのスイートを起こす glob。読めない場合は {@link triggerPaths} と同じ。 */
function pushGlobs(): readonly string[] {
  const lefthook = yaml.load(readFileSync(LEFTHOOK_PATH, "utf8")) as {
    "pre-push"?: { commands?: { "scripts-test"?: { glob?: string[] } } };
  };

  return lefthook["pre-push"]?.commands?.["scripts-test"]?.glob ?? [];
}

/**
 * `PACKAGES` のうち、与えられた起動条件に覆われていないソースルートを挙げる。
 *
 * @remarks
 * 覆っていると認めるのは、ソースルートで始まり、その先にワイルドカードを持つエントリだけです。
 * 単一ファイルを名指しするエントリは、ソースルートで始まっていても配下の他のファイルを起こさないため
 * 覆っているとは扱いません。逆に、ワイルドカードの及ぶ範囲が配下の全ファイルかどうかまでは見ません。
 */
function uncoveredSourceRoots(entries: readonly string[]): string[] {
  const covers = (sourceRoot: string) => (entry: string) =>
    entry.startsWith(sourceRoot) && entry.slice(sourceRoot.length).includes("*");

  return PACKAGES.map((target) => `${target.dir}/${target.sourcePrefix}`).filter(
    (sourceRoot) => !entries.some(covers(sourceRoot)),
  );
}

describe("ゲートを起動する条件の追従", () => {
  describe("正常系", () => {
    it("走査対象パッケージのソースは scripts-check.yaml の paths に覆われている", () => {
      expect(uncoveredSourceRoots(triggerPaths()).join("\n")).toBe("");
    });

    it("走査対象パッケージのソースは .lefthook.yaml の pre-push glob に覆われている", () => {
      expect(uncoveredSourceRoots(pushGlobs()).join("\n")).toBe("");
    });
  });
});
