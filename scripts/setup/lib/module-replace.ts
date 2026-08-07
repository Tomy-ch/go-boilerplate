import path from "node:path";

// Go モジュール名が現れうるファイル種別。ここに載らない拡張子は走査しても書き換えない。
const TARGET_EXTENSIONS: ReadonlySet<string> = new Set([
  ".go",
  ".yaml",
  ".yml",
  ".mod",
  ".md",
  ".js",
  ".json",
  ".cjs",
  ".mjs",
  ".html",
]);
const TARGET_BASENAMES: ReadonlySet<string> = new Set(["Dockerfile"]);

/** 走査そのものを打ち切るディレクトリ名。 */
export const EXCLUDED_DIRECTORIES: ReadonlySet<string> = new Set([
  "vendor",
  "tmp",
  "node_modules",
  ".git",
]);

const EXCLUDED_PATH_PREFIXES = [`docs${path.sep}`, `scripts${path.sep}setup${path.sep}`];
const EXCLUDED_PATH_SUFFIXES = [".gen.go", ".sql.go", `${path.sep}openapi.gen.yaml`];
// mockgen 生成物（make gen-api で再生成されるため対象外）
const EXCLUDED_BASENAME_PATTERNS = [/^mock_.*\.go$/, /_mock\.go$/];

/**
 * モジュール名の指定を検証する。
 *
 * @throws 単純なプロジェクト名でない、または新旧が同一の場合。
 */
export function ensureModuleArguments(oldModule: string, newModule: string): void {
  ensureSimpleModuleName(oldModule, "旧モジュール名");
  ensureSimpleModuleName(newModule, "新モジュール名");

  if (oldModule === newModule) {
    throw new Error("旧モジュール名と新モジュール名が同一です。");
  }
}

function ensureSimpleModuleName(value: string, flagName: string): void {
  if (!/^[A-Za-z0-9._-]+$/.test(value)) {
    throw new Error(`${flagName} は go-boilerplate のような単純なプロジェクト名で指定してください。`);
  }
}

/**
 * リポジトリルートからの相対パスが一括置換の対象かを判定する。
 *
 * 生成物（`*.gen.go` / `*.sql.go` / mock / `openapi.gen.yaml`）は置換後に再生成されるため、
 * また `docs/` と `scripts/setup/` はモジュール名を説明する散文・置換ツール自身であるため除く。
 */
export function isReplacementTarget(relativePath: string): boolean {
  const baseName = path.basename(relativePath);
  const ext = path.extname(relativePath);

  if (EXCLUDED_PATH_PREFIXES.some((prefix) => relativePath.startsWith(prefix))) {
    return false;
  }

  if (EXCLUDED_PATH_SUFFIXES.some((suffix) => relativePath.endsWith(suffix))) {
    return false;
  }

  if (EXCLUDED_BASENAME_PATTERNS.some((pattern) => pattern.test(baseName))) {
    return false;
  }

  if (TARGET_BASENAMES.has(baseName)) {
    return true;
  }

  return TARGET_EXTENSIONS.has(ext);
}

export type ModuleReplacement = {
  content: string;
  occurrences: number;
};

/**
 * 本文中の旧モジュール名をすべて新モジュール名へ置き換える。出現しなければ `null` を返す。
 *
 * 置換は正規表現ではなく文字列分割で行う。モジュール名はインポートパスの一部（`example-api/internal/...`）
 * としても現れるため境界を要求できず、また `$&` などの置換パターンを含む名前を素通しさせないため。
 */
export function replaceModuleOccurrences(
  content: string,
  oldModule: string,
  newModule: string,
): ModuleReplacement | null {
  const parts = content.split(oldModule);

  if (parts.length === 1) {
    return null;
  }

  return { content: parts.join(newModule), occurrences: parts.length - 1 };
}
