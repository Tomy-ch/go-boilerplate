## GHのラベルを操作する
.PHONY: reset-labels ## すべてのラベルを削除（既存ラベル含む）
.PHONY: delete-all-labels ## .github/settings/labels.json に基づいてラベルを作成

delete-all-labels:
	@echo "🗑 既存のラベルを削除します..."
	@gh label list --limit 1000 --json name -q '.[].name' | while read -r label; do \
		echo "🔸 delete label: $$label"; \
		gh label delete "$$label" --yes; \
	done

create-default-labels:
	@echo "🏷 ラベルを作成します..."
	@jq -c '.[]' .github/settings/labels.json | while read -r label; do \
		name=$$(echo $$label | jq -r .name); \
		desc=$$(echo $$label | jq -r .description); \
		color=$$(echo $$label | jq -r .color); \
		echo "🔸 create label: $$name"; \
		gh label create "$$name" --description "$$desc" --color "$$color" || echo "⚠️ $$name already exists"; \
	done
