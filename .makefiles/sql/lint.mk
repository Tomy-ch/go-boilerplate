## SQLに対するLintコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: sql-lint ## SQLのLintを一括実行
.PHONY: sql-lint-migrations ## マイグレーションのSQLのLintを実行
.PHONY: sql-lint-dml ## DML系SQLのLintを実行
.PHONY: sql-lint-seed ## シードデータSQLのLintを実行
# -----CI内で実行するコマンド群-----
.PHONY: sql-lint-migrations-ci ## マイグレーションのSQLのLintを実行(CI用)
.PHONY: sql-lint-dml-ci ## DML系SQLのLintを実行(CI用)
.PHONY: sql-lint-seed-ci ## シードデータSQLのLintを実行(CI用)

# -----Dockerコンテナ内で実行するコマンド群-----
sql-lint:
	@make sql-lint-migrations
	@make sql-lint-dml
	@make sql-lint-seed

sql-lint-migrations:
	@docker compose run --rm python_tool_runner make sql-lint-migrations-ci
sql-lint-dml:
	@docker compose run --rm python_tool_runner make sql-lint-dml-ci
sql-lint-seed:
	@docker compose run --rm python_tool_runner make sql-lint-seed-ci

# -----CI内で実行するコマンド群-----
sql-lint-migrations-ci:
	sqlfluff lint database/migrations/ --config docker/database/sqlfluff/.migrations.sqlfluff
sql-lint-dml-ci:
	sqlfluff lint database/dml/ --config docker/database/sqlfluff/.dml.sqlfluff
sql-lint-seed-ci:
	sqlfluff lint database/seed/ --config docker/database/sqlfluff/.seed.sqlfluff

