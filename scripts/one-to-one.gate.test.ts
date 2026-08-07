import { existsSync, readFileSync } from "node:fs";
import { relative, resolve } from "node:path";
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
 * `scripts/` の 1:1 テスト対応ゲート。
 *
 * @remarks
 * Go 側の `internal/architest` が `internal/` `pkg/` `scripts/` に対して同じ原則を強制している。
 * TypeScript 側だけ無検査だと「テストが在る」の意味が言語で変わるため、同じ検査をここへ置く。
 * 検査の中身は `lib/one-to-one.ts` が持ち、ここはツリーの走査と型解決だけを担う。
 */

const PACKAGE_ROOT = import.meta.dirname;

/** `portal/gen-*.ts` のような宣言を、パッケージ相対パスに当てる正規表現へ変える。 */
function toMatcher(pattern: string): RegExp {
  const escaped = pattern.replace(/[.+^${}()|[\]\\]/g, "\\$&").replace(/\*/g, "[^/]*");

  return new RegExp(`^${escaped}$`);
}

const excluded = EXCLUDED_FROM_CHECKS.map(toMatcher);

/** module symbol の各 export が「呼べる値」かを型から判定する述語を作る。 */
function callablePredicate(checker: ts.TypeChecker, source: ts.SourceFile): (name: string) => boolean {
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

function collectViolations(): Violation[] {
  const configPath = ts.findConfigFile(PACKAGE_ROOT, ts.sys.fileExists, "tsconfig.json");

  if (configPath === undefined) {
    throw new Error("scripts/tsconfig.json が見つかりません");
  }

  const parsed = ts.parseJsonConfigFileContent(
    ts.readConfigFile(configPath, ts.sys.readFile).config,
    ts.sys,
    PACKAGE_ROOT,
  );
  const program = ts.createProgram(parsed.fileNames, parsed.options);
  const checker = program.getTypeChecker();
  const violations: Violation[] = [];

  for (const source of program.getSourceFiles()) {
    const absolute = source.fileName;

    if (source.isDeclarationFile || absolute.includes("node_modules")) {
      continue;
    }

    if (!absolute.startsWith(`${PACKAGE_ROOT}/`) || /\.test\.ts$/.test(absolute)) {
      continue;
    }

    const inPackage = relative(PACKAGE_ROOT, absolute);

    if (excluded.some((matcher) => matcher.test(inPackage))) {
      continue;
    }

    const testPath = absolute.replace(/\.ts$/, ".test.ts");
    const hasTest = existsSync(testPath);

    violations.push(
      ...checkFile({
        file: relative(resolve(PACKAGE_ROOT, ".."), absolute),
        testFile: hasTest ? relative(resolve(PACKAGE_ROOT, ".."), testPath) : null,
        exports: collectTestableExports(
          source.getFullText(),
          absolute,
          callablePredicate(checker, source),
        ),
        describes: hasTest ? collectDescribeTree(readFileSync(testPath, "utf8"), testPath) : [],
      }),
    );
  }

  return violations;
}

describe("scripts の 1:1 テスト対応", () => {
  describe("正常系", () => {
    it("呼べる export はすべて専用の describe を持ち、その直下は 正常系 / 異常系 になっている", () => {
      const violations = collectViolations();

      expect(formatViolations(violations)).toBe("");
    });
  });
});
