## DMLマージ関連のコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: merge-dml ## DMLのマージを実行
.PHONY: merge-dml-repo ## ドメイン用DMLのマージ
.PHONY: merge-dml-qs ## クエリサービス用DMLのマージ
.PHONY: merge-dml-cs ## コマンドサービス用DMLのマージ
.PHONY: merge-dml-sysq ## システムクエリ用DMLのマージ
.PHONY: merge-dml-core ## 指定したタイプのDMLのマージを実行 (例: make merge-dml-core type="repository" work-dir=".")
# -----CI用ターゲット-----
.PHONY: merge-dml-ci ## DMLのマージを実行（CI用）
.PHONY: merge-dml-ci-repo ## ドメイン用DMLのマージ（CI用）
.PHONY: merge-dml-ci-qs ## クエリサービス用DMLのマージ（CI用）
.PHONY: merge-dml-ci-cs ## コマンドサービス用DMLのマージ（CI用）
.PHONY: merge-dml-ci-sysq ## システムクエリ用DMLのマージ（CI用）
.PHONY: merge-dml-ci-core ## 指定したタイプのDMLのマージを実行（CI用） (例: make merge-dml-ci-core type="repository" work-dir=".")

# -----Dockerコンテナ内で実行するコマンド群-----
merge-dml:
	@docker compose run --rm go_tool_runner make merge-dml-ci work-dir="."
merge-dml-repo:
	$(MAKE) merge-dml-core type="repository" work-dir="."
merge-dml-qs:
	$(MAKE) merge-dml-core type="query_service" work-dir="."
merge-dml-sysq:
	$(MAKE) merge-dml-core type="system_query" work-dir="."
merge-dml-cs:
	$(MAKE) merge-dml-core type="command_service" work-dir="."
merge-dml-core:
	@echo "🔄 DMLのマージを実行します... (type=$(type) work-dir=$(work-dir))"
	@docker compose run --rm go_tool_runner make merge-dml-ci-core type="$(type)" work-dir="$(work-dir)"
	@echo "✅ DMLのマージが完了しました。 (type=$(type) work-dir=$(work-dir))"

# -----CI用ターゲット-----
merge-dml-ci:
	$(MAKE) merge-dml-ci-repo work-dir=$(work-dir)
	$(MAKE) merge-dml-ci-qs work-dir=$(work-dir)
	$(MAKE) merge-dml-ci-sysq work-dir=$(work-dir)
	$(MAKE) merge-dml-ci-cs work-dir=$(work-dir)
merge-dml-ci-repo:
	$(MAKE) merge-dml-ci-core type="repository" work-dir=$(work-dir)
merge-dml-ci-qs:
	$(MAKE) merge-dml-ci-core type="query_service" work-dir=$(work-dir)
merge-dml-ci-sysq:
	$(MAKE) merge-dml-ci-core type="system_query" work-dir=$(work-dir)
merge-dml-ci-cs:
	$(MAKE) merge-dml-ci-core type="command_service" work-dir=$(work-dir)
merge-dml-ci-core:
	go run ./cmd/ merge-dml --type=$(type) --work-dir=$(work-dir)
