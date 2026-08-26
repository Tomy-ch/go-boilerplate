> このファイルは canonical な `README.md`（英語）の日本語訳です。内容を更新する際は `README.md` を先に直し、その後この訳を同期してください。

# full-apply

`full-verify`（read-only の全体検証）の**対になる「適用」スキル**。
`full-verify` が `tmp/skills/reviews/` に出した指摘を、重大度順に上から実際のコードへ反映していく。

```txt
full-verify  ──生成──▶  tmp/skills/reviews/mod_*.md / architecture.md / _index.md
                                   │
                                   ▼
full-apply   ──適用──▶  コード修正 + コミット
                          ├─ tmp/skills/reviews/working.md          （台帳: 完了/保留と commit ハッシュ）
                          └─ tmp/skills/reviews/mod_*.md 冒頭コメント （各指摘の対応状況 + commit ハッシュ）
```

## 役割

- 指摘を 1 件ずつ「実コードを読む → 設計判断が不要か怪しいかを判定 → 直す/保留 →
  検証(build/test) → コミット → 台帳と mod へ記録」のループで処理する。
- **怪しい（設計判断・方針選択・公開 API 破壊・影響範囲不明）指摘はスキップ（保留）**し、
  理由を残す。直すのは「設計判断が不要な、明確かつ局所的」なものに限る。
- 中断・`/clear` に強い: `working.md` 台帳と各 `mod_*.md` の対応状況コメントから
  未処理分を再構成して再開できる。

## 使い方

```text
/full-apply                          # tmp/skills/reviews/ を対象、Low まで全件、ディレクトリ単位で停止
/full-apply --reviews-dir tmp/skills/reviews-config
/full-apply --severity high          # High までで止める
/full-apply --pace all               # しきい値まで連続実行
/full-apply --dry-run                # 判定だけ（直さない）
```

起動時に、対象ディレクトリ・重大度しきい値・停止粒度を一度だけ確認する。

## 設計上のルール（要点）

- 処理順は **Critical → High → Medium → Low**、各帯はパス順。`_index.md` は途中で
  切れていることがあるため、優先順は `mod_*.md` の `重大度` から都度再構成する。
- **衝突は先勝**（先に処理した修正を優先、後発は再評価して保留 or 解消扱い）。
- **同一 md 内の連動指摘**は公開 API 非破壊の範囲でまとめて 1 コミット。
- **保護対象は変更しない**: 生成物（`*.gen.go` / `*.sql.go` / `*_mock.go` /
  `openapi.gen.yaml` / `docs/` 生成物）・`AGENTS.md`・deny 配下。生成物指摘は生成元修正か保留。
- スコープ既定は CLAUDE.md の AI 改変範囲（`internal/` `pkg/` `database/` `openapi/`）。
  `cmd/` `scripts/` `internal/cli/` `internal/system/` を含めるにはユーザーの明示同意が要る。
- コミットは日本語＋ `Co-Authored-By`、push はしない、保護ブランチ直コミットしない。

## go ツールチェーンの注意

このリポジトリは mise 管理。環境によっては goenv の shim が `go` を先取りしたり、
`mise.toml` のピンとインストール済みバージョンがずれて `go`/`make` が失敗することがある。
その場合は mise 管理下の go を明示利用する（SKILL.md 手順 0 参照）。
