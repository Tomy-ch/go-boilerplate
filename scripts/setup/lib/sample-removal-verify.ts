// サンプル削除の「過不足なし」を判定する規則。verify-sample-removal.ts の入口は
// スナップショット読み込み・git / make / grep の起動・終了コードだけを担う。
//
// このモジュールは検証の成功後に verify-sample-removal.ts 自身と一緒に消える
// （検証ツールはサンプル削除の最終地点であり、コアだけの状態に残してはいけない）。

/** 残留サンプル参照の検出条件。生成物とテストは CI で regen を省くため除外する。 */
export const DANGLING_PATTERN =
  "usercount|userpurge|productimagegc|withdrawalarchive|user_roles|prefecture";
export const DANGLING_EXCLUDE = "_test\\.go|\\.gen\\.go";

/** サンプル削除を起動する make ターゲット。`.mk` のマーカー除去で消えるべきもの。 */
const SAMPLE_MAKE_TARGET = "setup-remove-sample-api";

/** 残留サンプル参照を洗い出す shell コマンド。ヒット無しでも非 0 で落ちないようにする。 */
export function buildDanglingCommand(): string {
  return `grep -rniE '${DANGLING_PATTERN}' internal/ cmd/ --include='*.go' | grep -vE '${DANGLING_EXCLUDE}' || true`;
}

/**
 * remove-sample-api が書き出したスナップショットから登録パスを取り出す。
 *
 * @throws JSON として読めない、または `registeredPaths` が配列でない・空の場合。
 */
export function parseSnapshot(json: string): string[] {
  const parsed: unknown = JSON.parse(json);
  const registeredPaths = (parsed as { registeredPaths?: unknown }).registeredPaths;

  if (!Array.isArray(registeredPaths) || registeredPaths.length === 0) {
    throw new Error("スナップショットの registeredPaths が空です");
  }

  return registeredPaths as string[];
}

/** `git status --porcelain` の出力から削除エントリの相対パスを取り出す。 */
export function parseDeletedPaths(porcelain: string): string[] {
  return porcelain
    .split("\n")
    .filter((line) => line.length > 3 && (line[0] === "D" || line[1] === "D"))
    .map((line) => line.slice(3));
}

/** 不足検出: 登録パスがまだ残っていれば「消えていない」。 */
export function findUnremovedPaths(
  registeredPaths: readonly string[],
  pathExists: (relativePath: string) => boolean,
): string[] {
  return registeredPaths
    .filter((relativePath) => pathExists(relativePath))
    .map((relativePath) => `未削除の登録パス: ${relativePath}`);
}

/** 過剰検出: 登録パスに含まれない削除は想定外（サンプル以外を巻き込んでいる）。 */
export function findUnregisteredDeletions(
  registeredPaths: readonly string[],
  deletedPaths: readonly string[],
): string[] {
  const isRegistered = (deletedPath: string): boolean =>
    registeredPaths.some(
      (registered) => deletedPath === registered || deletedPath.startsWith(`${registered}/`),
    );

  return deletedPaths
    .filter((deletedPath) => !isRegistered(deletedPath))
    .map((deletedPath) => `登録外の削除を検出: ${deletedPath}`);
}

/** 削除ツール自身の make ターゲットが `.mk` のマーカー除去で消えていることを確認する。 */
export function findLeftoverMakeTarget(makeHelpOutput: string): string[] {
  return makeHelpOutput.includes(SAMPLE_MAKE_TARGET)
    ? [`make ターゲット ${SAMPLE_MAKE_TARGET} が残っています`]
    : [];
}

/** 残留サンプル参照（実コード）の grep 結果を失敗メッセージへ変換する。 */
export function findDanglingReferences(danglingHits: string): string[] {
  const hits = danglingHits.trim();

  return hits === "" ? [] : [`残留サンプル参照:\n${hits}`];
}

export type VerificationInput = {
  registeredPaths: readonly string[];
  pathExists: (relativePath: string) => boolean;
  gitStatusPorcelain: string;
  makeHelpOutput: string;
  danglingHits: string;
};

/** 4 種の検査をすべて走らせ、失敗メッセージを 1 本の配列にまとめる。 */
export function collectFailures(input: VerificationInput): string[] {
  return [
    ...findUnremovedPaths(input.registeredPaths, input.pathExists),
    ...findUnregisteredDeletions(input.registeredPaths, parseDeletedPaths(input.gitStatusPorcelain)),
    ...findLeftoverMakeTarget(input.makeHelpOutput),
    ...findDanglingReferences(input.danglingHits),
  ];
}
