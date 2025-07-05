.PHONY: apply-branch-protection

apply-branch-protection: ## .github/settings/branch-protection.json を対象リポジトリにPOST
	@REPO=$$(gh repo view --json name,owner -q '.owner.login + "/" + .name'); \
	echo "🔧 Applying branch protection rules to $$REPO..."; \
	gh api \
		--method POST \
		-H "Accept: application/vnd.github+json" \
		/repos/$$REPO/rulesets \
		--input .github/settings/branch-protection.json; \
	echo "✅ ブランチルールを $$REPO に適用しました。"
