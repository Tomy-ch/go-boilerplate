> この文書は英語の [`SKILL.md`](SKILL.md) から保守される日本語訳です。単独で編集しないでください。

# 文書ドリフトレビュー

権威の順序は **README > code > skill** です。このスキルはドリフトを見つけます。実装の挙動で、記録済みの判断を黙って上書きしません。

## スコープと分類

既定では、既存 PR の `baseRefName` に対する変更済み production Go ファイルを対象にします。PR がなければ、`origin` の live state を読み最新の release line を返す `make base-branch` でベースを解決します。GitHub のデフォルトブランチは以前の release line を指すことがあるため、`gh repo view --json defaultBranchRef` には決して fallback しません。ベースを解決できなければ、日本語で報告して停止します。空の file list は detector を一つも fan-out せず、drift がない正常結果と区別できないためです。全体走査または layer 指定も受け付けます。生成物、mock、test は除外します。スコープと複数選択の分類を 1 回だけ確認し、既定では 5 分類すべてを対象にします。

要求された分類だけを監査します。

- **A — README → code:** 実装が文書化済み規約に違反している、または一致しなくなった。
- **B — code → README:** 意図的に見える反復パターンが文書化されていない。報告には独立した 3 件以上の根拠を要する。
- **C — skill ↔ README:** skill が古い規則を重複させている、または正本 README と矛盾する。
- **D — DDD ledger ↔ ADR/README corpus:** DDD パターン台帳の参照先または記載範囲が正本 corpus と一致しない。これは台帳のドリフトであり、Evans への忠実性評価ではない。比較は `ddd-audit` の担当である。
- **E — business glossary ↔ structural prose:** `docs/spec/glossary.md` の用語が、実装構造または判断のための散文に現れた。用語は glossary の Terms 表と Mechanism vocabulary 節が決め、このスキルは決めない。

Go ファイルを `domain`、`usecase`、`controller`、`infrastructure`、`pkg` に対応付けます。D では `.agents/ddd-audit/pattern-ledger.yaml` から実行時に `corpus` glob を読み（ここにハードコードしない）、変更ファイルと交差させます。D に対象 corpus があれば、layer detector と並行して `.codex/agents/drift-detector-ddd.toml` を実行します。E では、`internal/**/README.md`、`pkg/**/README.md`、`docs/adr/*.md`、`docs/rules.md`、`docs/architecture.md` から `*.ja.md` を除いた prose corpus を実行時に解決します。変更ファイルとの E の交差が空でなければ、または全体走査なら、`.codex/agents/drift-detector-glossary.toml` を実行します。解決済み prose ファイル一覧だけを渡し、categories は渡さず、layer detector と同じ並行 fan-out に含めます。`docs/spec/glossary.md` がなければ、Terms 表が probe list であるため detector を skip し、その理由を報告します。E は prose のみの変更でも選べます。これらの layer、DDD/E corpus の外にある変更ファイルは未監査として報告します。対象がなければ正常に終了します。

## 読み取り専用の検出

1. `AGENTS.md`、関連する layer README、最寄りの subpackage README を読みます。C のときだけ対応する Codex skill を読みます。D は台帳と実行時解決した corpus だけを読みます。E は glossary detector が glossary、exclusions、実行時解決した prose corpus を読みます。
2. スコープ内の code を確認し、具体的な file:line 根拠を集めます。兄弟 code は補助根拠としてだけ使い、明示された規則の代わりにはしません。
3. 所見を権威順序と比較します。

   - A では code 修正または明示的な文書化済み例外を勧め、code は編集しません。
   - B では 3 件の反復と意図の根拠があるときだけ README 追加を提案します。それ以外は報告しません。
   - C では skill から重複した手順規則を外すか README を参照させます。README 規則を skill に複写しません。
   - D では D1 pointer rot、D2 semantic rot、D3 uncaptured interpretation だけを表面化します。解消のために ADR または README を書き換えず、承認可能な書き込み先は台帳だけです。
4. 実行環境が並列検査を支援するなら、独立した layer と corpus 駆動の DDD・glossary detector を並行実行します。そうでなければ逐次実行します。検出は常に読み取り専用です。

## 書き込み前の報告

書き込みを提案する前に、Japanese の layer 別レポートを返します。

```text
back-prop 検出結果（スコープ: <scope>、種別: <A/B/C/D/E>）

[<layer>]
- A|B|C: <file:line または README section>
  判断: <なぜドリフトか>
  根拠: <canonical README / repeated code evidence>
  選択肢: <code fix | README update | skill simplification | documented exception | ignore>

[ddd]
- D1|D2|D3: <ledger entry と corpus file:line>
  判断: <なぜ台帳と corpus がずれたか>
  根拠: <runtime-resolved canonical corpus>
  選択肢: <ledger update | ddd-audit follow-up | ignore>

[glossary]
- E1: <layer README file:line>
  判断: <glossary の用語が構造の散文へ漏れた>
  根拠: <glossary term と文脈>
  選択肢: <remove term | restate in structural language | ignore>
- E2: <ADR または rules file:line>
  判断: <用語が決定記録または統べる文書へ漏れた>
  根拠: <glossary term と文脈>
  対応: 報告のみ。承認・書き換えの対象外

[glossary exclusions]
- <抑制件数、および有効な全 exclusion の reason と until>

総件数: <n>
```

ユーザーが各所見の解消を選ぶまで何も書き込みません。所見がなければ、検査した layer と分類を示します。

## 承認済みの文書変更を適用する

README または Codex skill の変更は、ユーザーが当該所見を明示的に承認してからだけ適用します。書き込み前に意図する差分を示し、権威順序を保つ理由を説明します。

- 実装 code、生成物、`AGENTS.md` は絶対に編集しません。
- 編集は影響を受ける英語正本 README または `.codex/skills/<name>/SKILL.md` に限ります。個別承認された D 所見では `.agents/ddd-audit/pattern-ledger.yaml` も許可されます。
- D 所見を ADR または README の書き換えで解消しません。そのような決定記録の変更はユーザー向け follow-up として表面化します。
- code 修正の提案は、実装せずタスクとして報告します。
- detector の E1/E2 分離を保ちます。E1 は layer README のため通常の個別承認対象です。E2（ADR または `docs/rules.md`）は報告のみで、承認を求めず編集もしません。E2 を `docs/spec/glossary.md` の編集で解消しません。用語の保守は `glossary` の担当であり、finding を消すために用語を削除すると定義を壊します。`.agents/glossary-drift/exclusions.yaml` を編集しません。除外の宣言は user の判断であり、detector や integrator が自分を黙らせる手段ではありません。
- 文書を編集した後は `make md-lint` を実行します。必要なら、承認済み Markdown 変更だけに `make md-fix` を実行してから再度 lint します。

## 完了

各所見を approved、rejected、code work への deferred、または ignored として報告します。E2 と抑制済み E 所見は常に report-only として見える状態を保ちます。変更した全ファイルと Markdown validation の結果を示します。stage、commit、push はしません。
