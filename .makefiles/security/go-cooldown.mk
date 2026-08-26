## Go モジュールの供給網 cooldown
# -----ホスト上で実行するコマンド群-----
.PHONY: go-cooldown-gate ## go.mod の差分で追加/更新した direct モジュールが cooldown を満たすか検査(違反でfail)
.PHONY: go-cooldown-audit ## go.mod 全件の cooldown 状況を棚卸し(警告のみ・ゲートしない)

# -----ホスト上で実行するコマンド群-----
# gate が検知器ではなく防御の本体である理由、direct だけを落とす理由は
# .github/workflows/README.md の Go Cooldown 節。base に既定を置かない理由は
# .makefiles/README.md の go-cooldown-gate 行。CI はワークフローが base ref から解決して渡す。
go-cooldown-gate:
	@test -n "$(BASE)" || { echo "❌ BASE が要ります: make go-cooldown-gate BASE=origin/release/vX.Y.0"; exit 1; }
	@go run ./scripts/go-cooldown gate --base=$(BASE)

# 棚卸しのみ。バイパスの期限切れだけ失敗させる理由は .makefiles/README.md の
# go-cooldown-audit 行（期限は go.mod が変わらなくても訪れる）。
go-cooldown-audit:
	@go run ./scripts/go-cooldown audit
