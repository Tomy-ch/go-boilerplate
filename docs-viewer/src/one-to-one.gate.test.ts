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
} from "../../scripts/lib/one-to-one";

/**
 * `docs-viewer` の 1:1 テスト対応ゲート。
 *
 * @remarks
 * 検査の中身は `scripts/lib/one-to-one.ts` が持ちます。パッケージは互いに独立していて依存を
 * 共有しないため（`docs-viewer/pnpm-workspace.yaml`）、共有するのは依存を持たない純粋モジュール
 * 1 本だけに留め、相対パスで直接読みます。ここが担うのはツリーの走査と型解決だけです。
 */

const PACKAGE_ROOT = resolve(import.meta.dirname, "..");
const REPOSITORY_ROOT = resolve(PACKAGE_ROOT, "..");

/**
 * ビューアーの entry。読み込まれた時点で DOM を触るため、判断はすべて `mount/mount-portal.tsx`
 * 側に置いてある。カバレッジ母数から外している対象と同じ（`vitest.config.ts`）。
 */
const EXCLUDED = ["src/main.tsx"];

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
    throw new Error("docs-viewer/tsconfig.json が見つかりません");
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

    if (!absolute.startsWith(`${PACKAGE_ROOT}/src/`) || /\.test\.tsx?$/.test(absolute)) {
      continue;
    }

    if (EXCLUDED.includes(relative(PACKAGE_ROOT, absolute))) {
      continue;
    }

    const testPath = absolute.replace(/\.(tsx?)$/, ".test.$1");
    const hasTest = existsSync(testPath);

    violations.push(
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

  return violations;
}

describe("docs-viewer の 1:1 テスト対応", () => {
  describe("正常系", () => {
    it("呼べる export はすべて専用の describe を持ち、その直下は 正常系 / 異常系 になっている", () => {
      expect(formatViolations(collectViolations())).toBe("");
    });
  });
});
