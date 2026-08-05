## mise.toml が宣言するツールの供給網 cooldown
# -----ホスト上で実行するコマンド群-----
.PHONY: mise-cooldown-gate ## mise.toml の差分で追加/更新したツールが cooldown を満たすか検査(違反でfail)
.PHONY: mise-cooldown-audit ## mise.toml 全件の cooldown 状況を棚卸し(警告のみ・ゲートしない)

# -----ホスト上で実行するコマンド群-----
# mise 自身に短縮名の backend を解決させるため、mise が PATH に居ることを前提とする。
# GitHub API は未認証だと 60 req/hour（IP 単位）で 1 回の実行を賄えないため、GITHUB_TOKEN を
# 渡す。CI はワークフローが渡し、手元では gh auth token を使う。
#
# base に既定を置かないのは、リリース線が進んだときに古い base を黙って使い、gate が実際には
# 何も見ていない状態へ縮退するため。
mise-cooldown-gate:
	@test -n "$(BASE)" || { echo "❌ BASE が要ります: make mise-cooldown-gate BASE=origin/release/vX.Y.0"; exit 1; }
	@GITHUB_TOKEN="$${GITHUB_TOKEN:-$$(gh auth token 2>/dev/null)}" go run ./scripts/mise-cooldown gate --base=$(BASE)

# 棚卸しなので finding があっても正常終了する。ただしバイパスの期限切れだけは失敗させる
# （期限は mise.toml が変わらなくても訪れるため、回収はこの経路が担う）。
mise-cooldown-audit:
	@GITHUB_TOKEN="$${GITHUB_TOKEN:-$$(gh auth token 2>/dev/null)}" go run ./scripts/mise-cooldown audit
