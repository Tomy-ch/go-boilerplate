export function ensureRepositoryReference(value) {
  if (!/^[A-Za-z0-9][A-Za-z0-9-]*\/[A-Za-z0-9._-]+$/.test(value)) {
    throw new Error("リポジトリ参照は <owner>/<repo> 形式で指定してください。")
  }
}

export function ensureFourDigitYear(value) {
  if (!/^\d{4}$/.test(value)) {
    throw new Error("--year は 4 桁の西暦で指定してください。")
  }
}

const CODE_OWNER_HANDLE = /^@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?$/
const CODE_OWNER_EMAIL = /^[^@\s]+@[^@\s]+\.[^@\s]+$/

// 組織そのもの(@org)は所有者になれないが、ユーザー名と構文が同一なので弾けない。
export function ensureCodeOwners(values) {
  if (values.length === 0) {
    throw new Error("--owners には 1 つ以上の所有者を指定してください。")
  }

  for (const value of values) {
    if (CODE_OWNER_EMAIL.test(value)) {
      continue
    }

    if (!CODE_OWNER_HANDLE.test(value)) {
      throw new Error(
        `所有者 "${value}" が不正です。@user / @org/team / user@example.com のいずれかで指定してください。`
      )
    }
  }
}
