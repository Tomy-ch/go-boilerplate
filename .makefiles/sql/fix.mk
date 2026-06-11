## SQLに対するFixコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: sql-fix ## SQLの自動修正を一括実行
.PHONY: sql-fix-migrations ## マイグレーションのSQLの自動修正を実行
.PHONY: sql-fix-dml ## DML系SQLの自動修正を実行
.PHONY: sql-fix-seed ## シードデータSQLの自動修正を実行
# -----CI内で実行するコマンド群-----
.PHONY: sql-fix-ci ## 全カテゴリのSQL自動修正を1コンテナで実行(CI用)
.PHONY: sql-fix-migrations-ci ## マイグレーションのSQLの自動修正を実行(CI用)
.PHONY: sql-fix-dml-ci ## DML系SQLの自動修正を実行(CI用)
.PHONY: sql-fix-seed-ci ## シードデータSQLの自動修正を実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
sql-fix:
	@docker compose run --rm python_tool_runner make sql-fix-ci

sql-fix-migrations:
	@docker compose run --rm python_tool_runner make sql-fix-migrations-ci
sql-fix-dml:
	@docker compose run --rm python_tool_runner make sql-fix-dml-ci
sql-fix-seed:
	@docker compose run --rm python_tool_runner make sql-fix-seed-ci

# -----CI内で実行するコマンド群-----
sql-fix-ci: sql-fix-migrations-ci sql-fix-dml-ci sql-fix-seed-ci

sql-fix-migrations-ci:
	sqlfluff fix database/migrations/ --config docker/database/sqlfluff/.migrations.sqlfluff
sql-fix-dml-ci:
	sqlfluff fix database/dml/ --config docker/database/sqlfluff/.dml.sqlfluff
sql-fix-seed-ci:
	sqlfluff fix database/seed/ --config docker/database/sqlfluff/.seed.sqlfluff

