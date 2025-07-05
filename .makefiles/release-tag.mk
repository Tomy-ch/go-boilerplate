define do-release-tag
	@echo "🔄 productionブランチの最新を取得中..."; \
	git fetch origin production; \
	git checkout production; \
	git reset --hard origin/production; \
	echo "✅ 最新のproductionを取得完了"; \
	echo "🔄 最新のタグを取得中..."; \
	make fetch-tags \
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

.PHONY: release-patch-tag ## productionブランチにリリースタグ(vX.Y.Z+1)を作成
.PHONY: release-minor-tag ## productionブランチにリリースタグ(vX.Y+1.0)を作成
.PHONY: release-major-tag ## productionブランチにリリースタグ(vX+1.0.0)を作成

release-patch-tag:
	$(call do-release-tag,\
		$(call get-latest-version),\
		v$(shell \
			V=$$(echo $(call get-latest-version) | sed 's/^v//'); \
			IFS=. read major minor patch <<< $$V; \
			echo "$$major.$$minor.$$((patch + 1))" \
		) \
	)

release-minor-tag:
	$(call do-release-tag,\
		$(call get-latest-version),\
		v$(shell \
			V=$$(echo $(call get-latest-version) | sed 's/^v//'); \
			IFS=. read major minor patch <<< $$V; \
			echo "$$major.$$((minor + 1)).0" \
		) \
	)

release-major-tag:
	$(call do-release-tag,\
		$(call get-latest-version),\
		v$(shell \
			V=$$(echo $(call get-latest-version) | sed 's/^v//'); \
			IFS=. read major minor patch <<< $$V; \
			echo "$$((major + 1)).0.0" \
		) \
	)
