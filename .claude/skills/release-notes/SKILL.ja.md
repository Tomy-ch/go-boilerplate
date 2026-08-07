> **このファイルは `SKILL.md` の日本語訳です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# リリースノート生成手順

このスキルは、指定された `origin` の git タグから現在の `HEAD` までの変更を要約した日本語のリリースノートを生成し、`.github/release/<NEW_VERSION>.md` に書き出すための作業手順を定義する。

正式な書式の参照例は以下を参照すること。

- `.github/release/v1.1.0.md`
- `.github/release/v1.0.0.md`

## 最初に行うこと: FROM タグと新バージョンの確認

このスキルでは、**スキル起動直後に必ず `AskUserQuestion` で 2 つの値を順に確認する**。
スキル引数や直近メッセージに値らしき文字列があっても、それを採用して即実行に進んではならない（誤指定を防ぐため、明示的な確認を必須とする）。

### 1. FROM タグ（比較元）

1. `git fetch --tags --prune origin` を実行し、ローカルのタグ参照を最新化する。
2. `origin` の最新 SemVer タグを収集する:

    ```sh
    git ls-remote --tags origin | awk -F/ '{print $NF}' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -5
    ```

3. `AskUserQuestion` を呼び出す:
    - 質問: 「比較元 (FROM) の origin タグを指定してください。」
    - 直近 3 件のタグを選択肢として提示し、それ以外は「Other」フォールバックで受ける。
4. 受け取った回答が `^v[0-9]+\.[0-9]+\.[0-9]+$` にマッチし、`git rev-parse <FROM>^{commit}` が成功することを検証する。値を `<FROM_TAG>` として後段で使用する。

### 2. 新バージョン（NEW_VERSION）

1. リポジトリ内のヘルパーで bump 候補を計算する:

    ```sh
    tsx scripts/semver <FROM_TAG> patch
    tsx scripts/semver <FROM_TAG> minor
    tsx scripts/semver <FROM_TAG> major
    ```

2. 現在ブランチ名から推測値も取得する（`git rev-parse --abbrev-ref HEAD`）。`release/v[0-9]+\.[0-9]+\.[0-9]+` にマッチする場合は、その値も追加候補として提示する。
3. `AskUserQuestion` を呼び出す:
    - 質問: 「新しいリリースのバージョン (NEW_VERSION) を指定してください。」
    - 選択肢: `scripts/semver/index.ts` の patch / minor / major 候補と、存在すればブランチ由来候補。
4. 受け取った回答が `^v[0-9]+\.[0-9]+\.[0-9]+$` にマッチすることを検証する。値を `<NEW_VERSION>` として後段で使用する。

両方の値が確定するまで、git 履歴の解析やファイル書き込みは一切行わないこと。

## 前提

- `<FROM_TAG>` は `origin` に存在し、`HEAD` から到達可能であること。
- 作業ツリーが dirty でも構わないが、リリースノートに反映されるのは `<FROM_TAG>..HEAD` のコミット済み差分のみ。`git status --porcelain` が空でない場合、書き込み前に一度ユーザーへ知らせる。
- 生成ファイルをコミットする際は、`production` / `develop` / `staging` / `release/*` ブランチで直接作業しないこと（AGENTS.md の Git ルール参照）。git 状態の*読み取り*は現在ブランチで構わないが、保護ブランチ上にいる場合は、適切な feature ブランチでコミットするようユーザーに案内する。

## AI Modification Scope について

このスキルは AGENTS.md の "Exception: Skill Execution" 節に基づき、スキル実行中に限り通常の AI Modification Scope の縛りを解放する。具体的には以下のパスへの**新規作成**がスキル実行中に許可される:

- `.github/release/<NEW_VERSION>.md`（新規作成のみ）

以下はスキル実行中も保護対象のまま:

- `AGENTS.md` / `CLAUDE.md`
- `.github/release/` 配下の既存ファイル（このスキルは既存ファイルの上書きを一切行わない。`.github/release/<NEW_VERSION>.md` が既に存在する場合は処理を中止してユーザーに確認する）
- 生成物（`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, `docs/` 配下の生成物）
- `.github/release/` 以外のすべて

## 実行手順

`<FROM_TAG>` と `<NEW_VERSION>` が確定したあとに以下を実行する。

### 1. ガード: 出力先ファイルが存在しないこと

```sh
test ! -e .github/release/<NEW_VERSION>.md
```

既に存在する場合は処理を中止し、進め方をユーザーに確認する（無断で上書きしないこと）。

### 2. 差分メタデータを収集する

`<FROM_TAG>..HEAD` から以下を取得する:

```sh
# コミット数
git rev-list --count <FROM_TAG>..HEAD

# 変更ファイル数と +/- 行数（diffstat サマリ）
git diff --shortstat <FROM_TAG>..HEAD

# コミットログ（マージ以外を先に、文脈としてマージも取得）
git log --no-merges --pretty=format:'%h %s' <FROM_TAG>..HEAD
git log --merges     --pretty=format:'%h %s' <FROM_TAG>..HEAD

# トップレベルディレクトリでまとめた変更ファイル一覧（スコープ推定用）
git diff --name-only <FROM_TAG>..HEAD | awk -F/ '{print $1}' | sort -u
```

個々のコミットの文脈をより詳しく見たい場合は以下:

```sh
git log --no-merges --pretty=format:'%h%n%s%n%b%n---' <FROM_TAG>..HEAD
```

### 3. コミットを分類する

各非マージコミットの subject を確認し、バケットに振り分ける。このリポジトリで使われている英語／日本語スタイルのプレフィックス（`Feat:` / `Fix:` / `Refactor:` / `Docs:` / `Chore:` / `Test:` など）に両対応する:

| バケット | 出力先セクション |
| --- | --- |
| Feat / Feature | `### 新機能・改善` |
| Refactor / Perf / Chore（実装に影響するもの） | `### 新機能・改善`（サブ箇条として） |
| Fix / Bugfix | `## 不具合修正` |
| Docs | `### 新機能・改善` → `#### ドキュメント整備` |
| Test / CI / Build | `### 新機能・改善` → `#### 開発ツールチェーンの同期` または `#### その他の改善` |

判断が曖昧な場合は、手順 2 で取得した変更ファイルパスからスコープを推定する（例: `database/` → DB 周り、`openapi/` → API、`.github/workflows/` → CI など）。

### 4. リリースノートを作成する

`.github/release/<NEW_VERSION>.md` を**日本語で**、`v1.1.0` 形式に従って書き出す。必須のトップレベル構造は以下:

```markdown
<!-- markdownlint-disable MD041 -->
## 概要

{2〜4 行で、このリリースの趣旨を要約する。FROM_TAG → NEW_VERSION の位置づけ（パッチ／マイナー／メジャー）に触れる。}

変更規模は以下です。

- `<N>` コミット
- `<M>` ファイル変更
- `+<INS> / -<DEL>`

## 変更内容

### 新機能・改善

#### {サブカテゴリ（例: ランタイム・ライブラリのアップデート / 開発ツールチェーンの同期 / ドキュメント整備 / その他の改善）}

- {変更内容を、コミットの羅列ではなく「読んで理解できる粒度」でまとめる}
  - {参照すべきコミット hash や PR があれば付記}

## 動作確認手順

1. {このリリースで変わった箇所を確認するための具体手順}
2. ...

## 影響（想定されるメリット／注意点）

- ✅ {メリット}
- ⚠️ {注意点（破壊的変更・運用変更・依存更新の影響など）}

## 追加ライブラリ

- {追加された依存。無ければ「なし（既存依存のバージョン更新のみ）」}

## 不具合修正

- {Fix 系コミットを文章化したもの}

## 補足・備考

- {OpenAPI の破壊的変更の有無、portal 反映、リリースブランチ運用などのメモ}
```

内容に関するルール:

- **コミット subject の貼り付けで済ませない。** 読んで理解できる粒度の日本語文に要約する。
- **時系列ではなくテーマでグルーピングする。**
- **具体のファイルパス／コンポーネント名を引用する**と、読者が変更箇所に辿りやすい（例: `scripts/semver/index.ts`、`internal/controller/handler/...`）。
- **空セクションを捏造しない。** 該当する変更がなければ `- 該当なし` と書く。
- **既存のトーンに合わせる。** `.github/release/v1.1.0.md` の文体を比較対象として参考にする。

### 5. 書き込み前にプレビューを提示する

`Write` を呼ぶ前に、作成内容をユーザーに提示し（インライン、または要約 + ファイルパス）、最後にもう一度 `AskUserQuestion` で確定する:

- 質問: 「この内容で `.github/release/<NEW_VERSION>.md` に書き出してよいですか？」
- 選択肢: 「書き出す」 / 「修正したい箇所を指摘する」

ユーザーが確定したあとに限り `Write` を実行する。

### 6. Markdown Lint による検証

書き込み後、以下を実行する。

```sh
make md-fix
make md-lint
```

`make md-fix` はリポジトリ全体に対して `markdownlint-cli2 --fix` を実行し、よくある違反（見出し / リスト / コードブロック周辺の空行、行末空白、ファイル末尾の改行など）を自動修正する。続けて `make md-lint` で `.markdownlint-cli2.yaml` 準拠かを検証する。

`make md-lint` がエラーを報告する場合:

1. lint 出力を確認する。
2. 自動修正で解消できないルール（見出し階層、重複見出し、bare URL など）を手で修正する。
3. clean になるまで `make md-fix` → `make md-lint` を繰り返す。

`make md-lint` がクリーン終了するまでスキルを完了報告しない。

`make md-fix` はリポジトリ全体を対象にするため、本リリースノートとは無関係な Markdown も自動修正される可能性がある。その場合、変更された他ファイルの一覧を完了報告時にユーザーへ提示し、レビューできるようにする。

### 7. 最終確認

書き込みおよび lint 後:

- `.github/release/<NEW_VERSION>.md` が存在することを確認する。
- ステージ・コミット・プッシュは行わない。ファイルを作成した旨をユーザーに伝え、git 操作は AGENTS.md のルールに従ってユーザー自身に委ねる。

## チェックリスト

完了報告の前に以下を確認する:

- [ ] `<FROM_TAG>` を `AskUserQuestion` で確認し、`origin` に存在することを検証した
- [ ] `<NEW_VERSION>` を `AskUserQuestion` で確認し、SemVer に対して検証した
- [ ] `.github/release/<NEW_VERSION>.md` がまだ存在しない
- [ ] 差分メタデータ（コミット数 / ファイル数 / +/- 行数）を `git` から取得した
- [ ] コミットを `v1.1.0` 形式のセクションに分類した
- [ ] リリースノートを日本語で、正規フォーマットに沿って起草した
- [ ] プレビューをユーザーが承認した
- [ ] `.github/release/<NEW_VERSION>.md` を書き出した
- [ ] `make md-lint` がクリーン終了する
- [ ] git 操作を行っていないことをユーザーに伝えた

## 注意事項

- 出力ファイル本文の言語は `CLAUDE.md` の言語ルールに従い**日本語**とする（対話中のやり取りの言語に関わらず）。
- このスキルは amend、force push、tag 操作、push を一切行わない。git 操作はユーザーの責任で行う。
- `SKILL.md` を更新したあとは、本ファイル `SKILL.ja.md` も同期更新すること。
