## Realtime Delivery 関連
.PHONY: realtime-smoke ## DynamoDB Local / GoAWS へ AWS SDK v2 で native 接続できるかを smoke で確認する

# 共有インフラ上で走るため、smoke は table / topic / queue を実行ごとに一意な名前で作り終了時に削除する
# （scripts/realtime-smoke）。flag は ARGS で渡す（例: make realtime-smoke ARGS="-format markdown"）。
realtime-smoke:
	@echo "🔎 DynamoDB Local / GoAWS の互換 smoke を実行します..."
	@$(MAKE) infra-up
	@go run ./scripts/realtime-smoke $(ARGS)
