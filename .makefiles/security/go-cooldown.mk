## Go モジュールの供給網 cooldown
# -----ホスト上で実行するコマンド群-----
.PHONY: go-cooldown-gate ## go.mod の差分で追加/更新した direct モジュールが cooldown を満たすか検査(違反でfail)
.PHONY: go-cooldown-audit ## go.mod 全件の cooldown 状況を棚卸し(警告のみ・ゲートしない)

# -----ホスト上で実行するコマンド群-----
# npm と違い Go には解決時に窓を強制する機構が無く、go get に「新しすぎるから採るな」と
# 言わせる手段が無い。したがって gate は検知器ではなく防御の本体で、ここを通さない限り窓は
# どこにも存在しない。既存依存は grandfather し、差分で入るものだけを見る。
# indirect は MVS で direct の要求下限に縛られ自分では下げられないことがあるため報告に留める。
#
# base に既定を置かないのは、リリース線が進んだときに古い base を黙って使い、gate が実際には
# 何も見ていない状態へ縮退するため。CI はワークフローが base ref から解決して渡す。
go-cooldown-gate:
	@test -n "$(BASE)" || { echo "❌ BASE が要ります: make go-cooldown-gate BASE=origin/release/vX.Y.0"; exit 1; }
	@go run ./scripts/go-cooldown gate --base=$(BASE)

# 棚卸しなので finding があっても正常終了する。ただしバイパスの期限切れだけは失敗させる
# （期限は go.mod が変わらなくても訪れるため、回収はこの経路が担う）。
go-cooldown-audit:
	@go run ./scripts/go-cooldown audit
