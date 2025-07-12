#!/bin/bash

FILES=$(find .makefiles -name '*.mk')

echo "📦 Makeターゲット一覧"
echo "-------------------------------------------"

for file in $FILES; do
  current_category=""
  while IFS= read -r line; do
    if [[ "$line" =~ ^##\ (.*) ]]; then
      # カテゴリ見出し行
      current_category="${BASH_REMATCH[1]}"
      echo ""
      echo "📂 $current_category"
    elif [[ "$line" =~ ^\.PHONY:\ ([^[:space:]]+)[[:space:]]*##[[:space:]]*(.*)$ ]]; then
      # .PHONY 行（単一ターゲット + コメント付き）
      target="${BASH_REMATCH[1]}"
      comment="${BASH_REMATCH[2]}"
      printf "🛠  %-24s %s\n" "$target" "$comment"
    fi
  done < "$file"
done
