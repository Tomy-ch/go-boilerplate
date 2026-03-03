## ブランチ保護ルールを設定する
.PHONY: apply-branch-protection ## .github/settings/branch-protection.json を対象リポジトリにPOST

apply-branch-protection:
	@set -e; \
	REPO=$$(gh repo view --json name,owner -q '.owner.login + "/" + .name'); \
	echo "🔧 Applying branch protection rules to $$REPO..."; \
	RESPONSE=$$(mktemp); \
	if ! gh api \
		--method POST \
		-H "Accept: application/vnd.github+json" \
		-H "X-GitHub-Api-Version: 2022-11-28" \
		/repos/$$REPO/rulesets \
		--input .github/settings/branch-protection.json \
		--verbose \
		> $$RESPONSE 2>&1; then \
			echo ""; \
			echo "❌ gh api failed."; \
			echo "------ GitHub API Response ------"; \
			cat $$RESPONSE; \
			echo "----------------------------------"; \
			echo ""; \
			echo "👉 Please check the error above."; \
			echo "👉 If this is an API compatibility issue, please update GitHub CLI (gh) via your package manager."; \
			echo ""; \
			rm -f $$RESPONSE; \
			exit 1; \
	fi; \
	rm -f $$RESPONSE; \
	echo "✅ ブランチルールを $$REPO に適用しました。"
