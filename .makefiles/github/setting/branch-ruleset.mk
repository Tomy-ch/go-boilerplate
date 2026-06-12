## ブランチ保護ルールを設定する
.PHONY: apply-branch-protection ## .github/settings/branch-protection.json を対象リポジトリにPOST

apply-branch-protection:
	@set -e; \
	REPO=$$(gh repo view --json name,owner -q '.owner.login + "/" + .name'); \
	echo "🔧 ブランチ保護ルールを $$REPO に適用します..."; \
	RULESET_NAME=$$(jq -r .name .github/settings/branch-protection.json); \
	EXISTING_ID=$$(gh api /repos/$$REPO/rulesets -q ".[] | select(.name==\"$$RULESET_NAME\") | .id" | head -n1); \
	if [ -n "$$EXISTING_ID" ]; then \
		METHOD=PUT; URL=/repos/$$REPO/rulesets/$$EXISTING_ID; \
	else \
		METHOD=POST; URL=/repos/$$REPO/rulesets; \
	fi; \
	RESPONSE=$$(mktemp); \
	trap 'rm -f "$$RESPONSE"' EXIT; \
	if ! gh api \
		--method $$METHOD \
		-H "Accept: application/vnd.github+json" \
		-H "X-GitHub-Api-Version: 2022-11-28" \
		"$$URL" \
		--input .github/settings/branch-protection.json \
		> "$$RESPONSE" 2>&1; then \
			echo ""; \
			echo "❌ gh api の実行に失敗しました。"; \
			echo "------ GitHub API Response ------"; \
			cat "$$RESPONSE"; \
			echo "----------------------------------"; \
			echo "👉 上記レスポンスを確認してください（権限/認証は gh auth status、API 非互換は gh の更新を検討）。"; \
			exit 1; \
	fi; \
	echo "✅ ブランチルールを $$REPO に適用しました。"
