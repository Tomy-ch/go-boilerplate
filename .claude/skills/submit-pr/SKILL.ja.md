> **このファイルは `SKILL.md` の日本語訳です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# Submit PR

このスキルは、現在のブランチを `origin` に push し、対応する GitHub プルリクエストが存在することを保証する。次の 2 ケースを自動で判別して処理する。

- **Create**: 現在のブランチに対する PR が存在しない場合 → `-u` 付き push（upstream 未設定時）と新規 PR 作成を行う。
- **Update**: 既存の open PR が存在する場合 → ユーザーに確認してから push する（PR 上の diff は自動で更新される）。

PR の本文は `.github/pull_request_template.md` をひな型にして埋める。自動 push は行わず、既存 PR のタイトル / 本文を勝手に上書きせず、force push もしない。

## 前提

- `gh` CLI がインストールされ、認証済みであること（`gh auth status` が成功する）。
- 現在のブランチが保護ブランチ（`production` / `develop` / `staging` / `release/*`）ではないこと。
- working tree がクリーンであること。未コミット変更がある場合はスキルを中断し、先に `/commit` を実行するよう促す。

## Step 0. 事前チェック

並列で実行:

```sh
git rev-parse --abbrev-ref HEAD                          # 現在のブランチ
git status --porcelain                                   # working tree の状態
git rev-parse --verify '@{u}' 2>/dev/null                # upstream の有無
git log '@{u}'..HEAD --oneline 2>/dev/null               # 未 push コミット（upstream がある場合）
gh auth status
```

以下のいずれかに該当する場合は中断する。

- ブランチが `^(production|develop|staging|release/.+)$` にマッチする → feature ブランチに切り替えるようユーザーに伝える。
- `git status --porcelain` が非空 → 先に `/commit`（または stash）を実行するようユーザーに伝える。
- `gh auth status` が失敗 → `gh auth login` の実行をユーザーに依頼する。

Step 1 に進むときの状態は以下の 4 パターン。

| upstream | 未 push コミット | 意味 |
| --- | --- | --- |
| なし | n/a | 初回 push |
| あり | > 0 | 追加 push |
| あり | 0 | push 不要だが PR が未作成の可能性 |
| あり | 0 + PR open | 何もすることが無い（Step 1 で処理） |

## Step 1. 既存 PR とベースブランチの検出

```sh
gh pr view --json number,state,baseRefName,headRefName,url,title,body 2>/dev/null
gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'
```

結果に応じて分岐する。

- **PR が存在し state が `OPEN`** → "update" 経路。ベースブランチは固定（結果の `baseRefName`）。
- **PR が存在するが state が `MERGED` / `CLOSED`** → `AskUserQuestion` でユーザーに確認:
  - 質問: 「このブランチには `<state>` 状態の PR #N があります。新規 PR を作成しますか？」
  - 選択肢: 「新規 PR を作成する」 / 「キャンセル」
- **PR が存在しない** → "create" 経路。ベースブランチはリポジトリのデフォルトブランチ。

"create" 経路で、ローカルに複数の `release/*` ブランチがある等、デフォルト以外を対象にしたい可能性がある場合は `AskUserQuestion` で確認:

- 質問: 「ベースブランチをこれで作成しますか？」
- 選択肢: 「`<default-branch>` を使う」 / 「別のブランチを指定する」

早期終了の特殊ケース:

- "update" 経路で未 push コミットが 0 件 → push する必要が無い旨を伝えて終了。既存 PR の URL を表示する。
- "create" 経路で未 push コミットが 0 件だがリモートブランチは存在 → Step 2 へ進む（リモートに既にある内容で PR を作成する）。

## Step 2. コンテキスト収集とテンプレート読み込み

タイトル・本文を組み立てるための入力を収集する。`<base>` は Step 1 で確定したベースブランチ。

```sh
git log <base>..HEAD --pretty=format:'%h %s'                # コミットタイトル
git log <base>..HEAD --pretty=format:'%h%n%s%n%b%n---'      # コミットタイトル + 本文
git diff <base>...HEAD --shortstat                          # diff サマリ
git diff <base>...HEAD --name-only                          # 変更ファイル
```

`.github/pull_request_template.md` を読み、`#` / `##` ヘッダでセクションを識別する。現行テンプレートは以下のセクションを持つ。

- `# 概要`
- `## 変更内容`
- `## 動作確認方法`

HTML コメントのプレースホルダーは取り除く。テンプレートが存在しない場合は、同じ 3 セクション構成をインラインのフォールバックとして使う。

## Step 3. タイトルと本文の生成

### タイトル

- 最も大きい変更から導出する。単一コミットの PR ならそのコミットタイトルを使う（冗長な場合のみ先頭の `<Prefix>:` を外す）。複数コミットなら全体の意図を日本語で要約する。
- 70 文字以内。
- ブランチ名に issue 番号が埋め込まれている場合（`feature/1234-...`、`bugfix/5678-...`）、`#1234` を自然な形でタイトルに含める。
- "update" 経路では、ユーザーから明示的な指示がない限り既存タイトルを変更しない。

### 本文

テンプレートの各セクションを日本語で埋める。

- **概要**: PR の意図を 1〜3 文で要約。主にコミットメッセージから抽出する。
- **変更内容**: 領域別の箇条書き（API / DB / 内部ロジック / テスト / ドキュメント など）。変更ファイルとコミットタイトルを参照する。生のファイル一覧の貼り付けは避け、意味のある粒度でまとめる。
- **動作確認方法**: 具体的な確認手順。実際の変更内容に合わせて適応する（API 変更なら `make serve` + curl、マイグレーションなら `make db-local-migrate-up`、ロジックなら `make test` など）。

ブランチ名から issue 番号が拾えれば、本文末尾に `closes #N` を追加する（自然なら 概要 に折り込んでも可）。

## Step 4. ユーザー確認

確定したタイトル、ベースブランチ、push コマンド、本文全文を表示する。

### Create 経路

`AskUserQuestion`:

- 質問: 「以下の内容で PR を作成しますか？」
- 選択肢:
  - 「この内容で作成する」
  - 「draft で作成する」
  - 「title / body を修正したい」
  - 「キャンセル」

「修正したい」が選ばれた場合、自由記述のフィードバックを収集し、該当セクションを再生成して再確認する。

### Update 経路

未 push コミット一覧と diff サマリを表示してから、`CLAUDE.md` で指定された文面で確認する。

- 質問: 「変更はローカルにコミット済みです。これらの変更をプルリクエストにプッシュしますか？」
- 選択肢: 「push する」 / 「キャンセル」

## Step 5. push

```sh
# 初回 push (upstream 未設定)
git push -u origin <branch>

# 2 回目以降
git push
```

ユーザーから明示指示がない限り `--force` / `--force-with-lease` は使わない。

push が失敗（non-fast-forward、権限エラー、ネットワークエラー等）した場合は、エラー内容をそのままユーザーに伝えて停止する。自動復旧は試みない。

## Step 6. PR の作成 / 更新

### PR を作成する

```sh
gh pr create \
  --base "<base-branch>" \
  --title "<title>" \
  --body "$(cat <<'EOF'
<body>
EOF
)" [--draft]
```

### PR を更新する

Step 5 の push で既に PR の diff は更新されている。デフォルトでは PR のタイトル・本文には触れない。

ユーザーから明示的に更新指示があった場合のみ、以下を実行する。

```sh
gh pr edit <number> [--title "<new-title>"] [--body "$(cat <<'EOF'
<new-body>
EOF
)"]
```

## Step 7. 結果報告

PR の URL と簡単な要約を日本語で表示する。

Create 経路:

```text
PR を作成しました: <url>
ベース: <base-branch>
タイトル: <title>
コミット数: N
```

Update 経路:

```text
PR を更新しました: <url>
追加コミット数: N
```

## 制約

- ❌ 保護ブランチ（`production` / `develop` / `staging` / `release/*`）への push
- ❌ `git push --force` / `--force-with-lease`（ユーザーから明示指示があった場合のみ可）
- ❌ 既存 PR のタイトル・本文の自動更新（明示指示があった場合のみ可）
- ❌ working tree に未コミット変更があるまま push する
- ❌ ユーザー確認なしで PR を作成する
- ❌ 既存 PR ブランチへの push を、`CLAUDE.md` 指定の文面で再確認せずに実行する
- ✅ `.github/pull_request_template.md` を本文のひな型として使う
- ✅ タイトル・本文は日本語
- ✅ `gh pr create` / `gh pr edit` の body は HEREDOC で渡す
- ✅ ブランチ名から issue 番号を検出してタイトル / 本文に反映する

## チェックリスト

完了報告の前に以下を確認する。

- [ ] 現在のブランチが保護ブランチではない
- [ ] push 前の working tree がクリーンだった
- [ ] `gh auth status` が成功した
- [ ] PR テンプレートを読み、本文に反映した
- [ ] タイトル・本文が日本語である
- [ ] タイトルが 70 文字以内である
- [ ] push 前にユーザー確認を取得した（update 経路では `CLAUDE.md` 規定文面で必須）
- [ ] PR URL をユーザーに伝えた
- [ ] `--force` 系を使っていない
