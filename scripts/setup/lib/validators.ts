/**
 * `<owner>/<repo>` 形式を検証する。
 *
 * @throws 形式に合わない場合。
 */
export function ensureRepositoryReference(value: string): void {
  if (!/^[A-Za-z0-9][A-Za-z0-9-]*\/[A-Za-z0-9._-]+$/.test(value)) {
    throw new Error("リポジトリ参照は <owner>/<repo> 形式で指定してください。");
  }
}

/**
 * 4 桁の西暦を検証する。
 *
 * @throws 4 桁の数字でない場合。
 */
export function ensureFourDigitYear(value: string): void {
  if (!/^\d{4}$/.test(value)) {
    throw new Error("--year は 4 桁の西暦で指定してください。");
  }
}

const CODE_OWNER_HANDLE = /^@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?$/;
const CODE_OWNER_EMAIL = /^[^@\s]+@[^@\s.]+(?:\.[^@\s.]+)+$/;

/**
 * CODEOWNERS の所有者表記を検証する。
 *
 * 組織そのもの(@org)は所有者になれないが、ユーザー名と構文が同一なので弾けない。
 *
 * @throws 空、または `@user` / `@org/team` / メールアドレスのいずれでもない値を含む場合。
 */
export function ensureCodeOwners(values: readonly string[]): void {
  if (values.length === 0) {
    throw new Error("--owners には 1 つ以上の所有者を指定してください。");
  }

  for (const value of values) {
    if (CODE_OWNER_EMAIL.test(value)) {
      continue;
    }

    if (!CODE_OWNER_HANDLE.test(value)) {
      throw new Error(
        `所有者 "${value}" が不正です。@user / @org/team / user@example.com のいずれかで指定してください。`,
      );
    }
  }
}
