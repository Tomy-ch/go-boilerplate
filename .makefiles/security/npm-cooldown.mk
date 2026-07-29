## npm の供給網 cooldown（.npmrc min-release-age）の遵守監査
# -----ホスト上で実行するコマンド群-----
.PHONY: npm-cooldown-audit ## lockfile が .npmrc の min-release-age を満たしているか監査(警告のみ・ゲートしない)

# -----ホスト上で実行するコマンド群-----
# min-release-age は npm の依存解決時にしか効かず、lockfile を再現するだけの
# npm ci では評価されない。CI とイメージビルドは全て npm ci のため、cooldown を
# 外して作られた lockfile は無症状で通る。その死角を埋める検知。
# finding があっても正常終了する（バイパスはテックリードの専任判断であり、
# CRITICAL への即応を CI が塞ぐのは誤りのため）。
npm-cooldown-audit:
	@go run ./scripts/npm-cooldown audit
