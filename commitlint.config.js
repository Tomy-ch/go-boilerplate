// コミットメッセージ規約（CLAUDE.md）を検証する commitlint 設定。
// 既定パーサで `<Type>: <subject>` を解析し、Type を許可リストに厳密一致で検査する。
// Type は Feat / CI のように大文字構成が混在するため type-case は課さず enum 厳密一致のみとする。
// Merge / Revert 等の自動生成メッセージは commitlint の defaultIgnores が無視する。
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
