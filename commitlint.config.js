// コミットメッセージ規約（CLAUDE.md）を検証する commitlint 設定。
// Type は Feat / CI のように大文字構成が混在するため type-case は課さない。
module.exports = {
  rules: {
    "type-enum": [
      2,
      "always",
      [
        "Feat",
        "Fix",
        "Refactor",
        "Perf",
        "Docs",
        "Test",
        "Build",
        "CI",
        "Chore",
        "Style",
        "Revert",
      ],
    ],
    "type-empty": [2, "never"],
    "subject-empty": [2, "never"],
  },
}
