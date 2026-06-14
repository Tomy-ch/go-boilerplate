## GitHub Actions の SHA pin コマンド群
.PHONY: pin-actions-resolve ## uses: の tag を SHA へ解決し lockfile を更新
.PHONY: pin-actions-apply ## lockfile を元に uses: を SHA へ固定

pin-actions-resolve:
	@go run ./scripts/pin-actions resolve

pin-actions-apply:
	@go run ./scripts/pin-actions apply
