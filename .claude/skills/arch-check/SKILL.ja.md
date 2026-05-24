---
name: arch-check
description: Go ソースファイルに対し、本プロジェクトのオニオンアーキテクチャ規約への適合性を監査する。`CLAUDE.md` と各レイヤーの canonical README（`internal/domain/README.md`、`internal/usecase/README.md`、`internal/controller/README.md`、`internal/infrastructure/README.md`、`pkg/README.md`）を **実行時の source of truth として読み込み**、レイヤールールはスキル内にハードコードしない。`make lint` を実行して depguard 由来の違反を取得した上で、depguard が表現できないルール（domain での stdlib/library 直接利用、`pkg/` から `internal/` への依存、handler の肥大化ヒューリスティック、README のみに書かれている境界ニュアンス等）を補完的にチェックする。読み込み前に `AskUserQuestion` でスコープ（変更ファイル / リポジトリ全体）を確認する。検出はレイヤー別にグルーピングし、各項目に source-of-truth ドキュメントを明記する。
---

> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Arch Check

このスキルは、Go ソースファイルに対し、本プロジェクトのオニオンアーキテクチャ規約への適合性を監査します。`CLAUDE.md` と各レイヤーの canonical README を **実行時の source of truth** として扱い、レイヤールールをスキル内に固定で持ちません。README が変化すればスキルの挙動も追従します。

## 使うとき

以下のような場合に使用します:

- commit/push 直前に、`make lint` の範囲を超えるレイヤー適合検査を行いたい
- フィーチャーブランチをレビューし、オニオン規約を end-to-end で確認したい
- レイヤーを跨ぐリファクタ後（例: usecase と domain の責務移動）
- コードレビューで指摘されたレイヤー違反疑いの追跡調査

以下の用途には使いません:

- 純粋なスタイル / フォーマット → `make fix` / `make lint`
- 一般的なコードレビュー → `/review` または `/ultrareview`
- 新規ファイルの sync 漏れ検出 → `sync-readme`

## Source of Truth（毎回読み直す）

スキルは実行ごとに以下を読み込みます。ルールはキャッシュしません。

| Source | 目的 |
| --- | --- |
| `CLAUDE.md`（「Layer Rules (Strict)」「Forbidden Shortcuts」「Core Architecture」節）| 最上位のアーキ制約 |
| `internal/domain/README.md` | domain 層のルール（純粋性、許容 import、値型 vs interface） |
| `internal/usecase/README.md` | usecase の責務と境界 |
| `internal/controller/README.md` | controller (handler) の責務と禁止パターン |
| `internal/infrastructure/README.md` | infrastructure 実装側の制約 |
| `pkg/README.md` | shared utility の純粋性（`internal/` 依存禁止） |
| `.golangci.yaml`（`depguard:` 節）| lint で既に強制されている内容を把握。**重複検査はしない** |

スコープに含まれる各レイヤーで、より深い階層に README がある場合（例: `internal/controller/handler/README.md`）はそれも読み込みます。親レイヤーのルールを上書き／補強していることがあります。

## 最初のステップ: スコープ確認

このスキルは **起動直後に必ず `AskUserQuestion` を呼ぶ** こと。

デフォルト推測:

- `gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'` でベース取得し、未マージのコミットがあれば **「変更ファイルのみ」** をデフォルト提案
- `main` / `release/*` 上、または diff なしの場合は **「リポジトリ全体」** をデフォルト提案

質問文:

- 「どのスコープでアーキ検査を実行しますか？」
- 選択肢:
  - 「変更ファイルのみ（ベースブランチとの diff）」
  - 「指定パッケージのみ」（パスをテキスト指定）
  - 「リポジトリ全体」
  - 「キャンセル」

スコープ確認前に Go ファイルや lint を読まない／走らせないこと。

## Step 1. スコープを具体的なファイルリストに変換

```sh
# 変更ファイルのみ
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$' || true

# パッケージ指定
find <pkg-path> -name '*.go' -not -name '*.gen.go' -not -name '*.sql.go' -not -name '*_mock.go' -not -name '*_test.go'

# リポジトリ全体
git ls-files '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$'
```

常に除外:

- 生成物: `*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`
- vendored: `vendor/**`
- テストファイル (`*_test.go`) — テスト規約はスコープ外（別途 `make test` 規約で管理）

ファイルリストが空なら、その旨を伝えて停止。

## Step 2. Lint ベースライン

lint を実行し出力を保存:

```sh
make lint 2>&1 | tee /tmp/arch-check-lint.out
```

以下を抽出:

- `depguard` 違反（層境界の権威ある検査）
- `forbidigo`, `gosec` 等、アーキに関わる他 linter の検出

`make lint` が無関係な理由（コンパイルエラー、lint 設定エラー）で失敗したら、その内容をそのまま報告して中断。Step 3 に進まない（壊れたコード上の semantic check は信頼できない）。

## Step 3. Semantic Check（スキル定義の補完検査）

各 Go ファイルに対し、linter では表現できない検査を行う。ルールは「Source of Truth」で読み込んだドキュメントから **実行時に導出** すること。スキルにルールリストを固定で持たせない。

ファイルパスからレイヤーを判定:

| パス prefix | レイヤー |
| --- | --- |
| `internal/domain/` | domain |
| `internal/usecase/` | usecase |
| `internal/controller/` | controller |
| `internal/infrastructure/` | infrastructure |
| `pkg/` | pkg |
| その他の `internal/`（cli/, system/, di/, config/ 等）| infrastructure-adjacent。CLAUDE.md のガイダンスを直接適用 |

各ファイルについて:

1. `import (...)` ブロックを抽出
2. 各 import を、そのファイルのレイヤーの README + CLAUDE.md に照らして評価
3. README が明示的に禁止しているもの、または責務記述から暗黙に禁じられるものを抽出
4. **controller**: handler 関数の本体に business logic の兆候がないか追加検査（ヒューリスティック: handler 関数が ~30 行超、repository 風のクエリ、controller README で usecase 側の責務とされている処理を直接行っている等）。これらは **suggestion**（強制違反ではない）として扱う
5. **domain**: `time` / `context` の使用が domain README の慣行と整合しているか確認（プロジェクトは `time.Time` を値型として許容しつつ `time.Now()` を禁じる、等の柔軟性がある。**ハードコードしたリストではなく README の文言** に従う）

スキルは source-of-truth から導出できないルールを発明しない。曖昧な場合は "violation" ではなく "needs human judgment" として surface する。

## Step 4. レポート

レイヤー別 → ファイル別にグルーピング。各項目に以下を含める:

- **Layer**: domain / usecase / controller / infrastructure / pkg
- **File:line**（特定できれば）
- **Severity**: `violation`（明示ルールへの違反）/ `suggestion`（ヒューリスティック、誤検出の可能性あり）
- **Source of truth**: 根拠ドキュメントと該当行/節
- **Suggested remediation**: 明らかな場合のみ一行ヒント。曖昧なら省略

出力テンプレート（日本語）:

```text
アーキテクチャ検査結果（スコープ: <scope description>）

[lint baseline]
  make lint: OK / FAIL (<n>件)
    - <違反サマリ（あれば）>

[domain] <n>件
  internal/domain/foo/bar.go:12
    violation: "go.uber.org/zap" を直接 import している
    source: internal/domain/README.md L42 "domain 層は logging framework を直接利用しない"
    remediation: pkg/ または usecase 側で wrap し、interface 経由で受け取る

[controller] <n>件
  internal/controller/handler/.../baz_handler.go:88
    suggestion: handler 関数が 45 行ある（README で "lightweight" と規定）
    source: internal/controller/handler/README.md L21 "handler は軽量に保つ"
    remediation: business logic を usecase に移すことを検討

総計: violations <n>, suggestions <m>
```

検出 0 件のときは明示する:

```text
アーキテクチャ検査結果（スコープ: <scope description>）
違反は検出されませんでした。
```

## Step 5. クロージング

- 自動修正は行わない。必ずユーザーに委ねる
- `/commit` からチェインされた場合、`violation` が 1 件以上あれば非ゼロ exit。スタンドアロン実行時は情報的 exit のみ
- push / commit / ファイル変更を行わない

## AI 修正スコープ

このスキルは原則 read-only。以下を読む:

- `CLAUDE.md`、各レイヤー README、`.golangci.yaml`、スコープ内の `*.go`
- `make lint` を実行（`/tmp/arch-check-lint.out` に出力する場合あり）

このスキルは:

- 一切のソースファイル / README / 設定を変更しない
- stage / commit しない
- remote に push しない

ユーザーがレポート後に修正を依頼してきた場合は、別スキルまたは手動修正を勧めて exit する。本スキルの契約は "audit only"。

## 制約事項

- ❌ レイヤールールをスキル内にハードコードする — 必ず実行時に README/CLAUDE.md から読み込む
- ❌ depguard / `make lint` で既にカバーされている検査を重複させる
- ❌ ファイル変更
- ❌ スコープ確認 `AskUserQuestion` をスキップする
- ❌ ヒューリスティック検出を `violation` として扱う
- ✅ ユーザー向けメッセージは日本語
- ✅ 各検出に source-of-truth ドキュメントと該当行/節を明記
- ✅ 生成物 / vendor を除外
- ✅ 毎回 README を読み直す（ルールキャッシュなし）

## チェックリスト

完了報告前に確認:

- [ ] `AskUserQuestion` でスコープを確認した
- [ ] `CLAUDE.md` および関連レイヤー README を今回の実行で読んだ
- [ ] `make lint` を実行し、結果をレポートに反映した
- [ ] 生成物 / vendor をスコープから除外した
- [ ] 各 violation に source-of-truth ドキュメントを明記した
- [ ] ヒューリスティック検出は `suggestion` ラベル（`violation` ではない）
- [ ] ファイルの変更 / stage / commit を行っていない
- [ ] レポートは日本語
