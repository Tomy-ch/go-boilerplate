import fs from "node:fs";
import path from "node:path";

import { ROOT_DIR } from "./runtime";

/**
 * ファイル本文の変換。書き換え不要なら `null` を返す。
 *
 * @throws 本文が想定外で置換規則を適用できない場合（呼び出し側が中断できるよう投げる）。
 */
export type Transformer = (content: string) => string | null;

export type ListFilesOptions = {
  excludedDirectories?: ReadonlySet<string>;
  /** エントリのフルパスを受け取る。 */
  shouldIncludeFile?: (entryPath: string) => boolean;
};

export function toAbsolutePath(relativePath: string): string {
  return path.join(ROOT_DIR, relativePath);
}

export function toRelativePath(filePath: string): string {
  return path.relative(ROOT_DIR, filePath);
}

/**
 * 変換結果を書き戻し、書き換えたファイルの相対パスを返す。
 *
 * 対象が存在しない場合と変換後が元と同じ場合は書き込まず `null` を返す。
 * `dryRun` では書き込みだけを飛ばすので、戻り値は実行時と一致する。
 */
export function updateFile(
  relativePath: string,
  transformer: Transformer,
  dryRun: boolean,
): string | null {
  return updateAbsoluteFile(toAbsolutePath(relativePath), transformer, dryRun);
}

/** {@link updateFile} の絶対パス版。 */
export function updateAbsoluteFile(
  filePath: string,
  transformer: Transformer,
  dryRun: boolean,
): string | null {
  // 存在確認と読み出しを分けると、その間に消えたファイルで例外になる。読めなければ不在として扱う。
  let original: string;
  try {
    original = fs.readFileSync(filePath, "utf8");
  } catch {
    return null;
  }

  const updated = transformer(original);

  if (updated === null || updated === original) {
    return null;
  }

  if (!dryRun) {
    fs.writeFileSync(filePath, updated);
  }

  return toRelativePath(filePath);
}

/** ディレクトリ配下のファイルを名前順で再帰的に列挙する。 */
export function listFilesRecursive(
  dirPath: string,
  options: ListFilesOptions = {},
  files: string[] = [],
): string[] {
  const excludedDirectories = options.excludedDirectories ?? new Set<string>();
  const shouldIncludeFile = options.shouldIncludeFile ?? (() => true);
  const entries = fs
    .readdirSync(dirPath, { withFileTypes: true })
    .sort((a, b) => a.name.localeCompare(b.name));

  for (const entry of entries) {
    const entryPath = path.join(dirPath, entry.name);

    if (entry.isDirectory()) {
      if (excludedDirectories.has(entry.name)) {
        continue;
      }

      listFilesRecursive(entryPath, options, files);
      continue;
    }

    if (entry.isFile() && shouldIncludeFile(entryPath)) {
      files.push(entryPath);
    }
  }

  return files;
}

/** 直下のファイルだけを名前順で列挙する。`predicate` はファイル名(basename)を受け取る。 */
export function listChildFiles(
  relativeDir: string,
  predicate: (name: string) => boolean = () => true,
): string[] {
  const dirPath = toAbsolutePath(relativeDir);

  return fs
    .readdirSync(dirPath, { withFileTypes: true })
    .filter((entry) => entry.isFile() && predicate(entry.name))
    .map((entry) => path.join(dirPath, entry.name))
    .sort((a, b) => a.localeCompare(b));
}
