## Go言語のテスト関連のコマンド群
.PHONY: test ## CI用のテスト実行（キャッシュ無効）
.PHONY: test-cached ## ローカル用テスト実行（キャッシュ有効・pre-commit向け）
.PHONY: gen-test-repo ## テストの実行とテストレポートの生成
.PHONY: test-cover-ci ## CI用のカバレッジ付きテスト実行
.PHONY: cover-gate ## 総カバレッジが閾値以上か検証（CIゲート）
.PHONY: test-scripts ## CI用の scripts 配下ツールのテスト実行（キャッシュ無効）
.PHONY: test-scripts-cached ## ローカル用の scripts 配下ツールのテスト実行（キャッシュ有効・pre-commit向け）

# カバレッジ対象外パッケージ（test / test-cached / gen-test-repo / test-cover-ci で共有）
GO_TEST_EXCLUDE := /(gen|cmd|mock|apperror|scripts)(/|$$)

# カバレッジゲートの下限（docs/rules.md の 90% フロア）
COVERAGE_THRESHOLD := 90

# DB を使うテストはテスト用 DB の seed を fixture として読む。その seed の issuer はスロットのポートに
# 追従する（.makefiles/database/seed.mk）ため、host 実行の go test にも同じ値を渡す。
GO_TEST_ENV = $(LOAD_SLOT); export AUTH_ISSUER="$(AUTH_ISSUER_SH)"; $(if $(GO_TEST_LOAD_ENV),export $(GO_TEST_LOAD_ENV);,)

test:
	@$(GO_TEST_ENV) TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	$(GOBP_NICE) go test $$TGT_PKGS -race -cover -count=1 $(GO_TEST_P_FLAG)

test-cached:
	@$(GO_TEST_ENV) TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	$(GOBP_NICE) go test $$TGT_PKGS -cover $(GO_TEST_P_FLAG)

gen-test-repo:
	@echo "🔄 テストを実行し、レポートを生成します..."
	go clean -testcache
	rm -f docs/coverage/coverage.out
	$(GO_TEST_ENV) TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	COVER_PKGS="$$(go list ./... \
		| grep -Ev '$(GO_TEST_EXCLUDE)' \
		| tr '\n' ',' \
		| sed 's/,$$//')"; \
	go test $$TGT_PKGS -coverpkg=$$COVER_PKGS -coverprofile=docs/coverage/coverage.out -covermode=set  >/dev/null 2>&1
	go tool cover -html=docs/coverage/coverage.out -o docs/coverage/index.html
	rm -f docs/coverage/coverage.out
	@echo "✅ テストレポートの生成が完了しました。"

test-cover-ci:
	@$(GO_TEST_ENV) TGT_PKGS="$$(go list ./... | grep -Ev '$(GO_TEST_EXCLUDE)')"; \
	COVER_PKGS="$$(go list ./... \
		| grep -Ev '$(GO_TEST_EXCLUDE)' \
		| tr '\n' ',' \
		| sed 's/,$$//')"; \
	go test $$TGT_PKGS -race -coverpkg=$$COVER_PKGS -coverprofile=coverage.out -covermode=atomic -count=1

# scripts 配下の開発ツールは GO_TEST_EXCLUDE でカバレッジ母数から外れており、そのままでは
# test / test-cached のいずれにも乗らない。ツール自体がゲート（供給網ピン・lint）なので、
# 壊れ方が「静かに何も検査しなくなる」方向に出る。カバレッジ計測とは切り離して実行だけを足す。
test-scripts:
	@$(GOBP_NICE) go test ./scripts/... -race -count=1 $(GO_TEST_P_FLAG)

test-scripts-cached:
	@$(GOBP_NICE) go test ./scripts/... $(GO_TEST_P_FLAG)

cover-gate:
	@test -f coverage.out || { echo "❌ coverage.out がありません（先に make test-cover-ci を実行）"; exit 1; }
	@total="$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$$NF); print $$NF}')"; \
	awk -v t="$$total" -v th="$(COVERAGE_THRESHOLD)" 'BEGIN { \
		if (t == "") { print "❌ 総カバレッジを取得できません"; exit 1 } \
		if (t+0 < th+0) { printf "❌ 総カバレッジ %.1f%% がしきい値 %d%% を下回っています\n", t, th; exit 1 } \
		printf "✅ 総カバレッジ %.1f%% (しきい値 %d%%)\n", t, th \
	}'
