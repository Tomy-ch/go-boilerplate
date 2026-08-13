> **このファイルは `SKILL.md` の日本語訳です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# Tool Map

`/tool-map` として起動された。ユーザーの引数文字列は `$ARGUMENTS`。

このコマンドは、**プロジェクトレベル**の `.codex/` ディレクトリ配下に登録された全カスタマイズエントリを棚卸しする（`~/.codex/` はスキャンしない）。commands・skills・agents の 3 種を対象とし、単一の Markdown レポートを生成する。

## Step 1. 入力の解決

`$ARGUMENTS` を以下の任意フラグとしてパースする。欠落または不正なフラグは `ask the user explicitly` で確認する。

| フラグ | 値 | 既定 |
| --- | --- | --- |
| `--lang` | `en` / `ja` | `ja` |
| `--output` | `inline` / `file` | `inline` |
| `--output-path` | 任意の相対パス | `./TOOL_MAP.md`（en）または `./TOOL_MAP.ja.md`（ja）— `--output=file` のときのみ使用 |
| `--include` | `commands,skills,agents` のカンマ区切り部分集合 | `commands,skills,agents`（3 種すべて） |

`ask the user explicitly` のフォールバック質問（未解決のフラグのみ尋ねる）:

1. 出力言語は？ — `en` / `ja`
2. 出力先は？ — `inline` / `file`（`file` の場合は `--output-path` も尋ねる）
3. 含めるエントリ種別は？ — `commands` / `skills` / `agents` / `all`

必要な値がすべて解決するまで、スキャンも書き込みも行わない。

## Step 2. エントリの列挙

現在の作業ディレクトリ配下のプロジェクトレベルのパスのみをスキャンする:

| 種別 | パス glob | エントリファイル |
| --- | --- | --- |
| commands | `.codex/commands/` | `<name>.md` <!-- skill-lint-ignore --> |
| skills | `.codex/skills/` | `<name>/SKILL.md`（`SKILL.ja.md` など `*.ja.md` 翻訳ファイルは除外） |
| agents | `.codex/agents/` | `<name>.md` |

探索コマンド（`Bash` の `find` / `ls` を使い、ファイルごとに `Read`）:

```sh
test -d .codex/commands && find .codex/commands -maxdepth 1 -type f -name '*.md'
test -d .codex/skills   && find .codex/skills   -mindepth 2 -maxdepth 2 -type f -name 'SKILL.md'
test -d .codex/agents   && find .codex/agents   -maxdepth 1 -type f -name '*.md'
```

除外:

- `*.ja.md` 翻訳ファイル（エントリとして読み込まれない）。
- `SKILL.md` を持たない `.codex/skills/` 配下のディレクトリ。
- 隠しファイル（`.DS_Store` など）。

## Step 3. メタデータの抽出

見つかった各エントリについて frontmatter と本文を読み、以下を抽出する:

| フィールド | commands | skills | agents |
| --- | --- | --- | --- |
| Name | ファイル名の語幹 | `name:`（フォールバック: ディレクトリ名） | ファイル名の語幹（frontmatter `name:` があればそれも） |
| Description | `description:` | `description:` | `description:` |
| Argument hint | `argument-hint:` | — | — |
| Allowed tools | `allowed-tools:` | — | `tools:` |
| Model override | `model:` | — | `model:` |
| Path | 作業ディレクトリからの相対 | 作業ディレクトリからの相対 | 作業ディレクトリからの相対 |

Description は棚卸し表向けに最初の 1 文（または約 120 文字）に切り詰める。

## Step 4. 依存の検出

依存とは、あるエントリが別のエントリを名前で呼び出す・明示的に参照する相互参照のこと。本文テキストから以下で検出する:

1. **明示的な呼び出し表現**: `invoke the` + name + `skill`、`chains into` + name、`via the Skill tool with` + name、`Agent({ subagent_type: '` + name + `' })`、他コマンドの `/<name>` リテラル呼び出し。
2. スキャン済み別エントリの `name:` に一致する **バッククォート囲みの名前**。
3. `Chain`・`Calls`・`Depends on`・`Chains into` などの **セクション見出し** に続く名前参照。

依存は有向辺 `caller → callee` として記録する。各辺に caller 種別と callee 種別のタグを付ける（例: `skill → skill`、`skill → command`）。

除外:

- 自己参照（自分の名前に言及するエントリ）。
- 無関係な言語のフェンス済みコードブロック内の言及（`bash`・`sh`・`make`・`sql` など）。**ただし** 明らかに呼び出し例である場合を除く。
- 偶然名前に一致する一般的な英単語。一致はバッククォート囲みか、上記の呼び出し表現のいずれかに従うことを要求する。

検出した参照がスキャン集合に存在しない名前を指す場合は、Notes セクション用に **broken edge**（壊れた辺）として記録する。

## Step 5. レポートの描画

選択した言語で以下の 4 セクションを生成する。skill 名とパスは言語に関わらずそのまま。

### 1. Summary

種別ごとの 1 行カウント。含めた種別のみ表示する。

- `Commands: N`
- `Skills:   M`
- `Agents:   K`
- `Total:    N + M + K`

### 2. Inventory Tables

含めた種別ごとに 1 表を、commands → skills → agents の順で。種別が 0 件なら表は省略する（Summary では 0 を報告する）。

- **Commands 表**: Name | Description | Args | Allowed Tools | Model | Dependencies | Path
- **Skills 表**: Name | Description | Dependencies | Path
- **Agents 表**: Name | Description | Tools | Model | Dependencies | Path

`Dependencies` は `<callee-name> (<callee-type>)` のカンマ区切り一覧。無ければ空。

### 3. Dependency Graph

Mermaid の `graph LR` 図。読者が区別できるよう、各ノードに種別ごとの class を適用する:

```mermaid
graph LR
  classDef cmd fill:#cce5ff,stroke:#3b82f6
  classDef skl fill:#d4edda,stroke:#22c55e
  classDef agt fill:#fff3cd,stroke:#f59e0b
  %% nodes (include isolated entries so standalone tools appear)
  %% edges: caller --> callee
  %% class assignments
```

ノード id にはエントリ名を使う（Mermaid で不許可の文字は `_` に置換する）。

### 4. Notes

以下を指摘する短い散文セクション:

- **Leaf エントリ**（出入りの辺が無い）。
- **Hub エントリ**（2 つ以上から依存される）。
- **Broken edges**（スキャンに存在しない名前への参照）。
- **種別をまたぐ連鎖**（例: command を呼ぶ skill — あり得るが珍しい）。
- **空の種別**（例: 「プロジェクトレベルの agents は見つからなかった」）。

### Language

- `en`: セクション見出し・散文・表ヘッダを英語で。
- `ja`: セクション見出し・散文・表ヘッダを日本語で。

## Step 6. 出力

- `inline`: レポート全文を応答に含めて終了する。
- `file`: レポートを `--output-path` に書き込み、短い確認（1 行サマリ + ファイルパス）で応答する。レポート全文を応答に重複させない。

## Step 7. Markdown Lint で検証（`--output=file` のときのみ）

`--output=file` のとき、レポート書き込み後に実行する:

```sh
make md-fix
make md-lint
```

`make md-fix` はリポジトリ全体に `markdownlint-cli2 --fix` を走らせ、よくある問題（見出し / リスト / コードブロック周りの空行、末尾空白、ファイル末尾改行など）を自動修正する。`make md-lint` はその結果が `.markdownlint-cli2.yaml` に対してクリーンかを検証する。

`make md-lint` が残るエラーを報告した場合:

1. lint 出力を読む。
2. 自動修正で解決できない違反（見出し階層、重複見出し、裸 URL など）を手で直す。
3. `make md-fix` → `make md-lint` をクリーンになるまで繰り返す。

`make md-lint` がクリーンに終了するまで、コマンド完了を報告しない。

`make md-fix` はリポジトリ全体を対象とするため、レポートと無関係な Markdown を変更し得る。完了報告時にそうしたファイルを列挙し、ユーザーが広い変更セットをレビューできるようにする。

`--output=inline` のときはこのステップをスキップする（ファイルを書いていない）。

## Constraints

- このコマンドは **既定で読み取り専用**。書き込みはユーザーが明示的に `--output=file` を選んだ場合のみ、確認済みの出力パスに限り許される。
- スキャン範囲は **プロジェクトレベルのみ**。`~/.codex/` 配下は読み取りも列挙もしない。
- プラグイン提供のエントリは対象外。
- スキャンしたエントリを一切変更しない。このコマンドは検査のみ。
- `.codex/commands/`・`.codex/skills/`・`.codex/agents/` が存在しない場合、そのエントリ数を 0 として扱い、エラーにせずレポートに注記する。 <!-- skill-lint-ignore -->

## Checklist

完了報告の前に確認する:

- [ ] 必要な入力がすべて解決した（`$ARGUMENTS` または `ask the user explicitly` 経由）
- [ ] プロジェクトレベルの `.codex/{commands,skills,agents}/` のみをスキャンした
- [ ] skills スキャンから `*.ja.md` を除外した
- [ ] 各エントリの frontmatter をパースした（name・description・種別固有フィールド）
- [ ] 文書化ルールどおり依存を検出した（自己参照は除外、broken edge は記録）
- [ ] レポートに Summary・Inventory Tables・Dependency Graph（Mermaid）・Notes を含む
- [ ] 単独エントリがグラフに孤立ノードとして現れる
- [ ] `--output=file` の場合、ファイルを書き `make md-lint` がクリーンに終了した
- [ ] `--output=file` の場合、確認済み出力（および `make md-fix` の副作用）以外のパスを変更していない
