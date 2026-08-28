## Realtime Delivery 関連
.PHONY: realtime-init ## Realtime Delivery の table（EventLog / StreamTicket / InstanceLease）を作る（冪等 one-shot）
.PHONY: realtime-smoke ## DynamoDB Local / GoAWS へ AWS SDK v2 で native 接続できるかを smoke で確認する

# table は application の起動時に作らない（docs/design/realtime-delivery.md）。app コンテナ内で実行するので
# env/.env の ENDPOINT_REALTIME（compose のサービス名）がそのまま使える。何度実行しても同じ状態に収束する。
realtime-init:
	@echo "🔄 Realtime Delivery の table を作成します..."
	@$(MAKE) infra-up
	@$(COMPOSE_APP) run --rm api_server go run ./cmd/ realtime-init
	@echo "✅ Realtime Delivery の table が揃っています。"

# 共有インフラ上で走るため、smoke は table / topic / queue を実行ごとに一意な名前で作り終了時に削除する
# （scripts/realtime-smoke）。flag は ARGS で渡す（例: make realtime-smoke ARGS="-format markdown"）。
realtime-smoke:
	@echo "🔎 DynamoDB Local / GoAWS の互換 smoke を実行します..."
	@$(MAKE) infra-up
	@go run ./scripts/realtime-smoke $(ARGS)
