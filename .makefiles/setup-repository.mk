.PHONY: init-repo

init-repo: ## リポジトリの初期化
	@echo "🔧 Checking existing setup..."

	@if git rev-parse --verify refs/tags/v0.0.0 >/dev/null 2>&1; then \
		echo "❌ Tag v0.0.0 already exists. Aborting."; exit 1; \
	fi

	@echo "✅ Initial setup can proceed"

	# gh-loginへの明示的ログイン
	@make gh-login

	# 初期版のタグの生成
	@git tag -a v0.0.0 -m "Initial boilerplate tag"
	@git push origin v0.0.0

	# ブランチの作成
	@if git show-ref --verify --quiet refs/heads/release/v0.1.0; then \
		echo "🟡 Branch release/v0.1.0 already exists. Skipping."; \
	else \
		git branch release/v0.1.0; \
	fi

	@if git show-ref --verify --quiet refs/heads/develop; then \
		echo "🟡 Branch develop already exists. Skipping."; \
	else \
		git branch develop; \
	fi

	@if git show-ref --verify --quiet refs/heads/staging; then \
		echo "🟡 Branch staging already exists. Skipping."; \
	else \
		git branch staging; \
	fi

	@if git show-ref --verify --quiet refs/heads/production; then \
		echo "🟡 Branch production already exists. Skipping."; \
	else \
		git branch production; \
	fi

	@git push origin release/v0.1.0 develop staging production

	@REPO=$$(gh repo view --json name,owner -q '.owner.login + "/" + .name'); \
		gh api -X PATCH repos/$$REPO -f default_branch=release/v0.1.0

	@git fetch --prune
	@git checkout release/v0.1.0

	@echo "✅ Initialization complete. Default branch: release/v0.1.0"