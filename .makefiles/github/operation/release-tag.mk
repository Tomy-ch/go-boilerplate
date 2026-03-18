get-latest-version = $(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1)

define do-release-tag
	echo "🔄 productionブランチの最新を取得中..."; \
	git fetch origin production; \
	git checkout production; \
	git reset --hard origin/production; \
	echo "✅ 最新のproductionを取得完了"; \
	echo "🔄 最新のタグを取得中..."; \
	git fetch --tags origin; \
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

.PHONY: tag-patch ## リリースタグ(vX.Y.Z+1)を作成
.PHONY: tag-minor ## リリースタグ(vX.Y+1.0)を作成
.PHONY: tag-major ## リリースタグ(vX+1.0.0)を作成

tag-patch:
	@V=$(call get-latest-version); \
	NEXT=$$(docker compose run --rm node_tool_runner node scripts/semver.js $$V patch); \
	$(call do-release-tag,$$V,$$NEXT)

tag-minor:
	@V=$(call get-latest-version); \
	NEXT=$$(docker compose run --rm node_tool_runner node scripts/semver.js $$V minor); \
	$(call do-release-tag,$$V,$$NEXT)

tag-major:
	@V=$(call get-latest-version); \
	NEXT=$$(docker compose run --rm node_tool_runner node scripts/semver.js $$V major); \
	$(call do-release-tag,$$V,$$NEXT)
