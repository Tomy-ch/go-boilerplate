.PHONY: init-repo

init-repo: ## リポジトリの初期化
	@echo "🔧 設定を確認中..."

	@if git rev-parse --verify refs/tags/v0.0.0 >/dev/null 2>&1; then \
		echo "❌ タグ 【v0.0.0】 があります。初期化を停止します。"; exit 1; \
	fi

	@echo "✅ 初期化を開始します"

	@echo "🔧 ghコマンドのログインを開始します..."
	@make gh-login
	@echo "✅ ghコマンドのログインが完了しました。"

	@echo "🔧 v0.0.0のタグ打ちを開始します..."
	@git tag -a v0.0.0 -m "Initial boilerplate tag"
	@git push origin v0.0.0
	@echo "✅ v0.0.0のタグ打ちが完了しました。"

	@echo "🔧 ブランチ作成を開始します..."
	@if git show-ref --verify --quiet refs/heads/release/v0.1.0; then \
		echo "🟡 ブランチ 【release/v0.1.0】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch release/v0.1.0; \
	fi

	@if git show-ref --verify --quiet refs/heads/develop; then \
		echo "🟡 ブランチ 【develop】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch develop; \
	fi

	@if git show-ref --verify --quiet refs/heads/staging; then \
		echo "🟡 ブランチ 【staging】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch staging; \
	fi

	@if git show-ref --verify --quiet refs/heads/production; then \
		echo "🟡 ブランチ 【production】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch production; \
	fi

	@git push origin release/v0.1.0 develop staging production
	@echo "✅ ブランチの作成を終了します。"

	@echo "🔧 デフォルトブランチの設定を開始します..."
	@REPO=$$(gh repo view --json name,owner -q '.owner.login + "/" + .name'); \
		gh api -X PATCH repos/$$REPO -f default_branch=release/v0.1.0

	@git fetch --prune
	@git checkout release/v0.1.0
	@echo "✅ デフォルトブランチの設定を終了します。"

	@echo "🔧 ルールセットの適用を開始します..."
	@make apply-branch-protection
	@echo "✅ ルールセットの適用を終了します。"

	@echo "✅ Initialization complete. Default branch: release/v0.1.0"
