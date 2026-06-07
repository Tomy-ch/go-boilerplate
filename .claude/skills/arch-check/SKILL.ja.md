> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Arch Check

layer 別アーキ適合性チェックの統合スキル。scope に応じて 1〜5 の per-layer skill を chain。

## 使うとき

- commit / push 直前に touched layer 全体の適合性チェック
- 複数 layer に跨る feature ブランチのレビュー
- マージ前の CI gate

単一 layer のみ気にする時は per-layer skill（`arch-check-<layer>`）を直接呼ぶ。

以下の用途には使いません:

- formatting / style — `make fix` / `make lint`
- general code review — `/review` / `/ultrareview`
- spec validation — `verify-spec`

## chain される per-layer skill

| skill | layer | lean A 強制 |
| --- | --- | --- |
| `arch-check-domain` | `internal/domain/**` | entity ↔ SQL カラム soft 対応（method 形式 / VO ラップは逸脱許容、suggestion のみ） |
| `arch-check-usecase` | `internal/usecase/**` | thin orchestrator + boundary 利用 + tx 境界 |
| `arch-check-controller` | `internal/controller/**` | handler pure template + operationId ↔ method 一致 |
| `arch-check-infra` | `internal/infrastructure/**` | Repository pure template + sqlc gen soft 対応（multi-query / switch dispatch / JOIN 許容）+ pgerror 利用 |
| `arch-check-pkg` | `pkg/**` | `internal/` 依存禁止 + framework 非依存 |

## 最初のステップ: scope + TODO opt 確認

`AskUserQuestion` を起動直後に 2 質問 batched で呼ぶ。

既定検出 scope: ブランチ vs base (`gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'`):

- 未マージのコミットあり → 「変更ファイルのみ」を既定
- main / release/* / no diff → 「リポジトリ全体」を既定

```text
質問 1: どのスコープでアーキ検査を実行しますか？
選択肢:
  - 変更ファイルのみ（ベースブランチとの diff、touched layer のみ chain）
  - リポジトリ全体（5 layer skill 全部 chain）
  - 特定 layer のみ（layer を続けて指定）
  - キャンセル

質問 2: suggestion 検出箇所に TODO hand-off コメントを追加しますか？
選択肢:
  - 追加する（既定） — 各 per-layer skill が逸脱位置に `// TODO:` を書き込む（人間に解決を委ねる）
  - 追加しない（read-only） — レポートのみ、コード一切触らない
```

TODO opt は domain / controller / infra の per-layer skill に伝播。usecase / pkg は violation 中心なので TODO 書き込み対象外（opt 関係なく read-only）。

## Step 1. scope を layer に解決

「変更ファイルのみ」モード:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$' || true
```

path prefix で layer マッピング:

| path prefix | chain する skill |
| --- | --- |
| `internal/domain/` | `arch-check-domain` |
| `internal/usecase/` | `arch-check-usecase` |
| `internal/controller/` | `arch-check-controller` |
| `internal/infrastructure/` | `arch-check-infra` |
| `pkg/` | `arch-check-pkg` |

その他 `internal/` パス（cli / system / di / config 等） → 報告のみ、専用 layer skill 無し（CLAUDE.md guidance を直接適用）。

「リポジトリ全体」モード: 5 skill 全部 chain。

「特定 layer のみ」モード: layer を user に追加質問、該当のみ chain。

layer 検出なし（changed-files で Go 変更無し） → 明示メッセージで exit。

## Step 2. per-layer skill chain

scope 内の各 layer について、`Skill` tool で起動、scope + TODO opt context を渡して child の `AskUserQuestion` をスキップ:

- `arch-check-domain`（scope = changed-domain-files, TODO opt = yes/no）
- ... 等

per-layer skill は sequential 実行（lint output を `/tmp/arch-check-*.out` で再利用可）or 独立なら parallel。出力集約のため sequential 推奨。

各 child の findings + TODO 追加数を layer ラベル付きで収集。

## Step 3. 集約レポート

全 per-layer findings を 1 つの日本語レポートに集約:

```text
arch-check 統合結果（スコープ: <scope>）

[lint baseline]
  make lint: OK / FAIL (<n>件)

[domain] violations: N, suggestions: K
  internal/domain/foo/bar.go:12 ...

[usecase] violations: N, suggestions: K
  ...

[controller] violations: N, suggestions: K (lean A)
  ...

[infra] violations: N, suggestions: K (lean A)
  ...

[pkg] violations: N, suggestions: K
  ...

総計: violations <sum>, suggestions <sum>
TODO hand-off: 追加 <sum> 件, スキップ <sum> 件（既存コメント）
```

全 clean:

```text
arch-check 統合結果（スコープ: <scope>）
全 layer で違反は検出されませんでした（チェック済み: <layer list>）。
```

## Step 4. クロージング

- 統合 skill 自体は read-only（child も read-only）
- `/commit` から chain 時は violations > 0 で non-zero status
- 単独実行時は情報的、exit 0
- 自動修正なし

## AI 修正スコープ

統合 skill 自体は何も書かない。read scope / write scope は per-layer skill に委譲:

- 読み込み: 各 layer の README + 関連ファイル
- 書き込み: user opt 時のみ、`internal/<layer>/**/*.go` の suggestion 位置への `// TODO:` hand-off コメント追加（per-layer skill が実施、integrator 経由でも child の制約に従う）

## 制約事項

- ❌ ソースファイルを直接読まない（per-layer skill 任せ）
- ❌ scope + TODO opt `AskUserQuestion` をスキップ
- ❌ heuristic findings (handler bloat 等) を hard violation 扱い（child skill が `suggestion` ラベル付け、integrator は respect）
- ❌ ファイル変更を integrator 自身で実施（per-layer skill 経由の TODO 書き込みのみ、user opt 時）
- ✅ 日本語集約レポート
- ✅ touched layer のみ chain（changed-files モードで効率化）
- ✅ per-layer skill が独立 standalone 動作可能であることを維持
- ✅ TODO opt を child に propagate

## チェックリスト

- [ ] scope + TODO opt を `AskUserQuestion` で確認
- [ ] 変更ファイル or full repo で layer 検出
- [ ] touched layer の per-layer skill を chain
- [ ] 各 child skill が自身の README + lean A 規則を適用
- [ ] 集約日本語レポート出力
- [ ] integrator 自身のファイル変更なし（child skill が user opt 時に TODO コメント追加可能）
- [ ] commit / push なし
