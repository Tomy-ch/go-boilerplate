## リポジトリの初期化
.PHONY: setup-repo ## リポジトリの初期化
.PHONY: setup-replace-module ## node_tool_runnerでGoモジュール名の一括置換を実行
.PHONY: setup-replace-app-metadata ## node_tool_runnerでAPP_NAMEやOpenAPIタイトルの置換を実行
.PHONY: setup-replace-repository-reference ## node_tool_runnerでリポジトリ参照の置換を実行
.PHONY: setup-replace-license-copyright ## node_tool_runnerでLICENSEの著作権表示更新を実行
.PHONY: setup-remove-sample-api ## サンプルAPI(user/product/order)を一括削除し再生成・検証まで実行 # sample-api:line

SETUP_DRY_RUN_FLAG := $(if $(DRY_RUN),--dry-run,)

setup-repo:
	@if [ -n "$(DRY_RUN)" ]; then echo "❌ setup-repo は DRY_RUN 未対応です（ローカル/リモートを破壊的に変更します）。DRY_RUN を外して実行してください。"; exit 1; fi
	@echo "🔧 設定を確認中..."

	@if git rev-parse --verify refs/tags/v0.0.0 >/dev/null 2>&1; then \
		echo "❌ タグ 【v0.0.0】 があります。初期化を停止します。"; exit 1; \
	fi

	@echo "✅ 初期化を開始します"

	@echo "🔧 ghコマンドのログインを開始します..."
	@$(MAKE) gh-login
	@echo "✅ ghコマンドのログインが完了しました。"

	@echo "🔧 タグの初期化を開始します..."
	@TAGS=$$(git tag); \
	if [ -n "$$TAGS" ]; then \
		for tag in $$TAGS; do \
			git tag -d $$tag; \
			git push origin :refs/tags/$$tag || true; \
		done; \
		echo "🧹 すべてのタグを削除しました。"; \
	else \
		echo "🟡 削除対象のタグが存在しません。"; \
	fi
	@echo "✅ タグの初期化を終了します。"

	@echo "🔧 v0.0.0のタグ打ちを開始します..."
	@git tag -a v0.0.0 -m "Initial boilerplate tag"
	@git push origin v0.0.0
	@echo "✅ v0.0.0のタグ打ちが完了しました。"

	@echo "🔧 ブランチ作成を開始します..."
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

	@git push origin develop staging production
	@echo "✅ ブランチの作成を終了します。"

	@echo "🔧 デフォルトブランチの設定を開始します..."
	@REPO=$$(gh repo view --json name,owner -q '.owner.login + "/" + .name'); \
		gh api -X PATCH repos/$$REPO -f default_branch=production

	@git fetch --prune
	@ORIGINAL_BRANCH=$$(git branch --show-current); \
	git checkout production; \
	if echo $$ORIGINAL_BRANCH | grep -q "release/"; then \
		git branch -D $$ORIGINAL_BRANCH; \
		git push origin --delete $$ORIGINAL_BRANCH || true; \
	fi
	@echo "✅ デフォルトブランチの設定を終了します。"

	@echo "🔧 ルールセットの適用を開始します..."
	@$(MAKE) apply-branch-protection
	@echo "✅ ルールセットの適用を終了します。"

	@echo "🔧 ラベルの初期化を開始します..."
	@$(MAKE) delete-all-labels
	@$(MAKE) create-default-labels
	@echo "✅ ラベルの初期化を終了します。"

	@echo "🔧 リリースノートの初期化を開始します..."
	@if [ -d ".github/release" ]; then \
		find .github/release -type f ! -name "v0.0.0.md" -delete; \
		echo "🧹 v0.0.0.md 以外のリリースノートを削除しました。"; \
	else \
		echo "🟡 .github/release ディレクトリが存在しないためスキップします。"; \
	fi
	@echo "✅ リリースノートの初期化を終了します。"

	@git remote remove upstream || true
	@echo "✅ Initialization complete. Default branch: production"

setup-replace-module:
	@if [ -z "$(OLD_MODULE)" ] || [ -z "$(NEW_MODULE)" ]; then \
		echo "❌ OLD_MODULE と NEW_MODULE を指定してください。例: make setup-replace-module OLD_MODULE=go-boilerplate NEW_MODULE=example-api"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-module.mjs $(OLD_MODULE) $(NEW_MODULE) $(SETUP_DRY_RUN_FLAG)

setup-replace-app-metadata:
	@if [ -z "$(APP_NAME)" ] || [ -z "$(OPENAPI_TITLE)" ] || [ -z "$(COPILOT_TITLE)" ]; then \
		echo "❌ APP_NAME, OPENAPI_TITLE, COPILOT_TITLE を指定してください。"; \
		echo "例: make setup-replace-app-metadata APP_NAME='Example API' OPENAPI_TITLE='Example API with Onion Architecture' COPILOT_TITLE='example-api Copilot Instructions'"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-app-metadata.mjs \
		--app-name "$(APP_NAME)" \
		--openapi-title "$(OPENAPI_TITLE)" \
		--copilot-title "$(COPILOT_TITLE)" \
		$(SETUP_DRY_RUN_FLAG)

setup-replace-repository-reference:
	@if [ -z "$(REPOSITORY)" ]; then \
		echo "❌ REPOSITORY を指定してください。例: make setup-replace-repository-reference REPOSITORY=example-org/example-api"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-repository-reference.mjs $(REPOSITORY) $(SETUP_DRY_RUN_FLAG)

setup-replace-license-copyright:
	@if [ -z "$(COPYRIGHT_HOLDER)" ]; then \
		echo "❌ COPYRIGHT_HOLDER を指定してください。例: make setup-replace-license-copyright COPYRIGHT_HOLDER='Example Inc.' COPYRIGHT_YEAR=2026"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-license-copyright.mjs \
		--holder "$(COPYRIGHT_HOLDER)" \
		$(if $(COPYRIGHT_YEAR),--year $(COPYRIGHT_YEAR),) \
		$(SETUP_DRY_RUN_FLAG)

# sample-api:begin
# サンプルAPIの削除はコンテナ内（node_tool_runner）で行い、削除後の再生成・整形・検証・DB 再構築は
# Go ツールチェーンが必要なためホスト側の make ターゲットを連鎖させる。
# プレビューは DRY_RUN=1 を付ける（削除も再生成も行わない）。
# make は起動時に makefile を全読込するため、手順1の node がこの .mk からターゲットを strip（自消滅）
# しても、実行中のレシピは継続し regen まで走る。
setup-remove-sample-api:
	@docker compose run --rm node_tool_runner node scripts/setup/remove-sample-api.mjs $(SETUP_DRY_RUN_FLAG)
	@if [ -n "$(DRY_RUN)" ]; then \
		echo "🟡 DRY_RUN のため再生成・整形・検証はスキップしました。"; \
	else \
		echo "🔧 mock-auth-server の固定ユーザーを中立な既定へリセットします..."; \
		docker compose run --rm node_tool_runner node scripts/setup/reset-mock-auth-users.mjs; \
		echo "🔧 再生成・整形・検証・DB 再構築を実行します..."; \
		$(MAKE) db-local-reinit db-test-reinit; \
		$(MAKE) gen-api gen-query fix lint; \
		echo "✅ サンプルAPIの削除・再生成・検証が完了しました。"; \
	fi
# sample-api:end
