## pnpm の cooldown 例外の期限
# -----ホスト上で実行するコマンド群-----
.PHONY: pnpm-cooldown-check ## pnpm の minimumReleaseAgeExclude が期限・上限・解決状況を満たすか検査(違反でfail)

# 窓自体は pnpm の resolver が担うため gate は無い。理由は .makefiles/README.md の
# pnpm-cooldown-check 行と scripts/pnpm-cooldown の package doc。
#
# 期限は pnpm-workspace.yaml が変わらなくても訪れるため、CI ではスケジュール実行にも載せる。
pnpm-cooldown-check:
	@go run ./scripts/pnpm-cooldown
