define do-release-tag
	echo "🔄 productionブランチの最新を取得中..."; \
	git fetch origin production; \
	git checkout production; \
	git reset --hard origin/production; \
	echo "✅ 最新のproductionを取得完了"; \
	echo "🔄 最新のタグを取得中..."; \
	make fetch-tags; \
	echo "✅ 最新のタグを取得完了"; \
	echo "🔖 タグから最新タグバージョンを取得: $(1)"; \
	echo "➡️ 次のリリースバージョンを作成: $(2)"; \
	if [ -f .github/release/$(2).md ]; then \
		git tag -a $(2) -F .github/release/$(2).md; \
		git push origin $(2); \
		gh release create $(2) --title "$(2)" --notes-file .github/release/$(2).md; \
		echo "✅ タグを打ちました $(2) on production HEAD"; \
	else \
		echo "❌ .github/release/$(2).md が存在しません。タグとリリースをスキップしました。"; \
	fi
endef

## リリースタグの設定とリリースノートの設定コマンド

.PHONY: release-patch-tag ## リリースタグ(vX.Y.Z+1)を作成
.PHONY: release-minor-tag ## リリースタグ(vX.Y+1.0)を作成
.PHONY: release-major-tag ## リリースタグ(vX+1.0.0)を作成

release-patch-tag:
	@V=$(call get-latest-version); \
	V_NO_V=$$(echo $$V | sed 's/^v//'); \
	major=$$(echo $$V_NO_V | cut -d. -f1); \
	minor=$$(echo $$V_NO_V | cut -d. -f2); \
	patch=$$(echo $$V_NO_V | cut -d. -f3); \
	NEXT=v$$major.$$minor.$$((patch + 1)); \
	$(call do-release-tag,$$V,$$NEXT)

release-minor-tag:
	@V=$(call get-latest-version); \
	V_NO_V=$$(echo $$V | sed 's/^v//'); \
	major=$$(echo $$V_NO_V | cut -d. -f1); \
	minor=$$(echo $$V_NO_V | cut -d. -f2); \
	NEXT=v$$major.$$((minor + 1)).0; \
	$(call do-release-tag,$$V,$$NEXT)

release-major-tag:
	@V=$(call get-latest-version); \
	V_NO_V=$$(echo $$V | sed 's/^v//'); \
	major=$$(echo $$V_NO_V | cut -d. -f1); \
	NEXT=v$$((major + 1)).0.0; \
	$(call do-release-tag,$$V,$$NEXT)
