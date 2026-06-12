## Go言語のテスト関連のコマンド群
.PHONY: test ## CI用のテスト実行（キャッシュ無効）
.PHONY: test-cached ## ローカル用テスト実行（キャッシュ有効・pre-commit向け）
.PHONY: gen-test-repo ## テストの実行とテストレポートの生成
.PHONY: test-cover-ci ## CI用のカバレッジ付きテスト実行

# カバレッジ対象外パッケージ（test / test-cached / gen-test-repo / test-cover-ci で共有）
GO_TEST_EXCLUDE := /(gen|cmd|mock|apperror|scripts)(/|$$)

test:
	@TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	go test $$TGT_PKGS -cover -count=1

test-cached:
	@TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	go test $$TGT_PKGS -cover

gen-test-repo:
	@echo "🔄 テストを実行し、レポートを生成します..."
	go clean -testcache
	rm -f docs/coverage/coverage.out
	TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	COVER_PKGS="$$(go list ./... \
		| grep -Ev '$(GO_TEST_EXCLUDE)' \
		| tr '\n' ',' \
		| sed 's/,$$//')"; \
	go test $$TGT_PKGS -coverpkg=$$COVER_PKGS -coverprofile=docs/coverage/coverage.out -covermode=atomic  >/dev/null 2>&1
	go tool cover -html=docs/coverage/coverage.out -o docs/coverage/index.html
	rm -f docs/coverage/coverage.out
	@echo "✅ テストレポートの生成が完了しました。"

test-cover-ci:
	@TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	COVER_PKGS="$$(go list ./... \
		| grep -Ev '$(GO_TEST_EXCLUDE)' \
		| tr '\n' ',' \
		| sed 's/,$$//')"; \
	go test $$TGT_PKGS -coverpkg=$$COVER_PKGS -coverprofile=coverage.out -covermode=atomic -count=1
