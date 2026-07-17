# `.claude/` — 本リポジトリのエージェント設定

このディレクトリは、**リポジトリに同梱される** Claude Code 設定を保持します。再利用可能な skill、
subagent、scaffolding が読むスペックテンプレート、補助スクリプト、project スコープの権限 / プラグイン
宣言です。リポジトリを clone して信頼したメンバーは、全員が同じエージェント挙動を継承します。

英語 canonical 版は [`README.md`](README.md) を参照してください（本ファイルはその日本語参考訳です）。

## `AGENTS.md` との関係

`AGENTS.md`（リポジトリ直下）は人手で管理する **運用契約** で、エージェントが何に触れてよいか・どう振る舞う
べきかを定めます。それが真の出所であり、本 README はそのルールを再掲しません。特に `AGENTS.md` は
`.claude/**` を、ユーザーが明示的に変更を要求するか実行中の skill がその実行の間だけスコープを緩和しない限り、
**AI 編集のスコープ外** として扱います。まず `AGENTS.md` を読んでください。

## 構成

| パス | 内容 |
| --- | --- |
| `settings.json` | project スコープの権限（`allow` / `ask` / `deny`）、有効化プラグイン、既知マーケットプレイス。リポジトリを信頼した全員と共有されます。 |
| `skills/<name>/` | 再利用可能な skill。英語 canonical の `SKILL.md`（+ 参考訳 `SKILL.ja.md`）と、任意の同梱 `scripts/` / `references/` を持ちます。`/<name>` で起動。 |
| `agents/` | skill が使う subagent 定義（例: integrator skill が並列でファンアウトする読み取り専用のレイヤー別ワーカー `arch-auditor-*` / `drift-detector-*`）。 |
| `scaffold-spec/` | スペック形式定義（`domain-spec.md`・`usecase-spec.md`・`verify-rules.md` など）。`scaffold-*` / `verify-spec` / `new-spec-*` skill が **実行時に** 読むため、形式変更が skill を編集せずに伝播します。 |
| `scripts/` | リポジトリレベルのエージェントツールスクリプト（プロジェクトのビルドツールではありません。それは直下の `scripts/` にあります）。 |

## 初回セットアップ: 公式プラグイン

本リポジトリは Anthropic 公式プラグインに依存し、`settings.json` に **project スコープ** で宣言しています
（`enabledPlugins` + `extraKnownMarketplaces`）:

- `skill-creator` — skill の作成/更新のためにローカルの `manage-skill` skill がラップします。
- `feature-dev` — ガイド付き機能開発ワークフロー（`/feature-dev`）。`code-explorer` / `code-architect` /
  `code-reviewer` エージェント付き。

信頼済みの clone には既に入っています。もし無ければ（または一覧に新規追加したら）、冪等な bootstrap を
実行します:

```bash
bash .claude/scripts/bootstrap-plugins.sh
```

新たに有効化したプラグインは **次の** Claude Code セッションで読み込まれます。

## 規約

- **英語が canonical。** skill / README 本文は命令形の英語で書き、対になる `*.ja.md` は `canonicalize-doc`
  skill で同期する人間向け参考訳です。ユーザーへの実行時出力は引き続き `CLAUDE.md` に従います（日本語）。
- **skill の著述。** skill の作成・更新には `/manage-skill` を使います。`skill-creator` をラップし、本
  リポジトリの配置・frontmatter・翻訳ペア・eval 成果物の規約を適用します。
- **既存物の探索。** 手書きの一覧を本 README で保守する代わりに、`/tool-map` を実行すると skill / agent /
  command の全インベントリと依存マップが得られます。
