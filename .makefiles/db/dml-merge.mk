## DMLマージ関連のコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: merge-dml ## DMLのマージを実行
.PHONY: merge-dml-repo ## ドメイン用DMLのマージ
.PHONY: merge-dml-qs ## クエリサービス用DMLのマージ
.PHONY: merge-dml-sysq ## システムクエリ用DMLのマージ
.PHONY: merge-dml-core ## 指定したタイプのDMLのマージを実行 (例: make merge-dml-core type="repository" work-dir="/app")
# -----CI用ターゲット-----
.PHONY: merge-dml-ci ## DMLのマージを実行（CI用）
.PHONY: merge-dml-ci-repo ## ドメイン用DMLのマージ（CI用）
.PHONY: merge-dml-ci-qs ## クエリサービス用DMLのマージ（CI用）
.PHONY: merge-dml-ci-sysq ## システムクエリ用DMLのマージ（CI用）
.PHONY: merge-dml-ci-core ## 指定したタイプのDMLのマージを実行（CI用） (例: make merge-dml-ci-core type="repository" work-dir="/app")

# -----Dockerコンテナ内で実行するコマンド群-----
merge-dml:
	make merge-dml-repo
	make merge-dml-qs
	make merge-dml-sysq
merge-dml-repo:
	make merge-dml-core type="repository" work-dir="/app"
merge-dml-qs:
	make merge-dml-core type="query_service" work-dir="/app"
merge-dml-sysq:
	make merge-dml-core type="system_query" work-dir="/app"
merge-dml-core:
	@echo "🔄 DMLのマージを実行します... (type=$(type) work-dir=$(work-dir))"
	@docker compose run --rm go_tool_runner make merge-dml-ci-core type="$(type)" work-dir="$(work-dir)"
	@echo "✅ DMLのマージが完了しました。 (type=$(type) work-dir=$(work-dir))"

# -----CI用ターゲット-----
merge-dml-ci:
	make merge-dml-ci-repo work-dir=$(work-dir)
	make merge-dml-ci-qs work-dir=$(work-dir)
	make merge-dml-ci-sysq work-dir=$(work-dir)
merge-dml-ci-repo:
	make merge-dml-ci-core type="repository" work-dir=$(work-dir)
merge-dml-ci-qs:
	make merge-dml-ci-core type="query_service" work-dir=$(work-dir)
merge-dml-ci-sysq:
	make merge-dml-ci-core type="system_query" work-dir=$(work-dir)
merge-dml-ci-core:
	go run cmd/main.go merge-dml --type=$(type) --work-dir=$(work-dir)
