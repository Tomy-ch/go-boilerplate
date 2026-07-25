## アプリケーションジョブ関連
.PHONY: job ## Jobを実行（app コンテナ内で実行）

job:
	@test -n "$(NAME)" || { echo "❌ NAME は必須です。例: make job NAME=usercount"; exit 1; }
	@echo "🏃 Jobを実行します: $(NAME) $(ARGS)"
	@$(MAKE) infra-up
	@$(COMPOSE_APP) run --rm api_server go run ./cmd/ job $(NAME) $(ARGS)
	@echo "✅ Jobが完了しました。"
