> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Local Review

実装者とは**別モデル**で回す、ローカルの敵対的・低バイアスなコードレビュー。Copilot もクラウド `/code-review` も使わない。実装者自身のモデルには盲点があり、その盲点を別モデルで拾うのが本質。`/code-review` の finder → verify パターンを下敷きにしつつ、すべてローカルで完結させ、さらにモック単体テストでは構造的に届かない **ランタイム（curl + o11y）検証** を足す。

## 使うとき

- commit / PR 前に、実装者のモデル単独では出ないセカンドオピニオンが欲しいとき。
- 複数 layer に跨る変更で、モックテストは通るが DI / middleware / 実 DB 挙動が未検証のとき。
- バグ・認証/IDOR・レイヤ違反に絞った敵対的パスをかけたいとき。

以下には使わない:

- formatting / style — `make fix` / `make lint`
- 網羅的なレイヤ適合監査 — `arch-check`（本スキルの `architecture` lens は高シグナルな違反のみ）
- spec 検証 — `verify-spec`
- 修正の適用 — 本スキルはソースに対し read-only。指摘するだけで直すのはユーザー。

## 中核アイデア — reviewer ≠ implementer

バイアス低減が設計上の制約であって、おまけではない。よって reviewer は **コードを書いた者とは別モデルの subagent** として動く:

- 2 つの reviewer エージェント（`adversarial-reviewer` / `review-verifier`）は frontmatter で既定 **`sonnet`**。通常の Opus 実装者と異なる。
- **オーケストレーターは reviewer ≠ implementer を必ず保証する。** 本セッションのモデルを確認し、`sonnet` でなければ既定のままで spawn。本セッションが `sonnet` なら、`Agent` ツールの `model` 引数で reviewer を別モデルへ上書き（この引数はエージェント定義の frontmatter より優先）— 深さなら `opus`、安価な発散なら `haiku`。reviewer と implementer を同一モデルにしない。
- reviewer は **read-only**（エージェント定義に Edit/Write 権限なし）。本スキルが修正を当てることはない。

## Step 0 — スコープ確認

即座に `AskUserQuestion`。ベースは `gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'` で取得（本リポジトリのベースは `release/*`）。未マージのコミットがあれば「変更ファイルのみ」を既定、なければ作業ツリー / 指定パスを既定。

```text
質問: どの範囲をレビューしますか？
選択肢:
  - 変更ファイルのみ（ベースブランチとの diff）  ← 未マージのコミットがある場合の既定
  - 作業ツリーの未コミット変更（git status の差分）
  - 特定のパス/ファイルを指定
  - キャンセル
```

## Step 1 — コンテキスト収集

- ベース ref を解決しレビュー対象を作る: `git diff <base>...HEAD`（未コミットなら `git diff`）+ 変更ファイル一覧（`git diff --name-only ...`）。
- どの layer/領域が触られたか検出（`internal/controller/**`, `usecase`, `domain`, `infrastructure`, `pkg`, `openapi/**`, `database/**`）。
- **エンドポイント** が触られたか（controller handler か `openapi/**`）— Step 4 を回すかの判定。
- **共有** OpenAPI コンポーネント（複数 operation から参照される `components/*`）が編集されたか — Step 4 を全 consumer に広げる判定。

## Step 2 — Finder の fan-out（別モデル、lens ごとに1体）

`adversarial-reviewer` subagent を **lens ごとに1体** 並列起動（`Agent` 呼び出しを1メッセージにまとめる）。中核アイデアのモデル規則を適用。

| Lens | 起動条件 |
| --- | --- |
| `correctness` | 常時 |
| `security` | 常時（handler / auth / DTO / `openapi/**` が触られた時は特に） |
| `architecture` | 常時 |
| `runtime-gap` | controller / DI / `openapi/**` / `database/**` が触られた時 |

各 subagent プロンプトに必ず含める: lens 名 + その定義、ベース ref + 変更ファイル一覧 + diff、`CLAUDE.md` / 該当 `README.md` / OpenAPI spec / migrations へのポインタ。`agentType: "adversarial-reviewer"`、`model:` は規則どおり、`label` は `find:security` のように。

## Step 3 — 敵対的 verify

全 finding を集め、(file, line, claim) で **dedup**。残った finding ごとに `review-verifier` subagent を1体（並列）起動し、単一 finding + ベース ref を渡す。`agentType: "review-verifier"`、`label` は `verify:<file>`。

- **CONFIRMED** と **PLAUSIBLE** を残す。**REFUTED** は落とす（件数はレポート用に保持）。
- critical/high で単一判定が頼りないときは verifier を 2〜3 体立て多数決。重要な finding ほど単一意見より多様性。

## Step 4 — ランタイム検証（curl + o11y）— エンドポイント時のみ

**Step 1 でエンドポイントが触られた場合のみ** 実行し、subagent ではなく **オーケストレーター（メインセッション）** が行う（対話的 bash・実 DB/状態・ログ読み・ユーザー確認が要るため）。`scaffold-endpoint` Step 3.5 に倣う:

1. `make test`（モック）は実 Fx グラフを組まず、auth/OpenAPI middleware も DB も通らない。だから本ステージは Step 2 の `runtime-gap` lens が *予測* したものを実地で拾う場。
2. 既知状態の対象行を用意/seed。認証/状態依存の検査は平文/状態を自分で握る行を作る。
3. 対象エンドポイントを `curl`（ローカル認証: `Authorization: Bearer debug:<subject>`）し検証: 正常系 / 主要異常系（404 / 400 / 422）/ — **operation が `security:` 宣言を持つなら** トークン無し ⇒ 401（実際に保護されているか証明）。IDOR 形の finding は *別の* subject で curl し他 subject のリソースに到達できないことを検証。
4. **共有スキーマ波及:** 共有 `components/*` を編集した場合（Step 1）、変更分だけでなく **全 consumer** を curl。spec を `$ref` で grep し各々を叩く。
5. o11y ログを1リクエスト分だけ読む: trace が controller → usecase → infra を貫き、発行 SQL が期待どおりか確認。以降の再確認は再 curl せず o11y で足りる。
6. **破壊ガード:** データを変える curl で復旧手段が `make db-init`（等）しかない場合、実行前にユーザー確認（`CLAUDE.md` 準拠）。検証で作った行は片付ける。

ランタイムで確証した不具合は CONFIRMED として curl/o11y 証拠付きでレポートに統合。

## Step 5 — レポート合成（日本語）

1つの日本語レポートを出す:

```text
## ローカルレビュー結果（reviewer: <model> / implementer: <model>）

スコープ: <base>...HEAD（<N> files） / lens: correctness, security, architecture, runtime-gap
ランタイム検証: 実施（curl/o11y）/ 対象外（エンドポイント変更なし）

### CONFIRMED（要対応）
- [重大度] タイトル — path:行
  - 問題 / 根拠 / 修正案
  - 検証: verifier 判定（+ 該当すれば curl/o11y 結果）

### PLAUSIBLE（要確認・判断保留）
- ...

### 補足
- REFUTED: <n> 件（finder が挙げたが verifier が否定）
- ランタイム検証でカバーした経路 / スキップした経路
```

重大度順、CONFIRMED を PLAUSIBLE より先に。ランタイムで何を検査し何をスキップしたかは必ず明記（黙って省くと「全部見た」と誤読される）。

## やる / やらない

- ✅ reviewer モデル ≠ implementer モデルを保証（本セッションが sonnet なら既定を上書き）。
- ✅ finder は並列（1メッセージ・複数 `Agent` 呼び出し）、lens ごとに1体。
- ✅ レポート前に全 finding を独立 verify、REFUTED は落とす。
- ✅ 触られたエンドポイントはランタイム検証、共有スキーマ編集なら全 consumer に拡大。
- ✅ 復旧手段が `make db-init` しかない破壊系 curl は事前にユーザー確認。
- ❌ 修正を当てる — 本スキルは指摘まで（reviewer は構造的に read-only）。
- ❌ reviewer を implementer と同一モデルで回す。
- ❌ 思いつきの style nit を finding として出す / 網羅に見せるための水増し。
- ❌ verify 中に生成ファイルや deny リスト対象を編集する。

## チェックリスト

- [ ] `AskUserQuestion` でスコープ確認、ベース ref 解決。
- [ ] reviewer モデル ≠ implementer モデルを確認。
- [ ] lens ごとに finder を fan-out（並列）。
- [ ] 全 finding を独立 verify、REFUTED は除外（件数は保持）。
- [ ] 触られたエンドポイントの curl + o11y 実施（共有スキーマ → 全 consumer）、破壊系は確認済み。
- [ ] 1つの日本語レポート: CONFIRMED → PLAUSIBLE、ランタイムのカバー範囲を明記。
