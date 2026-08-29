## Realtime Delivery 関連
.PHONY: realtime-init ## Realtime Delivery の table（EventLog / StreamTicket / InstanceLease）と fan-out の topic を作る（冪等 one-shot）
.PHONY: realtime-smoke ## DynamoDB Local / GoAWS へ AWS SDK v2 で native 接続できるかを smoke で確認する

# table と topic は application の起動時に作らない（docs/design/realtime-delivery.md）。app コンテナ内で実行するので
# env/.env の ENDPOINT_REALTIME / ENDPOINT_REALTIME_PUBSUB（compose のサービス名）がそのまま使える。
realtime-init:
	@echo "🔄 Realtime Delivery の table と topic を作成します..."
	@$(MAKE) infra-up
	@$(COMPOSE_APP) run --rm api_server go run ./cmd/ realtime-init
	@echo "✅ Realtime Delivery の table と topic が揃っています。"

# 共有インフラ上で走るため、資源は実行ごとに一意な名前で作り終了時に消す（scripts/realtime-smoke）。
realtime-smoke:
	@echo "🔎 DynamoDB Local / GoAWS の互換 smoke を実行します..."
	@$(MAKE) infra-up
	@go run ./scripts/realtime-smoke $(ARGS)
