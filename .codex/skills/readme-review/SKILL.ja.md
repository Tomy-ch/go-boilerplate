> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Readme Review

このスキルは、単一の README を「portal の manual としての価値を満たすか」のパターンと突き合わせて評価し、ユーザーが次の行動（README 補強・manifest 追加・portal 対象外として放置）を決められるスコアカードを出力します。

## 使うとき

- 新規 README を書いたあと、portal の manual として十分か知りたい
- `portal-manifest-sync` が `borderline` / `not-yet-manual-grade` に分類した README を深堀したい
- README が godoc 領域の内容を書いていないか疑いたい
- 薄い README を厚くする前に「何が足りないか」のチェックリストが欲しい

以下の用途には使いません:

- リポジトリ全体のレビュー → `portal-manifest-sync` が同じ基準（後述）で 4 クラスに batch 分類する。本スキルは borderline ケースの個別深堀用
- README 編集 → `sync-readme`（drift 修正）または手動編集
- manifest 追加 → `portal-manifest-sync`（curation flow）に chain

## 他スキルへの source-of-truth 提供

後述の Step 2（P1〜P7 / N1〜N4 / 4 クラス閾値）は、`portal-manifest-sync` が batch 分類のために **実行時に参照** する。**重複定義しないこと**。基準が進化する場合（例: 新しい positive パターンが manifest entry から発見された）は本 SKILL.md だけ編集すれば、`portal-manifest-sync` の挙動も自動追従。

本スキルは引き続き、単一 README の **深堀分析**（強み / ギャップ / 補強提案 / 次アクションのフルスコアカード）の canonical 起動点。`portal-manifest-sync` は batch 性のためファイルごとに 1 行根拠だけ出す。

## 評価基準の導出根拠

評価パターンはハードコードした理論ではなく、`docs/portal/manifest.yaml` 登録済み 53 件（スキル初期作成時点。以降拡張）を読み込んで共通点を抽出して作りました:

- 散文 median 約 225 語、最大 1358
- 27/53 が Mermaid 使用
- 48/53 がテーブル使用
- 38/53 が概念系 H2 を持つ
- H2 頻度トップ: Notes (20), Directory Structure (12), Design Policy (11), Role (7), Public API (6), Rules (5), Architecture/Architectural Position (8), Test Strategy (4), Design Principles (4)

`## Public API` を持ちつつ manifest に登録されている 6 件は、すべて Role / Architecture / Design Principles / How It Works 等の概念セクションも豊富な「ハイブリッド型」でした。純粋な API ref とは異なります。

### キーワード更新ログ

Step 2 のキーワード集合は、`portal-manifest-sync` の実行で false-negative が観測されるたびに拡張する常設ルールです。追加基準:

- 本当に manual 品質の README が、当初リストにない言い換えを使っている（例: `Conventions` は実質的に Rules セクション）
- その言い換えが既存 manifest 登録エントリの少なくとも 1 件、またはユーザーが明示的に manual-worthy と判断した README に使われている

低品質 README を通すためのキーワード追加はしない。正当な言い換えのスペクトルを取りこぼさないことが目的。

## 最初のステップ: ターゲット確認

`ask the user explicitly`:

1. **対象 README パス** — canonical 英語版。引数 / 直近メッセージにあれば候補として提示
2. **出力詳細度** — 簡潔スコアカード（デフォルト）/ パターン別フル breakdown

`*.ja.md` パスが渡された場合は、ja を直接 review するか canonical sibling に切り替えるか確認。

ターゲット確定前にファイルを読まない。

## Step 1. ターゲット読み込み

README 全文を読む。抽出:

- 全 H2 見出し
- ` ```mermaid ` ブロックの有無 / 件数
- テーブル (`|...|`) の有無
- 散文語数（コード / テーブル / 見出しを除く）
- 翻訳 sibling (`README.ja.md`) の有無と sync convention 準拠

## Step 2. 各観点の評価

H2 見出しテキストだけでなく、各セクションの内容を読んで「観点を実質的に満たしているか」を判定。

### Positive 観点（満たすと +1）

| # | 観点 | シグナル |
| --- | --- | --- |
| P1 | **Role / Position** | H2 が `{Role, Position, Overview, Role in Onion Architecture, Architectural Position, Role in This Project}` のいずれかで、パッケージ／層が何でどこに位置するかを散文で説明 |
| P2 | **Design Intent / Why** | H2 が `{Design Policy, Design Principles, Design Intent, Why, Why ..., Rationale, Approach, Necessity, How It Works, Strategy, Trigger Strategy, Test Strategy, Application Policy}` のいずれかで、ルール列挙でなく理由を語っている |
| P3 | **Rules / Boundaries** | H2 が `{Rules, Do / Don't, Don'ts, Prohibited Practices, Forbidden, Constraints, Allowed dependencies, Disallowed, Implementation Rules, Conventions, Naming Convention, Naming, Policy}` — 明示的な指示 |
| P4 | **Architecture diagram** | ` ```mermaid ` ブロック + 解説 prose 1 文以上 |
| P5 | **Navigation** | H2 が `{Directory Structure, Subdirectories, Subdirectory Roles, Subpackages, Package List, Workflow List, Command List, File List, Module List}` — 内部構造案内 |
| P6 | **Notes / caveats** | H2 `Notes` 等で運用上の注意 / pitfalls |
| P7 | **Substantive prose** | 散文語数 ≥ 150 |

### Negative 観点（トリガすると −2）

| # | 兆候 | 判定 |
| --- | --- | --- |
| N1 | **Pure API reference** | `## Public API` あり、H2 ≤3、prose <150 語、Role/Design/Architecture なし → "godoc 領域" / out-of-scope-for-portal |
| N2 | **Stub** | H2 ≤1、prose <50 語 |
| N3 | **Index-only** | H2 が `Subpackages` / `Subdirectories` のみで narrative なし |
| N4 | **Operational reference** | H2 set が `{Command, Flags, Usage, Notes}` 系の CLI usage ref → portal manual 不向き、CLI ドキュメント / godoc-cli 領域 |

N1〜N4 は保守的に適用。Design / Role / Architecture セクションがあれば、API / Subpackages / Command 見出しがあっても N1 / N3 / N4 をトリガしない。

### 分類しきい値

- **manual-worthy**: positive ≥ 3 かつ negative トリガなし
- **borderline**: positive 1〜2 かつ negative トリガなし
- **not-yet-manual-grade**: positive 0、または positive あっても N2/N3 がトリガ
- **out-of-scope-for-portal**: N1（godoc 領域）または N4（CLI ref）トリガ

## Step 3. スコアカード出力

日本語。セクション:

```text
README Review: <path>

[判定] manual-worthy | borderline | not-yet-manual-grade | out-of-scope-for-portal

[強み] (満たす positive 観点)
  ✓ P1 Role: "Role in Onion Architecture" で層境界を明示（〜「the core of the business」）
  ✓ P4 Architecture diagram: Mermaid 図 2 件 + 解説 prose
  ✓ P6 Notes: 運用上の caveats 3 項目（"transaction boundaries are managed by..."）
  ✓ P7 散文 534 語

[ギャップ] (満たさない positive 観点)
  ✗ P2 Design Intent / Why セクションなし
  ✗ P3 Rules / Do-Don't の明文化なし

[アンチパターン] (トリガした negative)
  なし
  ※（または）⚠ N1 Pure API reference: `## Public API` 支配的（他 H2 ≤2, prose 43 語）→ godoc 領域

[補強提案]
  - "Why this layer exists" を 1〜2 段落追加すると、新規開発者・AI エージェントが層を必要性で理解できる
  - "Forbidden / Don't" を箇条書きで明示すると、規約逸脱を自動チェックしやすくなる（arch-check スキルとも連動）

[portal 適性]
  manual-worthy → portal-manifest-sync の curation flow で追加候補
  （または）borderline → README 補強後に再 review 推奨
  （または）not-yet-manual-grade → 内容拡充が先、または portal 対象外でよい
  （または）out-of-scope-for-portal → godoc 領域、portal 不要
```

verbose 時は、raw H2 リスト / 散文語数 / Mermaid・テーブル件数 / H2 → 観点マッピング も併記。

## Step 4. 次アクション提案

判定に応じて 1 行サジェスト:

| 判定 | サジェスト |
| --- | --- |
| manual-worthy | "/portal-manifest-sync の curation flow でこの README を追加候補として指定するか、手動で manifest に追加してください" |
| borderline | "P2/P3 等の不足 section を補ってから再 review を推奨" |
| not-yet-manual-grade | "/sync-readme で内容を拡充するか、portal 対象外として扱ってください" |
| out-of-scope-for-portal | "godoc / CLI ドキュメント側で扱うべき内容です。portal manifest への追加は不要" |

## AI 修正スコープ

完全 read-only。

- 読む: 対象 README、（任意）`*.ja.md` sibling
- 書く: 何も書かない
- edit / stage / commit / push はしない

ユーザーが review 結果を踏まえて改善を要求した場合: `sync-readme`（構造 drift）または手動編集を推奨。本スキル内では自動書き換えしない。

## 制約事項

- ❌ 評価ルールを manifest 実態から乖離した形でハードコードする — 上記基準は現時点の manifest snapshot から導出。convention が進化したら再導出
- ❌ "全観点を満たさない → 改善が必要" と短絡判定する — 意図的に簡素な reference もあり、その場合の正解は `out-of-scope-for-portal` であって "fix it" ではない
- ❌ README を manifest に自動追加
- ❌ README を編集
- ❌ ターゲット確認 `ask the user explicitly` をスキップ
- ✅ ユーザー向け出力は日本語
- ✅ 各観点判定に対して README の該当箇所 / 引用を根拠提示
- ✅ "内容を厚くすべき"（not-yet-manual-grade）と "別の場所が正しい"（out-of-scope-for-portal）を区別
- ✅ 意図的に最小限の README は「それでよい」と素直に判定する

## チェックリスト

- [ ] 対象 README パスを `ask the user explicitly` で確認した
- [ ] README 全文を読んだ
- [ ] 全 H2 見出しを列挙し観点にマッピングした
- [ ] 散文語数を算出した（コード / テーブル / 見出しを除く）
- [ ] Mermaid / テーブルの有無を確認した
- [ ] 各 positive 観点（P1〜P7）を Yes/No + 根拠で評価した
- [ ] 各 negative 観点（N1〜N4）をチェックした
- [ ] 最終分類が閾値と一致している
- [ ] 出力は日本語で、各観点に対する具体的根拠を併記
- [ ] 次アクション提案を含む
- [ ] ファイルを編集 / stage / commit していない
