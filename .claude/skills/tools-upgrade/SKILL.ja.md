> **このファイルは `SKILL.md` の日本語訳です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# ツールバージョン更新

このスキルは `mise.toml` の `[tools]` table に並ぶ全ツールについて、upstream 最新版との差分を監査し、**サプライチェーン隔離ゲート（supply-chain quarantine gate）** 付きで適用候補を提示する。`min_age_days` 未満の新しいリリースは「通知のみ」として扱い、自動適用しない。

理由: npm / PyPI / Go module proxy への悪意あるリリースの大半は、公開後 24〜72 時間以内に検知・取り下げが行われる。一定期間（既定 7 日）待つことで、コミュニティが検知する前に取り込んでしまうリスクを抑える。

## 使用タイミング

以下のような場合に使用する。

- 定期（月次・四半期）のツールバージョン棚卸し
- リリース直前の、既知 CVE 修正版が出ていないかの確認
- セキュリティアドバイザリ後の、対象ツールに更新があるかの確認

以下の用途では使用しない。

- Go 自体のアップグレード → `/go-upgrade` を使う（`make sync-versions` 経由の伝播範囲が異なる）
- Go module 依存のアップデート（`go.mod` の `require` ブロック）→ `make tidy-lib` を直接使う
- 単発のアドホックなバージョン bump → `mise.toml` を直接編集して `make sync-versions`

## 最初に行うこと: `min_age_days` の確認

このスキルでは、**スキル起動直後に必ず `AskUserQuestion` でしきい値を確認する**。

手順:

1. スキル引数に値があれば（例 `/tools-upgrade 14`）候補として質問文に併記する（「候補: `14`」）。
2. 必ず `AskUserQuestion` を呼ぶ。
    - 質問: 「自動適用候補と判定するための最小経過日数を指定してください（推奨: `7`）」
    - 既定候補: `7`
3. 受け取った回答が 0 以上の整数であることを軽く検証し、以下の手順で `<MIN_AGE_DAYS>` として使う。

`<MIN_AGE_DAYS>` 確定までは upstream API へのアクセスや `mise.toml` の読み込みは行わない。

## AI Modification Scope について

`CLAUDE.md` の "Exception: Skill Execution" 節に基づき、スキル実行中に以下のパスへの変更が許可される。

- `mise.toml`（`[tools]` table のみ、ユーザーが承認したエントリだけを書き換え）
- `go.mod`, `docker/**/Dockerfile`, `docker/**/README.md`, `docker/**/README.ja.md` — `make sync-versions` の下流出力としてのみ（スクリプトが atomic に処理）

以下は引き続き保護対象（スキル実行中でも変更不可）。

- `AGENTS.md` / `CLAUDE.md`
- 生成ファイル（`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, `docs/` の生成物）
- バージョン bump と無関係な全てのファイル

## 実行ステップ

### 1. `mise.toml` のパース

`mise.toml` を読み、`[tools]` 配下の全 key を列挙する。各 key について backend を判定する。

| Key format | Backend | 最新バージョンの取得元 |
| --- | --- | --- |
| `aqua:owner/repo` | aqua (GitHub Releases) | `gh api repos/owner/repo/releases/latest` |
| `go:path/to/module` | go install | GitHub Releases（hosted されていれば）または `go list -m -versions path/to/module` |
| `npm:package` | npm | `https://registry.npmjs.org/{package}` |
| `pipx:package` | pipx (PyPI) | `https://pypi.org/pypi/{package}/json` |
| 短い名前（例 `golangci-lint`） | mise registry のデフォルト | `mise registry` で resolve、続いて該当 backend を query |
| `go`（ランタイム） | 公式 download manifest | `https://go.dev/dl/?mode=json` |
| `node`（ランタイム） | 公式 download manifest | `https://nodejs.org/dist/index.json` |
| `python`（ランタイム） | 公式 download manifest | `https://www.python.org/api/v2/downloads/release/` |

各ツールについて以下を取得する。

- **stable な最新バージョン**（`-rc` / `-beta` / `-alpha` / `-pre` / `-dev` 等の pre-release タグは除外）
- **公開日時**（ISO 8601）

GitHub Releases 系は `gh api` を優先する（`GITHUB_TOKEN` 経由で認証され rate limit が緩和される）。それ以外のエンドポイントは `curl -fsSL` で取得する。

### 2. 分類

各ツールを以下のクラスに分類する。

| クラス | 条件 |
| --- | --- |
| **up-to-date** | `pinned == latest`（先頭 `v` の有無を正規化したうえで一致） |
| **eligible** | `pinned != latest` かつ `now - release_date >= MIN_AGE_DAYS` |
| **pending** | `pinned != latest` かつ `now - release_date < MIN_AGE_DAYS` |
| **resolution_failed** | backend lookup が失敗（ネットワークエラー / 404 / parse 失敗） |

セーフガード: semver で「downgrade」になる場合は `resolution_failed` 扱い（reason: "potential downgrade"）。

### 3. サマリ表示

分類結果を日本語で見出し別にまとめて表示する。例:

```text
ツールバージョン監査結果（min_age_days = 7）

✅ 更新候補（公開から 7 日以上経過 / supply-chain quarantine 通過）:
  - golangci-lint: 2.12.2 → 2.13.0 （公開 2026-05-18, 17 日前）
  - sqlc: 1.31.1 → 1.32.0 （公開 2026-04-29, 36 日前）

⚠️ supply-chain quarantine（公開から 7 日未満、通知のみ）:
  - air: 1.65.3 → 1.66.0 （公開 2026-06-02, 2 日前）

✓ 既に最新:
  - oapi-codegen 2.7.0
  - lefthook 2.1.8
  ... (省略可)

❌ 取得失敗:
  - pipx:sqlfluff: PyPI への接続失敗
```

### 4. 適用候補の per-tool 確認

**eligible** が空ならステップ 6 へスキップし、書き換えは行わない。

そうでなければ `AskUserQuestion` を `multiSelect: true` で呼ぶ。各 option は 1 つの eligible ツールに対応し、description にバージョン差分と公開日を載せる。既定状態: 全選択。

ユーザーは個別 deselect 可能（特定 bump が既知の壊れもの等）。

### 5. `mise.toml` の更新

承認された各ツールについて:

- `mise.toml` 内の該当行を特定する
- バージョンリテラルだけを置換する。key（`aqua:owner/repo` / `go:path/to/module` / 短い名前）と、もとが `v` prefix を使っていた場合はその慣習を保持する
- key の並び順を変えない、無関係な key を触らない、`[settings]` table も触らない

全承認分の置換を memory 上で計算したあと、`mise.toml` を **1 回だけ書き出す**（atomic single-pass）。

### 6. 必要なら `make sync-versions` を実行

`go` / `node` / `python` のいずれかが更新されていれば `make sync-versions` を実行する。これにより `go.mod` と Dockerfile / `docker/**/README.md` 群の `FROM golang:` / `FROM node:` / `FROM python:` ハードコードが伝播する。

非ランタイムのツールだけが更新された場合は `make sync-versions` は不要。

### 7. 検証

```sh
make lint
make test
```

結果テーブル（OK / FAIL）をユーザーに報告する。失敗しても自動ロールバックはしない — どう扱うか（修正コミット追加 / revert / そのまま）はユーザーが判断する。

### 8. 最終レポート

以下をまとめて報告する。

- 更新したツール数
- quarantine（pending）で見送ったツール数
- 検証結果
- 失敗があれば内容

コミット / stage / push は行わない。ユーザーが working tree をレビューしたうえで `/commit` 等を手動実行する。

## 注意事項

- **supply-chain quarantine の根拠**: 典型的な dependency confusion / malicious release インシデント（npm `ua-parser-js` 2021、PyPI `ctx` 2022 等）は公開後 24〜72 時間以内に検知・yank されている。7 日 quarantine は大半をカバーしつつルーチン bump にも追従できるバランス点。
- **pre-release の除外**: 常に最新の **stable** リリースを選ぶ。upstream が pre-release タグを出していても latest として選択しない。
- **calendar versioning**: `2024.12.30` のような calendar versioning を使うツールは lexicographic + semver fallback で比較する。downgrade ガードは常時有効。
- **rate limit**: GitHub API は anonymous で 60 req/h（IP 単位）。本スキルは `gh api` を経由して `GITHUB_TOKEN` 認証で 1000 req/h に上げる。
- **idempotency**: 複数回起動しても安全。適用後に再実行すると、適用済みツールは up-to-date として表示される。
- スキルは auto-push しない。ユーザーが working tree をレビューし、必要なら `make sync-versions` を回したうえでコミット・push する。

## チェックリスト

完了報告時に以下を確認すること。

- [ ] `<MIN_AGE_DAYS>` を `AskUserQuestion` でユーザーに確認済み
- [ ] `[tools]` 全エントリの backend を resolve（不能なら理由付きで resolution_failed に分類）
- [ ] 各ツールを up-to-date / eligible / pending / resolution_failed のいずれかに分類
- [ ] 分類結果テーブルをユーザーに提示
- [ ] eligible が非空なら、per-tool 適用候補を `AskUserQuestion` で確定
- [ ] `mise.toml` を承認分のみ atomic に書き換え、key 形式と `v` prefix 慣習を保持
- [ ] go / node / python が更新されたなら `make sync-versions` を実行
- [ ] `make lint` + `make test` を実行
- [ ] 最終結果テーブルをユーザーに報告
- [ ] `SKILL.md` 更新時は `SKILL.ja.md` も同期
- [ ] コミット / stage / push は一切実行しない
