## Go言語のテスト関連のコマンド群
.PHONY: test ## CI用のテスト実行
.PHONY: test-repo ## テストの実行とテストレポートの生成

test:
	@TGT_PKGS="$$(go list ./... | grep -Ev '/(gen|cli|cmd|mock|apperror)(/|$$)')"; \
	go test $$TGT_PKGS -cover -count=1

test-repo:
	@echo "🔄 テストを実行し、レポートを生成します..."
	@touch docs/coverage/coverage.out
	@TGT_PKGS="$$(go list ./... | grep -Ev '/(gen|cli|cmd|mock|apperror)(/|$$)')"; \
	COVER_PKGS="$$(go list ./... \
		| grep -Ev '/(gen|cli|cmd|mock|apperror)(/|$$)' \
		| tr '\n' ',' \
		| sed 's/,$$//')"; \
	go test $$TGT_PKGS -coverpkg=$$COVER_PKGS -coverprofile=docs/coverage/coverage.out -covermode=atomic  >/dev/null 2>&1
	@go tool cover -html=docs/coverage/coverage.out -o docs/coverage/test-result.html
	@rm -f docs/coverage/coverage.out
	@echo "✅ テストレポートの生成が完了しました。"
