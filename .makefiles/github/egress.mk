## harden-runner の allowed-endpoints を SSOT から生成・検証するコマンド群
.PHONY: egress-apply ## .github/egress.toml を各 workflow の allowed-endpoints へ反映
.PHONY: egress-check ## allowed-endpoints が SSOT 通りか検証（書き換えなし・CI/hook用）

egress-apply:
	@go run ./scripts/egress apply

egress-check:
	@go run ./scripts/egress check
