> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# DDD Audit

リポジトリの **DDD 解釈**を Evans 原義のパターン言語と照合する統合スキル。read-only の
`ddd-origin-auditor` を **パターン単位で** 並列 fan-out し、findings を検証したうえで、
台帳への書き込みループを integrator 自身が回す。

## このスキルが支える3層モデル

| 層 | 内容 | 所在 |
| --- | --- | --- |
| 1 | Evans の DDD パターン言語（プロジェクト非依存） | `.agents/ddd-audit/pattern-ledger.yaml` |
| 2 | 本リポジトリが各パターンをどう解釈したか | `docs/adr/` + per-layer `README.md` + `docs/rules.md` |
| 3 | 解釈をコードで強制する実装 | depguard / golangci / custom analyzer |

本スキルが監査するのは **層2 対 層1**。層3 は決定的で既に CI ゲートがあるので、そちらに置いたままにする。
この監査を linter でなく LLM でやる理由は、「そのパターンは（どんな言い回しであれ）解釈済みか」が
本質的に読解の問題だからである。ヒューリスティック linter に答えさせると規模が増えるほど自信満々に外し、
それを CI ゲートに載せるとリポジトリの DDD 主張がコイントスに乗ることになる。

## 使うとき

- リポジトリの DDD が Evans に忠実か、特定パターンが解釈済みかを問われたとき。
- ADR / domain README を追加・書き換えて、どのパターンが動くかを知りたいとき。
- Evans 未読のレビュアーを迎え、既読者と同水準のカバレッジが要るとき。
- 台帳が陳腐化して見えるとき（コーパスだけが動いた）。

次には使わない:

- Go コード対リポジトリ規則 — `arch-check`。domain 型の設計品質 — `type-design-reviewer`。
- README ↔ コード ↔ skill の drift — `back-prop`。
- feature spec の検証 — `verify-spec`。

## アーキテクチャ: fan-out 単位はドキュメントでなくパターン

検出は `.claude/agents/` 配下の read-only エージェント `ddd-origin-auditor` に委譲する。
integrator は Agent tool で **台帳の 1 パターンにつき 1 インスタンス**を並列起動する。

ドキュメント単位の fan-out は一見自明だが、ここでは誤りである。「Aggregate は解釈済みか」は、
ADR を 1 本持っている者には答えられない — 解釈は README の一節に、別の名前で、3 ファイル離れて
存在しうる。そこで各 auditor は 1 パターンを担当し、コーパス全体を走査する。ドキュメントは何度も
読まれるが、それは答えが分散している問いを立てた代償である。

auditor は **厳密に read-only**: 台帳を書かず、`AskUserQuestion` も呼ばない。書き込みはすべて
integrator が承認後に単一スレッドで行う。

## Step 0. スコープ確認

起動直後に `AskUserQuestion` を 2 問まとめて呼ぶ。

```text
質問 1: どの範囲を監査しますか？
選択肢:
  - 全パターン（台帳の全エントリ。初回 / 定期棚卸し向け）
  - 未解釈のパターンのみ（status が unexamined / examining / uninterpreted のもの）
  - 中核パターンのみ（scope: core。Evans 第2-3部の構築ブロック）
  - 変更文書に関係するパターンのみ（quick。ADR / README を触った直後向け）

質問 2: 検出後に台帳を更新しますか？
選択肢:
  - 更新する（既定） — 1 件ずつ承認を取りながら integrator が台帳へ書き込む
  - 更新しない（read-only） — レポートのみ
```

`arch-check` から chain されたときは scope が `quick` で渡ってくる — この Step は飛ばし、質問しない。

## Step 1. 台帳の読み込みとコーパス解決

`.agents/ddd-audit/pattern-ledger.yaml` を読む。パターン一覧（fan-out 単位）と `corpus` グロブ
（層2 の実体）の両方がここにある。どちらも本文にハードコードしない — 台帳が SSOT であり、
ここに写しを置けば README が 1 つ増えた瞬間に drift する。

Step 0 のスコープに従いパターンを選ぶ。`quick` では先に変更コーパスを解決し、`interpreted_by` が
変更ファイルを指すパターンに加え、`status` が `unexamined` または `uninterpreted` のパターンを
すべて残す（どちらもポインタを持たないのでファイル一致では永久に選ばれず、しかも誰も解釈して
いないパターンは誰も見張っていないパターンなので、最も価値の高い finding である）:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD"
```

選択パターンが空なら、その旨を述べて正常終了する。

## Step 2. 台帳の鮮度チェック（決定的 — 委譲しない）

監査の前に、diff を台帳自身の `corpus` グロブと突き合わせる。コーパスのファイルが変わったのに
`.agents/ddd-audit/pattern-ledger.yaml` が変わっていなければ、台帳は構造的に陳腐化している:

```sh
CHANGED=$(git diff --name-only "origin/${BASE}...HEAD")
echo "$CHANGED" | grep -q 'pattern-ledger\.yaml' && echo "ledger touched" || echo "ledger NOT touched"
```

これは純粋な集合比較なので、エージェントではなく shell で答える — `grep` で決まることにモデル呼び出しを
使うのが、監査が「重くて誰も回さないもの」に変わる経路である。最終レポートの冒頭バナーとして報告する。
ゲートではなく情報である。

## Step 3. auditor の並列 fan-out

選択した各パターンにつき `ddd-origin-auditor` を **Agent tool** で起動する。並列実行のため
**1 メッセージ内に複数のツール呼び出し**をまとめる。各 auditor に渡すもの:

- `pattern` — 台帳エントリの `id`（ちょうど 1 つ）
- `mode` — `full` または `quick`
- `files` — `quick` のときの変更コーパスファイル

各 auditor の最終メッセージが finding そのもの（Evans 原義の前提、3 値判定、根拠、台帳への反映案）
である。パターンをキーにして収集する。

> 現環境で `ddd-origin-auditor` を起動できない場合は、代わりに `.claude/agents/ddd-origin-auditor.md`
> の手順をパターンごとにインラインで実行する — 逐次になるので、その旨をレポートに書くこと。
> 待ち時間の性質が変わるためである。

## Step 4. `差異あり` の全件検証

`差異あり` の判定は、auditor が開くことのできない本の記憶に依拠している。前提を取り違えた finding は
本物とまったく同じ顔で出てくる。そのままユーザーに渡さず、`差異あり` 1 件につき `review-verifier` を
並列起動し、結論ではなく**前提**を攻撃させる。

verifier には auditor の Evans 原義の前提・判定・根拠を渡し、次を答えさせる:

1. 述べられた Evans 原義は本当に Evans のものか、後年のコミュニティ通説を彼に帰しただけか。
2. 引用された根拠は判定を支持するか、別語彙で解釈している節を見落としただけではないか。
3. auditor が見つけられなかった逸脱宣言がコーパスの別の場所に無いか。

`REFUTED` で戻ったものは 1 行の注記を残してレポートから落とす。`CONFIRMED` と `PLAUSIBLE` は
ラベル付きで残す — このラベルこそが、読者に懐疑心をどこへ使うかを教えるものである。

`差異なし` と `逸脱宣言あり` は検証を飛ばす。これらは「コーパスが既にそのパターンを扱っている」という
主張であり、その根拠は読者がワンクリックで確認できる引用だからである。

## Step 5. 集約レポート（日本語、決定前のチェックポイント）

```text
ddd-audit 結果（スコープ: <X>, 対象 <N> パターン）

台帳鮮度: <corpus 変更あり / 台帳未更新 = 陳腐化の疑い | 同期済み>

[差異あり・解釈あり] <n> 件
  <pattern> — <CONFIRMED|PLAUSIBLE>
    Evans 原義: <前提（反証可能な形）>
    根拠: <file:line>
    現状: <別解釈 / 別名で実質解釈済み>

[差異あり・未解釈] <n> 件
  <pattern> — <CONFIRMED|PLAUSIBLE>
    Evans 原義: <前提（反証可能な形）>
    根拠: <参照した範囲と、そこに無かったこと>

[逸脱宣言あり(スコープ外)] <n> 件
  <pattern> — <宣言箇所 file:line と理由の要約>

[差異なし] <n> 件
  <pattern> — <解釈の所在 file:line>

[検証で棄却] <n> 件
  <pattern> — <棄却理由の 1 行>

総計: 差異あり <n>（解釈あり <k> / 未解釈 <m>、うち CONFIRMED <c>）, 逸脱宣言 <n>, 差異なし <n>
```

**`差異あり` は 2 つに割って出す。** どちらになるかは台帳の `status` が決める——`interpreted` なら
解釈あり、`uninterpreted` なら未解釈。verdict だけでは「解釈はあるがずれている」と「そもそも解釈が
無い」が同じ見出しに並び、後者が前者の陰に隠れる。**誰かが取った立場と、誰も埋めていない空白は、
読者が次に取る手が違う**（前者は宣言するか閉じるか、後者はまず誰かが決めること）。台帳ヘッダの
`status` × `verdict` 表がこの対応の正本であり、報告はその読み方をそのまま反映する。

findings は観察として報告する。「修正してください」「対応必須」「違反」とは書かない — このリポジトリが
掲げるのは DDD 準拠であって Evans-strict 準拠ではなく、乖離が意図的な設計判断か見落としかの判定は
メンテナのものである。差異とその根拠を述べることが成果物のすべてであり、どうすべきかを述べるのは
監査が知りうる範囲を超える。

## Step 6. 台帳の per-item 更新（integrator 側、単一スレッド）

Step 0 で更新を選んだ場合のみ。反映案が現行エントリと異なる finding ごとに:

1. その finding が支持する選択肢で `AskUserQuestion` — 例:「台帳を提案どおり更新」/
   「status のみ更新し逸脱理由は保留」/「不採用として rejected にする」/「今回は触らない」。
2. 承認されたら YAML の diff（変更前 / 変更後）を提示し、`Edit` で
   `.agents/ddd-audit/pattern-ledger.yaml` に書き込む。`last_audited` を当日の日付にする。
3. ADR / README / 実装コードは決して書かない。コーパス自体の変更（例: ADR への逸脱宣言の追記）を
   望まれた場合は、ユーザーの作業として提示し、そこで止める。その散文はメンテナの声で書かれる設計
   表明であり、監査ツールが起草すれば、自らの finding を「誰も下していない決定の記録」に変えてしまう。

書き込み対象はちょうど 1 ファイル: `.agents/ddd-audit/pattern-ledger.yaml`。

書き込み後、台帳がパース可能でエントリ数が変わっていないことを確認する。

## Step 7. 終了時

- 検出は read-only auditor に委譲、検証は独立 verifier、書き込みは integrator 単一スレッド
- 台帳以外への書き込みなし（ADR / README / 実装コードは一切触らない）
- commit / push なし

## AI 改変スコープ

- 読み込み: `.agents/ddd-audit/pattern-ledger.yaml`、`docs/adr/`、per-layer `README.md`、
  `docs/rules.md`、`docs/architecture.md`（auditor が実施）
- 書き込み: **integrator のみ**、per-item 承認後に `.agents/ddd-audit/pattern-ledger.yaml` へ
- 触らない: ADR、README、実装コード、生成物、`AGENTS.md`

## 制約

- ❌ auditor を逐次起動（必ず1メッセージ内で複数 Agent 呼び出し＝並列）
- ❌ ドキュメント単位の fan-out（パターン単位でないと分散した解釈を追えない）
- ❌ `差異あり` を検証なしでユーザーに出す
- ❌ 裁定文言（「修正してください」「違反」「対応必須」）
- ❌ 台帳以外への書き込み、user 承認なしの台帳更新
- ❌ 台帳鮮度の判定を LLM にやらせる（shell の集合比較で足りる）
- ❌ パターン一覧 / corpus を skill 本文にハードコード（台帳が SSOT）
- ❌ `差異あり` を 1 つの見出しにまとめて出す（解釈ありと未解釈が混ざり、空白が逸脱の陰に隠れる）
- ✅ Japanese aggregated report、3 値判定、`file:line` 根拠
- ✅ Evans 原義の前提を finding ごとに明示（読者が反証できる形で）

## チェックリスト

- [ ] Scope + 台帳更新可否を `AskUserQuestion` で確認（`arch-check` chain 時はスキップ）
- [ ] 台帳から pattern 一覧と corpus を読み込み（ハードコードしない）
- [ ] 台帳鮮度を shell で判定
- [ ] 選択パターンの `ddd-origin-auditor` を **1メッセージ内で並列起動**
- [ ] `差異あり` を `review-verifier` で検証し REFUTED を落とす
- [ ] 集約 Japanese レポート出力（裁定文言なし、`差異あり` は status で 2 見出しへ分割）
- [ ] opt-in 時のみ per-item 承認 → 台帳へ書き込み → パース確認
- [ ] commit / push なし
