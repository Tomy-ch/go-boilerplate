define do-generate-from-branch
	echo "🔄 最新のタグを取得中..."; \
	make fetch-tags; \
	echo "✅ 最新のタグを取得完了"; \
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
	git fetch origin $$BASE_BRANCH; \
	git checkout -b $$BRANCH_NAME origin/$$BASE_BRANCH; \
	git push origin $$BRANCH_NAME; \
	echo "⚙️ GitHub上のデフォルトブランチを $$BRANCH_NAME に設定します。"; \
	gh repo edit --default-branch $$BRANCH_NAME; \
	echo "✅ デフォルトブランチを $$BRANCH_NAME に切り替えて、プッシュしました。"
endef

.PHONY: hotfix-patch-branch ## productionブランチからhotfixブランチ(vX.Y.Z+1)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: release-patch-branch ## productionブランチからreleaseブランチ(vX.Y.Z+1)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: release-minor-branch ## productionブランチからreleaseブランチ(vX.Y+1.Z)を作成して、デフォルトブランチに設定(現在のタグ基準)
.PHONY: release-major-branch ## productionブランチからreleaseブランチ(vX+1.Y.Z)を作成して、デフォルトブランチに設定(現在のタグ基準)

hotfix-patch-branch:
	@V=$(call get-latest-version); \
	V_NO_V=$$(echo $$V | sed 's/^v//'); \
	major=$$(echo $$V_NO_V | cut -d. -f1); \
	minor=$$(echo $$V_NO_V | cut -d. -f2); \
	patch=$$(echo $$V_NO_V | cut -d. -f3); \
	NEXT=v$$major.$$minor.$$((patch + 1)); \
	$(call do-generate-from-branch,$$V,$$NEXT,production,hotfix)

release-patch-branch:
	@V=$(call get-latest-version); \
	V_NO_V=$$(echo $$V | sed 's/^v//'); \
	major=$$(echo $$V_NO_V | cut -d. -f1); \
	minor=$$(echo $$V_NO_V | cut -d. -f2); \
	patch=$$(echo $$V_NO_V | cut -d. -f3); \
	NEXT=v$$major.$$minor.$$((patch + 1)); \
	$(call do-generate-from-branch,$$V,$$NEXT,production,release)

release-minor-branch:
	@V=$(call get-latest-version); \
	V_NO_V=$$(echo $$V | sed 's/^v//'); \
	major=$$(echo $$V_NO_V | cut -d. -f1); \
	minor=$$(echo $$V_NO_V | cut -d. -f2); \
	NEXT=v$$major.$$((minor + 1)).0; \
	$(call do-generate-from-branch,$$V,$$NEXT,production,release)

release-major-branch:
	@V=$(call get-latest-version); \
	V_NO_V=$$(echo $$V | sed 's/^v//'); \
	major=$$(echo $$V_NO_V | cut -d. -f1); \
	NEXT=v$$((major + 1)).0.0; \
	$(call do-generate-from-branch,$$V,$$NEXT,production,release)
