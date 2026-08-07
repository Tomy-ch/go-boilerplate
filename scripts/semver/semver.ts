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
