## GHのラベルを操作する
.PHONY: create-default-labels ## .github/settings/labels.json に基づいてラベルを作成
.PHONY: delete-all-labels ## すべてのラベルを削除（既存ラベル含む）

delete-all-labels:
	@set -e; \
	echo "🗑 既存のラベルを削除します..."; \
	labels="$$(gh label list --limit 1000 --json name -q '.[].name')"; \
	echo "$$labels" | while read -r label; do \
		[ -z "$$label" ] && continue; \
		echo "🔸 delete label: $$label"; \
		gh label delete "$$label" --yes; \
	done

create-default-labels:
	@set -e; \
	echo "🏷 ラベルを作成します..."; \
	labels_json="$$(jq -c '.[]' .github/settings/labels.json)"; \
	echo "$$labels_json" | while read -r label; do \
		[ -z "$$label" ] && continue; \
		name=$$(printf '%s' "$$label" | jq -r .name); \
		desc=$$(printf '%s' "$$label" | jq -r .description); \
		color=$$(printf '%s' "$$label" | jq -r .color); \
		echo "🔸 upsert label: $$name"; \
		gh label create "$$name" --description "$$desc" --color "$$color" --force; \
	done
