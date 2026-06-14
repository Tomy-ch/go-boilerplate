## GitHub Actions の SHA pin コマンド群
.PHONY: pin-actions-resolve ## uses: の tag を SHA へ解決し lockfile を更新
.PHONY: pin-actions-apply ## lockfile を元に uses: を SHA へ固定
.PHONY: pin-actions-check ## uses: が lockfile 通り固定済みか検証（書き換えなし・CI/hook用）

# supply-chain quarantine: N 日未満の新しすぎるコミットは採用しない（0 で無効）
PIN_ACTIONS_MIN_AGE_DAYS ?= 14

pin-actions-resolve:
	@go run ./scripts/pin-actions resolve --min-age-days=$(PIN_ACTIONS_MIN_AGE_DAYS)

pin-actions-apply:
	@go run ./scripts/pin-actions apply

pin-actions-check:
	@go run ./scripts/pin-actions check
