## Docker base image の digest pin コマンド群
.PHONY: pin-images-resolve ## FROM の tag を digest へ解決し lockfile を更新
.PHONY: pin-images-apply ## lockfile を元に FROM を digest へ正規化
.PHONY: pin-images-check ## FROM が lockfile 通り固定済みか検証（書き換えなし・CI/hook用）

# supply-chain cooldown: N 日未満の新しすぎる digest は採用しない（0 で無効）
PIN_IMAGES_MIN_AGE_DAYS ?= 14

pin-images-resolve:
	@go run ./scripts/pin-images resolve --min-age-days=$(PIN_IMAGES_MIN_AGE_DAYS)

pin-images-apply:
	@go run ./scripts/pin-images apply

pin-images-check:
	@go run ./scripts/pin-images check
