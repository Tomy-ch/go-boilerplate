## pnpm の cooldown 例外の期限
# -----ホスト上で実行するコマンド群-----
.PHONY: pnpm-cooldown-check ## pnpm の minimumReleaseAgeExclude が期限・上限・解決状況を満たすか検査(違反でfail)

# 兄弟の go-cooldown / tool-cooldown と違い、窓そのものは pnpm の resolver が install 毎に
# 強制しているので、ここに gate は要らない。検査対象は逃げ道である minimumReleaseAgeExclude の
# ほうで、その期限を機械が読めるようにするのがこのターゲット。理由は .makefiles/README.md の
# pnpm-cooldown-check 行と scripts/pnpm-cooldown の package doc。
#
# 期限は pnpm-workspace.yaml が変わらなくても訪れるため、CI ではスケジュール実行にも載せる。
pnpm-cooldown-check:
	@go run ./scripts/pnpm-cooldown
