> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Arch Check — Usecase

`internal/usecase/**` を usecase 層アーキ規則で監査するスキル。

## 使うとき

- `internal/usecase/` 変更を commit 直前にチェック
- Application Service orchestration のリファクタレビュー
- 単独 or `arch-check` 統合 chain

## source of truth（毎回読む）

| ソース | 用途 |
| --- | --- |
| `CLAUDE.md` (Layer Rules / Forbidden Shortcuts) | 上位制約 |
| `internal/usecase/README.md` | usecase 責務（thin orchestrator、domain IF のみ依存） |
| `internal/usecase/boundary/README.md` | boundary interface 規約（clock / tx / encrypter / 外部 IO 抽象） |
| `.golangci.yaml` `depguard:` | 既に強制済み |

## 最初のステップ: スコープ確認

単独: `AskUserQuestion`「変更ファイルのみ」/「internal/usecase/ 全体」/「キャンセル」。chain 時はスキップ。

## Step 1. ファイルスコープ解決

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/usecase/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
git ls-files 'internal/usecase/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

空 → 報告して exit。

## Step 2. lint baseline（usecase スコープ）

```sh
make lint 2>&1 | tee /tmp/arch-check-usecase-lint.out
```

`internal/usecase/` パスに絞り込み、depguard / forbidigo を記録。

## Step 3. semantic check

各 usecase Go ファイル:

1. import を `internal/usecase/README.md` で照合:
   - `internal/infrastructure/**` 直接 import → violation（usecase は domain IF 依存、infra impl 依存しない）
   - `database/sql`, `github.com/jackc/pgx/*` → violation（DB 漏洩）
   - `github.com/labstack/echo` 等 → violation（framework 漏洩）
   - `time.Now()` 呼び出し → violation（`boundary.Clock` を使う）
   - `math/rand` / `crypto/rand` 直接 → violation（boundary 経由を推奨）
2. thin-orchestrator ヒューリスティック: 関数 50 行超 or 条件分岐多数 → `suggestion`「業務ロジックは domain entity へ」
3. transaction 境界: 複数 Repository 書き込みで `tx.Manager.Do(...)` ラップが無い → `suggestion`
4. boundary 利用: 全外部依存（時刻 / 乱数 / 外部 HTTP / queue）は `internal/usecase/boundary/` interface 経由
5. Tracer span（推奨）: `internal/usecase/README.md` Observability 節に従い、各 public method の冒頭で `ctx, endSpan := u.tracer.Start(ctx); defer endSpan()` 配線が推奨。未配線は `suggestion`（推奨機能、blocker ではない）

## Step 4. レポート（日本語）

```text
arch-check-usecase 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[semantic] M 件
  internal/usecase/user/user_usecase.go:42
    violation: "time.Now()" を直接呼び出し
    source: internal/usecase/boundary/README.md "時刻は Clock 経由"
    remediation: u.clock.Now() に置換

  internal/usecase/order/order_usecase.go:88
    suggestion: 関数 67 行、conditional 多数。業務ロジックを domain 側へ
    source: internal/usecase/README.md "Application Service は thin orchestrator"

総計: violations <N+M>, suggestions <K>
```

空:

```text
arch-check-usecase 結果（スコープ: <scope>）
usecase 層の違反は検出されませんでした。
```

## Step 5. クロージング

- 単独: exit 0（情報的）
- chain: 違反数を caller に返す
- 自動修正なし

## AI 修正スコープ

read-only。ファイル変更なし。

## 制約事項

- ❌ usecase ルールをハードコード（必ず README から読む）
- ❌ depguard と重複
- ❌ ファイル変更
- ❌ 単独実行時の scope 確認をスキップ
- ✅ 日本語出力
- ✅ source-of-truth 引用
- ✅ thin-orchestrator + boundary 利用チェック
- ✅ README 毎回再読

## チェックリスト

- [ ] scope 確認 or 受領
- [ ] `internal/usecase/README.md` + `boundary/README.md` を毎回読み込み
- [ ] `make lint` 実行、usecase 絞り込み
- [ ] generated / test 除外
- [ ] boundary 利用 + thin-orchestrator + tx チェック実施
- [ ] findings に source-of-truth 引用
- [ ] レポート日本語
- [ ] ファイル変更なし
