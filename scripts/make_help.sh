#!/bin/bash
set -euo pipefail

echo "📦 Makeターゲット一覧"
echo "-------------------------------------------"

while IFS= read -r file; do
  while IFS= read -r line; do
    if [[ "$line" =~ ^##\ (.*) ]]; then
      # カテゴリ見出し行
      echo ""
      echo "📂 ${BASH_REMATCH[1]}"
    elif [[ "$line" =~ ^\.PHONY:\ ([^[:space:]]+)[[:space:]]*##[[:space:]]*(.*)$ ]]; then
      # .PHONY 行（単一ターゲット + コメント付き）
      printf "🛠  %-24s %s\n" "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
    elif [[ "$line" =~ ^\.PHONY: ]]; then
      # 説明（## ...）が無い / 形式不一致の .PHONY 行は help に出ないため警告する
      echo "⚠️  $file: 説明コメント(## ...)の無い .PHONY 行をスキップしました: $line" >&2
    fi
  done < "$file"
done < <(find .makefiles -name '*.mk' | sort)
