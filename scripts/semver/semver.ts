/** 次のバージョンを決めるときの繰り上げ単位。 */
export type BumpType = "patch" | "minor" | "major";

const BUMP_TYPES: readonly string[] = ["patch", "minor", "major"];

// 先頭ゼロ（`01.2.3`）を弾く SemVer の数値表記。タグ名の揺れをそのまま次版へ持ち込ませない。
const SEMVER = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

export function isBumpType(value: string): value is BumpType {
  return BUMP_TYPES.includes(value);
}

/**
 * 先頭の `v` を落とした `X.Y.Z` を返す。
 *
 * @throws SemVer の数値表記でない場合。
 */
export function normalizeVersion(version: string): string {
  const normalized = version.replace(/^v/, "");

  if (!SEMVER.test(normalized)) {
    throw new Error("version must be in the format X.Y.Z (optional leading 'v')");
  }

  return normalized;
}

/**
 * 繰り上げ後のバージョンを `vX.Y.Z` で返す。
 *
 * @throws 引数が SemVer の数値表記でない場合。
 */
export function bumpVersion(version: string, type: BumpType): string {
  const [major, minor, patch] = normalizeVersion(version).split(".").map(Number) as [
    number,
    number,
    number,
  ];

  switch (type) {
    case "patch":
      return `v${major}.${minor}.${patch + 1}`;
    case "minor":
      return `v${major}.${minor + 1}.0`;
    case "major":
      return `v${major + 1}.0.0`;
  }
}

/** 引数の解釈結果。`error` はそのまま利用者へ見せる文言。 */
export type ParsedArgs =
  | { ok: true; version: string; type: BumpType }
  | { ok: false; error: string };

/**
 * CLI 引数を解釈する。
 *
 * @remarks
 * 版と種別を別々に検査し、足りない方を名指しします。まとめて「usage」とだけ返すと、
 * どちらを直せばよいかが呼び出し側のログから読めません。
 */
export function parseArgs(argv: readonly string[]): ParsedArgs {
  const [version, type] = argv;

  if (version === undefined || version === "") {
    return { ok: false, error: "version is required" };
  }

  if (type === undefined || !isBumpType(type)) {
    return { ok: false, error: "type must be patch | minor | major" };
  }

  return { ok: true, version, type };
}
