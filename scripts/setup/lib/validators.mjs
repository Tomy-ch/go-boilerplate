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
