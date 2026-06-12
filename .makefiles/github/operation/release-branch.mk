GET_LATEST_VERSION_CMD := git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1

define do-generate-from-branch
	LATEST=$(1); \
	NEXT=$(2); \
	BASE_BRANCH=$(3); \
	BRANCH_PREFIX=$(4); \
	BRANCH_NAME=$$BRANCH_PREFIX/$$NEXT; \
	echo "🔖 タグから最新リリースバージョンを取得: 【 $$LATEST 】"; \
	echo "➡️ 次のリリースバージョンを作成: 【 $$NEXT 】"; \
	echo "🌱 ブランチを作成: $$BASE_BRANCH → 【 $$BRANCH_NAME 】"; \
	if git ls-remote --exit-code --heads origin $$BRANCH_NAME > /dev/null; then \
		echo "❌ ブランチ【 $$BRANCH_NAME 】は既に存在します。処理を中止します。"; \
		exit 1; \
	fi; \
	STATUS=$$(git status --porcelain); \
	if [ -n "$$STATUS" ]; then \
		echo "❌ 作業ツリーに未コミットの変更があります。変更をコミットまたは退避してから再実行してください。"; \
		git status --short; \
		exit 1; \
	fi; \
	git fetch origin $$BASE_BRANCH && \
	git checkout -b $$BRANCH_NAME origin/$$BASE_BRANCH && \
	git push origin $$BRANCH_NAME && \
	echo "⚙️ GitHub上のデフォルトブランチを $$BRANCH_NAME に設定します。" && \
	gh repo edit --default-branch $$BRANCH_NAME && \
	echo "✅ デフォルトブランチを $$BRANCH_NAME に切り替えて、プッシュしました。"
endef

## リリースブランチの切り替えコマンド

.PHONY: hotfix-patch ## hotfixブランチ(vX.Y.Z+1)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: branch-patch ## releaseブランチ(vX.Y.Z+1)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: branch-minor ## releaseブランチ(vX.Y+1.0)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: branch-major ## releaseブランチ(vX+1.0.0)を作成して、デフォルトブランチに設定(現在のタグ基準)

hotfix-patch:
	@echo "🔄 最新のタグを取得中..."; \
	git fetch --tags origin; \
	echo "✅ 最新のタグを取得完了"; \
	V=$$(${GET_LATEST_VERSION_CMD}); \
	if [ -z "$$V" ]; then \
		echo "❌ 最新のリリースタグを取得できませんでした。初期タグ作成が必要です。"; \
		echo "➡️ 先に make release-tag などで初期タグを作成してから再実行してください。"; \
		exit 1; \
	fi; \
	NEXT=$$(docker compose run --rm node_tool_runner node scripts/semver.mjs $$V patch); \
	$(call do-generate-from-branch,$$V,$$NEXT,production,hotfix)

branch-patch:
	@echo "🔄 最新のタグを取得中..."; \
	git fetch --tags origin; \
	echo "✅ 最新のタグを取得完了"; \
	V=$$(${GET_LATEST_VERSION_CMD}); \
	if [ -z "$$V" ]; then \
		echo "❌ 最新のリリースタグを取得できませんでした。初期タグ作成が必要です。"; \
		echo "➡️ 先に make release-tag などで初期タグを作成してから再実行してください。"; \
		exit 1; \
	fi; \
	NEXT=$$(docker compose run --rm node_tool_runner node scripts/semver.mjs $$V patch); \
	$(call do-generate-from-branch,$$V,$$NEXT,production,release)

branch-minor:
	@echo "🔄 最新のタグを取得中..."; \
	git fetch --tags origin; \
	echo "✅ 最新のタグを取得完了"; \
	V=$$(${GET_LATEST_VERSION_CMD}); \
	if [ -z "$$V" ]; then \
		echo "❌ 最新のリリースタグを取得できませんでした。初期タグ作成が必要です。"; \
		echo "➡️ 先に make release-tag などで初期タグを作成してから再実行してください。"; \
		exit 1; \
	fi; \
	NEXT=$$(docker compose run --rm node_tool_runner node scripts/semver.mjs $$V minor); \
	$(call do-generate-from-branch,$$V,$$NEXT,production,release)

branch-major:
	@echo "🔄 最新のタグを取得中..."; \
	git fetch --tags origin; \
	echo "✅ 最新のタグを取得完了"; \
	V=$$(${GET_LATEST_VERSION_CMD}); \
	if [ -z "$$V" ]; then \
		echo "❌ 最新のリリースタグを取得できませんでした。初期タグ作成が必要です。"; \
		echo "➡️ 先に make release-tag などで初期タグを作成してから再実行してください。"; \
		exit 1; \
	fi; \
	NEXT=$$(docker compose run --rm node_tool_runner node scripts/semver.mjs $$V major); \
	$(call do-generate-from-branch,$$V,$$NEXT,production,release)
