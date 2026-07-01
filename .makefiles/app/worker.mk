## 常駐プロセス(worker / outbox-relay)関連
.PHONY: worker ## worker を起動（development profile のネットワーク内で常駐実行。停止は Ctrl-C）
.PHONY: outbox-relay ## outbox relay を起動（development profile のネットワーク内で常駐実行。停止は Ctrl-C）

worker:
	@test -n "$(NAME)" || { echo "❌ NAME は必須です。例: make worker NAME=sampleworker"; exit 1; }
	@echo "🏃 worker を起動します: $(NAME) $(ARGS)（停止は Ctrl-C）"
	@docker compose --profile development run --rm api_server go run ./cmd/ worker $(NAME) $(ARGS)

outbox-relay:
	@echo "🏃 outbox relay を起動します: $(ARGS)（停止は Ctrl-C）"
	@docker compose --profile development run --rm api_server go run ./cmd/ outbox-relay $(ARGS)
