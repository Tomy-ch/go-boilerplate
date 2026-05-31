> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Arch Check — Pkg

`pkg/**` を shared utility 純粋性規則で監査するスキル。

## 使うとき

- `pkg/` 変更を commit 直前にチェック
- `pkg/` への新規ユーティリティ追加レビュー
- 単独 or `arch-check` 統合 chain

## source of truth（毎回読む）

| ソース | 用途 |
| --- | --- |
| `CLAUDE.md`（"pkg/" 節） | 上位制約: `internal/` 依存禁止、framework 非依存 |
| `pkg/README.md` | pkg 層規則 |
| `pkg/<name>/README.md`（近接 sub-README） | sub-package 固有補足 |
| `.golangci.yaml` `depguard:` | 既に強制済み |

## 主要規則（CLAUDE.md / README 由来）

| 規則 | ソース |
| --- | --- |
| `pkg/` は `internal/**` を import 禁止 | CLAUDE.md "pkg must not depend on infrastructure or framework-specific packages" |
| framework 非依存（echo / fx / gorm 等は禁止、明示許可なき限り） | CLAUDE.md / pkg/README.md |
| feature 固有業務ロジック禁止 | pkg/README.md |

## 最初のステップ: スコープ確認

単独: `AskUserQuestion`「変更ファイルのみ」/「pkg/ 全体」/「キャンセル」。chain 時はスキップ。

## Step 1. ファイルスコープ解決

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'pkg/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
git ls-files 'pkg/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

空 → exit。

## Step 2. lint baseline（pkg スコープ）

```sh
make lint 2>&1 | tee /tmp/arch-check-pkg-lint.out
```

`pkg/` パスに絞り込み。

## Step 3. semantic check

各 pkg Go ファイル:

1. **`internal/` import チェック**（最重要規則）:
   - `<module>/internal/**` import → `violation`
   - ソース: CLAUDE.md "pkg/" 節
2. **framework import 検査** per `pkg/README.md`:
   - `github.com/labstack/echo`, `go.uber.org/fx`, ORM パッケージ等 → sub-package README が明示的に許可しない限り `violation`
3. **feature 固有ロジック heuristic**:
   - 関数 / 型名が特定 aggregate を参照（例: `UserSomething`） → `suggestion`「feature 固有は `internal/` 側へ」

## Step 4. レポート（日本語）

```text
arch-check-pkg 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[internal/ 依存] K 件
  pkg/foo/foo.go:5
    violation: "go-boilerplate/internal/domain/user" を import している
    source: CLAUDE.md "pkg must not depend on infrastructure or framework-specific packages"

[framework 依存] M 件
  pkg/bar/bar.go:8
    violation: "github.com/labstack/echo" を import している
    source: pkg/README.md "framework-agnostic"

総計: violations <N+K+M>, suggestions <L>
```

空:

```text
arch-check-pkg 結果（スコープ: <scope>）
pkg 層の違反は検出されませんでした。
```

## Step 5. クロージング

- 単独: exit 0
- chain: 違反数返却
- 自動修正なし

## AI 修正スコープ

read-only。

## 制約事項

- ❌ pkg ルールをハードコード
- ❌ ファイル変更
- ❌ 単独実行時の scope 確認をスキップ
- ✅ 日本語出力
- ✅ source-of-truth 引用 + 行
- ✅ `internal/` 依存禁止 + framework 非依存チェック
- ✅ README 毎回再読

## チェックリスト

- [ ] scope 確認 or 受領
- [ ] CLAUDE.md + pkg/README.md + sub-README 読込
- [ ] `make lint` pkg 絞り込み
- [ ] `internal/` import チェック
- [ ] framework import チェック
- [ ] findings に source-of-truth 引用
- [ ] レポート日本語
- [ ] ファイル変更なし
