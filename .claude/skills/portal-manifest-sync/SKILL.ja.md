> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Portal Manifest Sync

このスキルは `docs/portal/manifest.yaml` をディスク上の実体（`README.md` / `README.ja.md`）と照合します。manifest は `scripts/portal/gen-portal-docs.ts` の入力で、各 `src` を `docs/portal/guides/` 配下の `dst` にコピーします。

## 重要な前提

### 1. manifest は manual であって辞書ではない

portal は **キュレートされた読み物としての manual** です。全 README の完全な辞書ではありません:

- 全サブパッケージの README を網羅的に並べると、概念の流れがノイズに埋もれる（`di/server/extension/decoration/README.md` 級のエントリは、アーキテクチャを理解しようとする読者にほとんど寄与しない）。
- `manifest.yaml` 内の `dst` ファイル名は人が辿りやすいようキュレートされている（例: 機械的な `database-dml.md` ではなく `database/dml/README.md` → `sqlc-query-guide.md`）。一括追加はこのキュレーション規律を壊す。

manifest 未登録のディスク README は **drift ではなく** 人間の判断待ち候補。

### 2. API surface は godoc の責務

公開 Go API は source レベルで godoc 経由 — 別の生成系で対応する（このスキルの責務外）。`readme-review` の N1（Pure API reference）基準を満たす README は **候補から除外**。件数のみ surface。

### 3. 評価基準は `readme-review` を参照（重複定義しない）

「manual-worthy か」の評価基準は本スキルにハードコードせず、`.claude/skills/readme-review/SKILL.md` を **single source of truth** として扱う（P1〜P7 positive、N1〜N4 negative、4 クラス閾値）。本スキルは Step 3 で **その基準を適用** する。基準が進化したら `readme-review` だけ編集すれば本スキルの挙動も追従。

4 クラスの結果（`readme-review` の閾値に従う）:

- **`manual-worthy`** — positive ≥ 3、negative トリガなし → curation に適する
- **`borderline`** — positive 1〜2、negative トリガなし → 少し補強で manual 化可能
- **`not-yet-manual-grade`** — N2/N3 トリガ（stub / index-only）→ README 充実が先、または portal 対象外
- **`out-of-scope-for-portal`** — N1（Pure API ref）/ N4（Operational ref）トリガ → godoc / CLI 領域、portal 不要

4 クラスとも自動追加しない。追加は人間の curation 判断。

## 使うとき

- ファイル移動 / 削除後、stale な `src` を掃除したい
- 翻訳ペアのギャップを検出・解消したい
- フィーチャーブランチをマージする前のリグレッション確認
- portal 未登録 README のスナップショットを取って手動キュレーション判断の材料にしたい
- 定期的な健康診断

以下の用途には使いません:

- portal そのものを生成 → `make gen-portal-docs`
- 未登録 README を一括追加 — manual のキュレーション意図を破壊するため
- 不足翻訳ファイルの生成 → `canonicalize-doc`
- README 本文の更新 → `sync-readme`
- 単一 README の **深堀レビュー**（強み・ギャップ・補強提案付き）→ `/readme-review <path>`

## 読み書き範囲

**読み込み（常時）**:

- `docs/portal/manifest.yaml`
- `.claude/skills/readme-review/SKILL.md`（評価基準の source of truth、実行ごとに再読込）
- リポジトリ全体の `README.md` / `README.ja.md`。次は常に除外する:
  - `docs/portal/guides/**`（原本の生成コピー）
  - `vendor/**`、`node_modules/**`
  - `.git/**`、`.claude/**`（スキル / 設定ファイル。portal のコンテンツではない）
  - `.gitignore` に一致するパス
- `scripts/portal/gen-portal-docs.ts`

**書き込み（承認後のみ）**:

- `docs/portal/manifest.yaml` — `stale` 削除 / 明示的なキュレーション要求での追加

**触らない**:

- `docs/portal/guides/**`（`make gen-portal-docs` が毎回再生成する）
- ソースの README そのもの
- `docs/` 配下の生成物

## 最初のステップ: スコープ確認

`AskUserQuestion`:

- 「manifest 同期のモードを選んでください」
- 選択肢:
  - 「検出 + 適用（差分を提示して、承認後に manifest.yaml を更新）」
  - 「検出のみ（dry-run、書き込みなし）」
  - 「キャンセル」

## Step 1. ソース列挙

```sh
find . -name '*README*.md' -type f \
  -not -path './docs/portal/guides/*' \
  -not -path './vendor/*' \
  -not -path './node_modules/*' \
  -not -path './.git/*' \
  -not -path './.claude/*' \
  | sed 's|^\./||' | sort
```

そこから:

- **`stale`**（修正対象）= `manifest_srcs - disk_srcs`
- **`pair_drift`**（プリフライト対象）= 各 disk README の sibling 翻訳の有無（両方向）
- **`uncurated_raw`** = `disk_srcs - manifest_srcs` — Step 3 の判定を通してから surface

## Step 2. Pair_drift プリフライト

pair_drift を残したまま entry を登録すると portal navigation が壊れる。本流前に解消する。

`pair_drift` 空: Step 3 へ。

非空:

- 質問: 「pair_drift が N 件あります。本流に進む前に解消しますか？」
- 選択肢:
  - 「canonicalize-doc を chain して順次解消」 — 各 pair_drift に対し `canonicalize-doc` 起動、全件処理後に Step 1 再列挙
  - 「未解消のまま続行（レポートに残す）」
  - 「中断」

## Step 3. `readme-review` の基準で分類

本スキルは独自の評価基準を持たない。`.claude/skills/readme-review/SKILL.md`（Step 2「Evaluate Each Criterion」+ 4 クラス閾値）を **実行時に再読込** して適用する。基準更新を取りこぼさないこと。

各 `uncurated_raw` エントリについて:

1. 英語 README を読み込む（`*.ja.md` sibling は Step 2 プリフライトで存在保証済み）
2. `readme-review` の P1〜P7（positive）/ N1〜N4（negative）を適用
3. `readme-review` の閾値で判定:
   - **`manual-worthy`** — positive ≥ 3、negative トリガなし
   - **`borderline`** — positive 1〜2、negative トリガなし
   - **`not-yet-manual-grade`** — N2（Stub）/ N3（Index-only）トリガ
   - **`out-of-scope-for-portal`** — N1（Pure API ref）/ N4（Operational ref）トリガ
4. 各ファイルに対して記録:
   - 判定
   - 1 行の根拠（manual-worthy / borderline では満たした positive 基準、それ以外ではトリガした negative）
   - `manual-worthy` / `borderline` のみ、inferred group (Step 4) と hypothetical dst (Step 5) を併記

Step をスキップしたり全部を `manual-worthy` 扱いにしてはならない。4 クラス分類こそが本スキルのユーザー価値。

注: 各ファイルの評価は inline で行い、`readme-review` スキル本体を 32 回呼ばないこと（個別 scorecard を 32 個出されても読めない）。`readme-review` は単一 README の **深堀レビュー** 用に温存する。

## Step 4. パス → カテゴリ対応の学習（参考情報のみ、manual-worthy / borderline のみ対象）

manifest のパス → グループ対応は **実行時に既存 manifest から導出**。manifest 編集の駆動には使わない。

アルゴリズム:

1. 各 manifest グループの `src` を集める
2. 最長共通パス prefix を計算
3. lookup テーブル構築
4. 各 `manual-worthy` / `borderline` ファイルに対して最長一致 prefix で group を tag
5. マッチしないものは `unmatched`

参考表（実行時にmanifest から構築する例）:

| Prefix | Group |
| --- | --- |
| `` (root) | `overview` |
| `.makefiles/` | `make_commands` |
| `env/` | `environment_variables` |
| `internal/controller/` | `controller` |
| `internal/domain/` | `domain` |
| `internal/usecase/` | `usecase` |
| `internal/infrastructure/` | `infrastructure` |
| `database/` | `database` |
| `internal/di/` | `di` |
| `internal/cli/` | `cli` |
| `internal/integration/` | `integration` |
| `pkg/` | `pkg` |
| `docker/` | `docker` |
| `scripts/` | `scripts` |
| `.github/workflows/` | `ci` |

## Step 5. dst パスの導出（参考情報のみ、manual-worthy / borderline のみ対象）

各 manual-worthy / borderline ファイルについて、既存規約から hypothetical dst を派生させてレポートに併記。追加提案ではない。

観測される規約:

- 英語: `docs/portal/guides/<flat-hyphenated-name>.md`
- 日本語: `docs/portal/guides/ja/<flat-hyphenated-name>.ja.md`

`<flat-hyphenated-name>` は src パスからレイヤー接頭辞を落とし `/` を `-` に置換したもの。既存 manifest の例:

- `internal/controller/handler/README.md` → `docs/portal/guides/controller-handler.md`
- `internal/controller/handler/debug/README.md` → `docs/portal/guides/controller-handler-debug.md`（仮に追加するなら） <!-- skill-lint-ignore -->
- `internal/controller/README.md` → `docs/portal/guides/controller.md`

bespoke 命名（例: `database/dml/README.md` → `sqlc-query-guide.md`）もあるため、機械的置換せずグループ実態に倣う。

## Step 6. レポート

3 セクション、4 クラス分類。日本語。

```text
Portal Manifest Sync 結果

== 修正対象 ==

[削除候補 = stale] N 件
  - src: foo/bar/README.md  (ファイルが存在しない)
  - src: foo/bar/README.ja.md

[翻訳ペアの欠落 = pair_drift] N 件
  ※ Step 2 で「未解消のまま続行」を選んだ場合のみここに残ります
  internal/controller/handler/testkit/testauth/README.md
    → README.ja.md が存在しない（canonicalize-doc で生成推奨）

== キュレーション候補 ==

[manual-worthy] N 件（※自動追加はしません。curation 判断のうえ追加してください。
判定基準: readme-review の P1〜P7 で positive ≥ 3、negative トリガなし）

  infrastructure (M 件) — group=infrastructure, dst=docs/portal/guides/infrastructure-rdb-…
    internal/infrastructure/rdb/pgerror/README.md
      → P1 Role + P2 Why (Necessity) + P4 Mermaid + P7 prose 385 語

[borderline] N 件（※P1〜P7 のいずれか 1〜2 のみ満たす。少し補強すれば
manual-worthy 化できる候補。詳細は /readme-review で個別レビュー推奨）

  internal/di/server/extension/decoration/README.md
    → P5 Navigation のみ。Role/Design の追加で manual-worthy 化可能

[not-yet-manual-grade] N 件（※N2 Stub または N3 Index-only。
README 側の充実が先。必要なら sync-readme で内容拡充推奨）

  internal/controller/handler/testkit/README.md
    → N3 Index-only: H2 が Subpackages のみで narrative なし

== 情報のみ ==

[out-of-scope-for-portal] N 件（※N1 Pure API ref または N4 Operational ref。
godoc / CLI ドキュメント領域で portal 不要）

  pkg/datetime/README.md
    → N1: `## Public API` 支配的、他 H2 ≤2、prose <150 語
  internal/cli/dumpschema/README.md
    → N4: Command/Flags/Usage/Notes パターンの CLI ref

[未マッチ = unmatched group] N 件
  path/that/doesnt/match/any/existing/group/README.md
    既存グループのどれにも prefix が一致しない（新規グループ要否は人間判断）
```

全クラス 0 件:

```text
Portal Manifest Sync 結果
ドリフトなし、追加候補なし、除外なし。
```

## Step 7. 修正対象クラスのみ承認

**`stale` のみ** `AskUserQuestion` で承認。4 つの候補クラスには承認を求めない。

### 削除（stale）

- 質問: 「manifest に残っているが実体のない N 件を削除しますか？」
- 選択肢: 「すべて削除」/「一部のみ削除」/「スキップ」

### Pair drift（Step 2 で未解消を選んだ場合のみ）

レポートに残すだけ。ここで再質問しない。

### 任意のキュレーションフロー（ユーザーが明示要求した場合のみ）

レポート閲覧後にユーザーが「これとこれを追加したい」と要求した場合のみ:

1. 追加したいファイル名をユーザーに指定してもらう
2. 各ファイルについて推定 group と提案 dst を提示し、ファイル単位（または ≤4 件チャンク）で確認
3. Step 8 で適用

**デフォルトではこのフローに入らない**。

## Step 8. manifest.yaml の更新

**in-place 編集**:

追加（任意キュレーションフロー経由のみ）:

1. 対象グループの最後のエントリを特定
2. 新規エントリペアを挿入（既存 `# English` / `# Japanese` スタイル継承）

削除（承認された stale）:

1. 該当 `src:` / `dst:` 行を特定
2. 2 行のエントリブロックを削除する（そのコメント配下の唯一のエントリだった場合は、先行するカテゴリマーカーのコメントも削除する）

常に:

- インデント（2 スペース）を維持する。
- 末尾の空行とセクション区切りを維持する。
- グループ内で en / ja のエントリを隣接させたままにする。

## Step 9. 検証

manifest を実際に編集した場合のみ:

```sh
make gen-portal-docs
```

警告 / 失敗があれば surface。

成功したら:

```sh
git diff docs/portal/manifest.yaml
```

`docs/portal/guides/**` の差分は期待されるもの。本スキルでは stage しない。

## Step 10. クロージング

- コミットしない
- 候補を一括追加しない
- 単一 README の **深堀レビュー** が必要なら `/readme-review <path>` を案内
- 4 クラス分類のサマリ:
  - stale M 件削除（または スキップ）
  - pair_drift K 件: P 件 canonicalize-doc 解消 / Q 件未解消
  - manual-worthy A 件（追加要否は user 判断）
  - borderline B 件（補強候補、/readme-review で深堀推奨）
  - not-yet-manual-grade C 件（情報のみ、README 充実が先）
  - out-of-scope-for-portal D 件（godoc / CLI 領域）

## AI 修正スコープ

`CLAUDE.md` / `AGENTS.md` の「Exception: Skill Execution」条項により、本スキルの実行中は AI Modification Scope が次に限って緩和される。

- `docs/portal/manifest.yaml` — 本スキルが書き込む唯一のファイル。

保護対象（触らない）:

- `docs/portal/guides/**`（生成物）
- `docs/portal/docs.json`（`make gen-docs-json` の生成物）
- ソースの README そのもの
- manifest 以外のすべて

## 制約事項

- ❌ カテゴリリストをハードコードする — 必ず既存 manifest から導出
- ❌ Step 2（pair_drift プリフライト）をスキップ — 非空なら必ず解消選択を提示
- ❌ Step 3 をスキップ — 内容を読まずに全部 manual-worthy にしない
- ❌ **`readme-review` の基準を本スキルに重複定義する** — 必ず実行時に SKILL.md を再読込
- ❌ 旧式の "any `## Public API` → exclude" を使う。N1 の完全条件（Public API + H2 ≤3 + prose <150 + Role/Design/Architecture なし）を必ず適用。単純見出し検知は hybrid 型 README を誤除外する（既存 manifest 登録済みエントリで実証済み）
- ❌ 不足翻訳ファイルを直接自動生成（`canonicalize-doc` chain を使う）
- ❌ `docs/portal/guides/**` を直接触る
- ❌ YAML 全体を再シリアライズ
- ❌ スコープ確認 `AskUserQuestion` をスキップ
- ❌ 種別承認なしで変更を適用
- ❌ **どのクラスからも候補を一括追加しない**
- ❌ uncurated を「修正対象のドリフト」としてフレーミングする
- ✅ ユーザー向け出力は日本語
- ✅ `manifest.yaml` を in-place 編集してフォーマットを維持
- ✅ 書き込み後に `make gen-portal-docs` で検証
- ✅ ディスク列挙から生成物 / vendor / `.claude` / `.git` を除外
- ✅ dst 衝突は明示的に surface

## チェックリスト

- [ ] `AskUserQuestion` でスコープを確認した
- [ ] manifest.yaml をパースして `src` 集合を抽出した
- [ ] ディスク README 列挙から `docs/portal/guides/`, vendor, .git, .claude を除外した
- [ ] Step 2 pair_drift プリフライトを提示した（または pair_drift 空でスキップ）
- [ ] `readme-review` SKILL.md を再読込して現在の基準を取得した
- [ ] Step 3 で各 uncurated に P1〜P7 / N1〜N4 + 閾値を適用し、1 行根拠付きで分類した
- [ ] パス→カテゴリ対応を manifest から導出した
- [ ] dst 規約を既存エントリから導出した
- [ ] stale クラスは削除前に承認を取った
- [ ] どの候補クラスも自動追加していない — すべて "curation はユーザー判断" として明示
- [ ] `manifest.yaml` を in-place 編集した（再シリアライズしていない）
- [ ] manifest 編集時のみ `make gen-portal-docs` を実行した
- [ ] `docs/portal/guides/**` を本スキルで変更していない
- [ ] commit / push を行っていない
- [ ] 最終レポートは日本語、4 クラス（manual-worthy / borderline / not-yet-manual-grade / out-of-scope-for-portal）+ stale + pair_drift の breakdown 入り
