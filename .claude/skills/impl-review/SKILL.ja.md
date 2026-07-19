> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Impl Review

実装者とは**別モデル**で回す、ローカルの敵対的・低バイアスなコードレビュー。Copilot もクラウド `/code-review` も使わない。実装者自身のモデルには盲点があり、その盲点を別モデルで拾うのが本質。`/code-review` の finder → verify パターンを下敷きにしつつ、すべてローカルで完結させ、さらにモック単体テストでは構造的に届かない **ランタイム（curl + o11y）検証** を足す。既定では、verifier を通過した CONFIRMED / PLAUSIBLE の指摘を、ブランチの PR に各指摘の行へアンカーした**インラインレビューコメント**として投稿する（`--no-comment` でオプトアウト。PR が無ければローカルレポートのみにフォールバック）。

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

- reviewer エージェント（`adversarial-reviewer` / `comment-reviewer` / `review-verifier`）は frontmatter で既定 **`sonnet`**。通常の Opus 実装者と異なる。
- **reviewer のモデルは Step 0 でユーザーが選ぶ。** 選択肢は `fable`（Fable 5）/ `sonnet` / `opus` / `haiku`、加えて実装者と異なるモデルへ解決される *自動* 既定。選んだモデルを `Agent` ツールの `model` 引数で各 reviewer subagent に渡す（この引数はエージェント定義の `sonnet` 既定より優先）— 深さなら `opus`、安価な発散なら `haiku`、独立した新しい視点なら `fable`。
- **オーケストレーターは reviewer ≠ implementer を必ず保証する。** ユーザーが本セッションの実装者と同一モデルを選んだ場合は、別モデルによるバイアス低減が損なわれる旨を警告し、確認してから進める。reviewer と implementer を無言で同一モデルにしない。
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

### reviewer モデルの選択

同じ `AskUserQuestion` 呼び出しの中で（スコープと並べて2つ目の質問として）、reviewer subagent を
どのモデルで動かすかを聞く。既存のティアに加え `fable`（Fable 5）が利用可能:

```text
質問: レビュアーをどのモデルで実行しますか？（バイアス低減のため 実装者 ≠ レビュアー を推奨）
選択肢:
  - 自動（実装者と異なるモデルを既定選択）  ← 既定
  - fable（Fable 5）
  - sonnet
  - opus（深掘り）
  - haiku（安価・高速な発散パス）
```

*自動* は、実装者が `sonnet` でなければエージェント定義の既定（`sonnet`）に、そうでなければ別ティアに
解決する。ユーザーが実装者と同一モデルを選んだ場合は（中核アイデアのとおり）別モデル保証が弱まる旨を
警告し、確認してから進める。選んだモデルは Step 2・Step 3 の各 `Agent` 呼び出しへ `model` 引数で渡す。

### フラグ

- `--no-comment` — Step 6 を抑止（PR に投稿せず）ローカルレポートのみ。**既定はオプトアウト**: ブランチに open な PR があれば、このフラグが無い限り Step 6 が残った指摘をインラインコメントとして投稿する。

## Step 1 — コンテキスト収集

- ベース ref を解決しレビュー対象を作る: `git diff <base>...HEAD`（未コミットなら `git diff`）+ 変更ファイル一覧（`git diff --name-only ...`）。
- どの layer/領域が触られたか検出（`internal/controller/**`, `usecase`, `domain`, `infrastructure`, `pkg`, `openapi/**`, `database/**`）。
- **エンドポイント** が触られたか（controller handler か `openapi/**`）— Step 4 を回すかの判定。
- **共有** OpenAPI コンポーネント（複数 operation から参照される `components/*`）が編集されたか — Step 4 を全 consumer に広げる判定。

## Step 2 — Finder の fan-out（別モデル、lens ごとに1体）

`adversarial-reviewer` subagent を **lens ごとに1体** 並列起動（`Agent` 呼び出しを1メッセージにまとめる）。中核アイデアのモデル規則を適用し、Step 0 で選んだ reviewer モデルを各 `Agent` 呼び出しの `model` 引数で渡す（*自動* がエージェント定義の既定に解決するときのみ省略可）。

| Lens | 起動条件 |
| --- | --- |
| `correctness` | 常時 |
| `security` | 常時（handler / auth / DTO / `openapi/**` が触られた時は特に） |
| `architecture` | 常時 |
| `runtime-gap` | controller / DI / `openapi/**` / `database/**` が触られた時 |

各 subagent プロンプトに必ず含める: lens 名 + その定義、ベース ref + 変更ファイル一覧 + diff、`CLAUDE.md` / 該当 `README.md` / OpenAPI spec / migrations へのポインタ。`agentType: "adversarial-reviewer"`、`model:` は規則どおり、`label` は `find:security` のように。

専用 finder を追加で並列起動する: (1) **comment-reviewer**（`agentType: "comment-reviewer"`, `label: "find:comment"`）— diff がコメントを追加/変更した時（ほぼ常時）、指摘は Step 5.5 で自動修正。(2) **type-design-reviewer**（`agentType: "type-design-reviewer"`, `label: "find:type-design"`）— diff が domain 型（`internal/domain/**/*.go`）に触れた時のみ。4軸ルーブリック（Encapsulation / Invariant Expression / Invariant Usefulness / Invariant Enforcement）で採点し、指摘は suggestion 級（自動修正しない）。

## Step 3 — 敵対的 verify

全 finding を集め、(file, line, claim) で **dedup**。残った finding ごとに `review-verifier` subagent を1体（並列）起動し、単一 finding + ベース ref を渡す。`agentType: "review-verifier"`、`label` は `verify:<file>`、Step 0 で選んだ reviewer `model`（reviewer ≠ implementer は同様に維持）。

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

## Step 6 — 指摘を PR にインラインコメント投稿（既定。`--no-comment` でオプトアウト）

既定では Step 5 の後、残った **CONFIRMED + PLAUSIBLE** の指摘を、ブランチの PR に **インラインレビューコメント**として投稿する — 1指摘につき1コメント、その `path:行` にアンカーし、1つの長文コメントにまとめない。**REFUTED は投稿しない。** Step 5 のローカルレポートは常に出す（本ステップは追加動作）。

以下のときは本ステップを丸ごとスキップ:

- `--no-comment` 指定時、または
- ブランチに open な PR が無い（`gh pr view` が空）— ローカルレポートのみとし、必要なら PR 作成を提案。

GitHub への投稿は外向きアクションなので、投稿前に **一度だけ** 確認する — 件数と対象 PR を提示（`AskUserQuestion`: 「<N> 件の指摘を PR #<番号> にインラインコメントとして投稿しますか？」/「投稿する」「投稿しない（ローカルレポートのみ）」）。

### 手順

1. PR 番号・リポジトリ・コメントをアンカーする commit を解決:

   ```sh
   gh pr view --json number,url -q '.number'
   gh repo view --json nameWithOwner -q '.nameWithOwner'
   git rev-parse HEAD                                # アンカー SHA
   git rev-parse @{u}                                # push 済み head — HEAD と異なれば警告
   ```

   アンカー commit は PR に push 済みの commit でなければならない。ローカル `HEAD` ≠ `@{u}` なら先に push するよう警告（`commit_id` が PR に無いコメントは API が拒否する）。

2. どの指摘をインラインにできるか判定。GitHub のインラインコメントは PR diff に含まれる行にしか付けられない。diff の hunk を解析（`gh pr diff <PR> --patch` か `git diff <base>...HEAD`）:
   - `(path, line)` が追加/文脈の hunk 内 → インライン、`side: "RIGHT"`。
   - `(path, line)` が削除行 → インライン、`side: "LEFT"`。
   - diff 外（reviewer が未変更の文脈を参照）→ インライン不可。レビュー要約 `body` にまとめる。

3. 1つのレビューに全コメントをまとめて atomic に投稿（N 個の単発コメントにしない）:

   ```sh
   gh api --method POST repos/<owner>/<repo>/pulls/<PR>/reviews --input payload.json
   ```

   `payload.json`:

   ```json
   {
     "commit_id": "<SHA>",
     "event": "COMMENT",
     "body": "🔎 impl-review (reviewer: <model>) — CONFIRMED <n> / PLAUSIBLE <m>\n\ndiff 外で行アンカー不可の指摘:\n- <path>: <要約>",
     "comments": [
       {
         "path": "<file>",
         "line": <n>,
         "side": "RIGHT",
         "body": "🔎 [CONFIRMED · high] <問題の要約>\n\n根拠: <...>\n修正案: <...>\n検証: <verifier 判定>"
       }
     ]
   }
   ```

   `event: "COMMENT"` を使う — これは助言的レビューであり `REQUEST_CHANGES` / `APPROVE` にしない。各コメント本文の先頭に `🔎 impl-review`（または `🔎 [判定 · 重大度]` タグ）を付け、人間のレビューと区別できるようにする。

4. 堅牢性: API がバッチを拒否（422 — 行が diff に無い）したら、該当コメントを要約 `body` へ移して再投稿。最後にインライン投稿分と要約分を報告 — 指摘を黙って落とさない。

## やる / やらない

- ✅ reviewer モデル ≠ implementer モデルを保証（Step 0 でユーザーが選択。実装者と同一を選んだ場合は警告して確認）。
- ✅ finder は並列（1メッセージ・複数 `Agent` 呼び出し）、lens ごとに1体。
- ✅ レポート前に全 finding を独立 verify、REFUTED は落とす。
- ✅ 触られたエンドポイントはランタイム検証、共有スキーマ編集なら全 consumer に拡大。
- ✅ 復旧手段が `make db-init` しかない破壊系 curl は事前にユーザー確認。
- ✅ 既定で CONFIRMED + PLAUSIBLE をブランチの PR にインラインコメント投稿（Step 6）。`--no-comment` か PR 無しのとき抑止。
- ✅ PR 投稿前に一度だけ確認（外向きアクション）。各コメントは `path:行` にアンカーし、diff 外の指摘はレビュー要約にまとめる。
- ❌ REFUTED を投稿する / `REQUEST_CHANGES`・`APPROVE` を使う — 投稿レビューは助言的 `COMMENT` のみ。
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
- [ ] `--no-comment` / PR 無し以外: 一度確認のうえ CONFIRMED + PLAUSIBLE をインライン PR コメント投稿（diff 外 → 要約 body）、REFUTED は除外、`event: COMMENT`。
