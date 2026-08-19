> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。直接編集せず、更新は `SKILL.md` から反映してください。

# Manage Skill

`/manage-skill` で起動されます。引数文字列: `$ARGUMENTS`。

このスキルは **本** リポジトリの `.claude/skills/` 配下のスキルを作成・保守します。薄いラッパーです。*手法*（ドラフト → テスト → レビュー → 改善 → 必要なら description 最適化）は Anthropic 公式の `skill-creator` スキルに由来し、このファイルはその上にリポジトリ固有の規約を重ねて、`commit` / `new-env` / `canonicalize-doc` などの既存スキルと同じ体裁で並ぶ成果物を作ります。

このスキルの日本語参考訳は同ディレクトリの `SKILL.ja.md` にあります（スキルとしては読み込まれません。参考用）。

## 使うとき

以下をユーザーが望むとき:

- 新規スキル / `/<name>` コマンドの作成、または現在の会話中の反復ワークフローのスキル化。
- `.claude/skills/` 配下の既存スキルの更新・修正・改善・リファクタ・リネーム・拡張 — こうした変更全般の入口であり、`SKILL.md` / `SKILL.ja.md` を手で編集する前に必ずこれを使います。
- スキルの `description` のトリガー精度最適化、またはスキルの eval / benchmark 実行。

以下には使いません:

- canonical ドキュメント（`docs/**`、パッケージ別 `README.md`）の編集 — それぞれ専用フロー（`sync-readme` / `canonicalize-doc` / `back-prop`）があります。
- 他の AI ツール設定（`.cursor/` / `.gemini/` / `.github/copilot-instructions.md`）— `AGENTS.md` によりスコープ外。

## スコープ注記（AGENTS.md）

`AGENTS.md` は通常 `.claude/**` をスコープ外として扱います。**このスキルを起動することが、それを緩和する明示的なユーザー指示** ですが、緩和は `.claude/skills/**` に限定され、この実行の間だけです。`AGENTS.md` の hard-protected パスはここでも保護されたままです。`AGENTS.md`・生成ファイル（`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`）・生成された `docs/` 配下・`permissions.deny` 配下には決して触れないこと。

## Step 0 — 公式手法の担保と読み込み（最初に行う）

公式の `skill-creator` が *how* の真の出所です。「存在する前提」で運用します。これは本リポジトリの `.claude/settings.json` に **project スコープ** で宣言されているため、信頼済みの clone には既に入っています。もし無ければ、リポジトリ共通のプラグイン bootstrap で担保します:

```bash
bash .claude/scripts/bootstrap-plugins.sh
```

bootstrap は `claude-plugins-official` マーケットプレイスを宣言し、このリポジトリが依存する公式プラグイン（`skill-creator`・`feature-dev`）を project スコープで有効化します。冪等であり、再実行は no-op です。新たに有効化されたプラグインは次セッションで読み込まれ、そのとき `skill-creator` は `/skill-creator` としても起動可能になります。ただしこのラッパーはそれに依存しません。パスでファイルを読むため同一セッションでも動作します。

その `SKILL.md` を全文読みます（スクリプトが表示したパス。必要なら以下の glob で探索）:

```bash
ls ~/.claude/plugins/marketplaces/*/plugins/skill-creator/skills/skill-creator/SKILL.md
```

そこに書かれた **Creating a skill**・**Running and evaluating test cases**・**Improving the skill**・**Description Optimization**・blind comparison・packaging はすべてそのまま適用されます。付属リソースは隣接して置かれ、そのまま利用します:

- `scripts/` — `aggregate_benchmark`・`run_loop`（description 最適化）・`package_skill` など。公式スキルのディレクトリから実行し、再実装しないこと。
- `eval-viewer/generate_review.py` — レビュービューア。常にこれを使い、レビュー用 HTML を自作しないこと。
- `agents/`（`grader.md`・`comparator.md`・`analyzer.md`）と `references/schemas.md`。

bootstrap が失敗した場合（ネットワーク無し、`claude` CLI 不在など）は、失敗をユーザーに報告し、中断する代わりにインラインで要約した手法（ドラフト → テスト → レビュー → 改善）で進めてよいか確認します。

## リポジトリオーバーレイ（＝このラッパーが足すもの）

公式手法に従いつつ、差異がある箇所では以下のリポジトリ固有ルールを適用します。競合時はこれらが優先されます。これらを無視したスキルはこのリポジトリに馴染まないからです。

### 1. 配置と構造

- 新規スキルは `.claude/skills/<name>/SKILL.md` に置き、`<name>` は `name:` フロントマターと一致する kebab-case にします。まず近隣（`commit`・`new-env`・`canonicalize-doc`・`scaffold-*` 群・integrator の `arch-check` / `back-prop`）を見て、ほぼ重複する新規追加より既存スキルの編集・拡張を優先します。
- 付属リソース（`scripts/`・`references/`・`assets/`）は必要時に公式の anatomy に従います。`SKILL.md` は約 500 行未満に保ち、詳細は明確なポインタ付きで `references/` に押し出します。

### 2. フロントマター規約（既存スキルに合わせる）

- `name`: kebab-case、ディレクトリ名と一致。
- `description`: **1 段落で密に**、英語で、公式の「pushy」なトリガー指針に従う（何をするか AND 具体的な使用コンテキスト、加えて *いつトリガーしないか* も明示）。`commit` / `new-env` の description の密度とトーンを研究し、それに合わせること。
- 任意: `argument-hint` と `allowed-tools`（`/command` が引数を取る、または固定のツールセットを使う場合。パターンは `commit` を参照）。不要なら省略します（多くのスキルは省略）。

### 3. 言語ルール（`CLAUDE.md`）

- `SKILL.md` は **英語 canonical** — 本文は既存スキル同様に命令形の英語で書きます。canonical の `SKILL.md` を日本語で書かないこと。
- ただしスキルの *実行時の振る舞い* は `CLAUDE.md` に従います。スキルが生成するユーザー可視の出力（応答・生成コードのコメント・テストケース名・PR / commit 文言）はすべて **日本語** であること。スキルを書く際はその要件を当該スキルの指示に組み込みます。

### 4. 日本語翻訳ペアの義務化

このリポジトリの全スキルは `SKILL.md` の隣に `SKILL.ja.md` 参考訳を同梱します。ここでは任意ではありません。canonical の `SKILL.md` が確定（作成）または変更（更新）されたら:

- `canonicalize-doc` スキルをチェインして canonical の `SKILL.md` から `SKILL.ja.md` を生成・同期します。
- 翻訳は標準の同期ノート見出しを持ち、それ自体はスキルとして読み込まれません。完了とする前にペアが同期していることを確認します。

### 5. eval 成果物はバージョン管理外に置く

公式手法は iteration / eval ディレクトリ・ベンチマーク・ビューア出力を含む `<skill-name>-workspace/` を書き出します。スキルディレクトリの兄弟に置くと、追跡対象の `.claude/skills/**` 内に落ちてしまいます。**配置を上書きします**。workspace はリポジトリの gitignore された `tmp/`（例: `tmp/skills/manage-skill/<skill-name>-workspace/`）に置き、このリポジトリの作業成果物規約（計画 / 成果物は git 外、`tmp/` は ignore）に合わせます。eval 実行結果・ベンチマーク・feedback JSON・ビューア HTML は決してコミットしないこと。

### 6. スキルの形に合うならリポジトリのパターンを再利用する

- 新スキルが読み取り専用の解析をレイヤー横断でファンアウトするなら、**integrator + レイヤー別サブエージェント** パターンを踏襲します（`arch-check` / `back-prop` は `*-auditor-*` / `*-detector-*` エージェントを並列起動し、integrator が単一スレッドで書き込みます）。可能なら新規サブエージェント型を発明せず既存を再利用します。
- スキルがユーザー選択でゲートされる決定的な多段フローなら、`new-env` / `commit` のように free-text プロンプトではなく `AskUserQuestion` でゲートします。
- 対象レイヤーの `README.md` / `docs/` を **実行時に真の出所として読み**（`arch-*`・`test-review`・`scaffold-*` はすべてこうしています）、drift するルールをハードコードしないこと。

## 新規スキルの作成

公式の **Creating a skill** フロー（Capture Intent → Interview → Write SKILL.md → Test Cases）を実行し、その上でオーバーレイを適用します。正しい配置・density の揃ったフロントマター / description・英語 canonical の本文・`tmp/` 配下の eval workspace。ドラフトが安定したら `canonicalize-doc` で `SKILL.ja.md` を生成します。

## 既存スキルの更新

- どのスキルか確認します（`.claude/skills/<name>/`）。`name` とディレクトリは変更しないこと。
- *インストール済み* のプラグインスキルと違い、リポジトリのスキルはその場で書き込み可能です。**`.claude/skills/<name>/` 配下を直接編集** します。read-only なコピー先を `/tmp` に作る手順は不要です。
- eval のベースライン用に、公式指針どおり編集前のスキルをスナップショット（`tmp/` へ）し、before / after を比較できるようにします。
- 編集後は `canonicalize-doc` で `SKILL.ja.md` を再同期します。更新された `SKILL.md` に古い日本語ペアが残るのは drift です。

## 評価 / 最適化

公式の test-run・benchmark・viewer・**Description Optimization** フローをそのまま使い、2 点だけ適応します。workspace は `tmp/` 配下（§5）に置くこと、description 最適化にはこのセッションを動かしているモデル ID（環境 / システムプロンプト参照）を渡してトリガーを本番に合わせること。

## Definition of Done

- 公式 `skill-creator` が解決済み（project スコープのプラグイン。無ければプラグイン bootstrap で担保）で、手法が読み込まれている。
- `.claude/skills/<name>/SKILL.md` が存在し、kebab の `name` = ディレクトリ、密な英語「pushy」`description` を持つ。
- `SKILL.ja.md` が canonical の `SKILL.md` から生成 / 同期され、整合している。
- eval 成果物が未コミット（workspace は gitignore された `tmp/`）。
- hard-protected パスに未接触。変更は `.claude/skills/**` のみ。
- 公式ループに従い、ユーザーが成果物（ビューアまたはインライン）をレビューし満足している。
- 新規または大きく変更したスキルがプラットフォーム専用でない場合、ローカル検証後にこの環境を送信元として `sync-ai` を起動する。転送契約を受信側環境の `manage-skill` に渡す。この起動自体が受信側の子操作である場合は、`sync-ai` を再起動しない。
- `make md-skill-lint` が通る。これは前項の存在部分を強制する検査で、片側の環境にしか無いスキルは落ちる。意図的にプラットフォーム専用と宣言する手順は `scripts/README.md` の Skill Lint 節に従う。
