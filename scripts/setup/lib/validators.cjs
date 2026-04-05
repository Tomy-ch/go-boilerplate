function ensureRepositoryReference(value) {
  if (!/^[^/\s]+\/[^/\s]+$/.test(value)) {
    throw new Error("リポジトリ参照は <owner>/<repo> 形式で指定してください。")
  }
}

function ensureFourDigitYear(value) {
  if (!/^\d{4}$/.test(value)) {
    throw new Error("--year は 4 桁の西暦で指定してください。")
  }
}

module.exports = {
  ensureRepositoryReference,
  ensureFourDigitYear
}
