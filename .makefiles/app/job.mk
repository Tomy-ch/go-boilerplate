## アプリケーションジョブ関連
.PHONY: job ## Jobを実行（development profile のネットワーク内で実行）

.PHONY: job
job:
	@echo "🏃 Jobを実行します: $(NAME) $(ARGS)"
	@docker compose --profile development run --rm api_server go run ./cmd/main.go job $(NAME) $(ARGS)
	@echo "✅ Jobが完了しました。"
