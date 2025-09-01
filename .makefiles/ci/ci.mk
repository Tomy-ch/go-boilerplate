## CI環境用のターゲット
.PHONY: ci-db-migrate-up ## CI環境用のDBマイグレーションアップ
.PHONY: ci-db-seed ## CI環境用のDBシード
.PHONY: ci-test ## CI環境用のテスト実行
.PHONY: ci-gen-sqlc ## CI環境用のSQLCコード生成

ci-db-migrate-up:
	go run cmd/main.go migrate-up --database test

ci-db-seed:
	go run cmd/main.go db-seed --database test

ci-gen-sqlc:
	go run cmd/main.go gen-sqlc --type=repository
	go run cmd/main.go gen-sqlc --type=query_service

ci-test:
	go test ./... -coverprofile=coverage.out
