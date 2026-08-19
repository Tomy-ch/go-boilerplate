# リポジトリ設定

[English](README.md) | 日本語

このディレクトリには、リポジトリ単位の GitHub 設定を JSON で格納しています（ブランチルールセットとラベル定義）。このボイラープレートから派生したリポジトリを、Web UI をクリックして回るのではなく再現可能な形で構成するためのものです。

## これらは意図の宣言であり、実態の写しではない

JSON は一方向の適用に渡す入力であって、GitHub が現在強制している内容のミラーではありません。

- `make apply-branch-protection` が `branch-protection.json` を `branch-protection` という名前のリポジトリルールセットとして送り、`make create-default-labels` が `labels.json` を送ります。いずれも派生リポジトリの初期化時に `make setup-repo` から一度だけ呼ばれます。
- 以後これらを再実行する仕組みも、実態と突き合わせる仕組みもありません。Web UI からルールが外された場合も、JSON を変更したまま適用を走らせなかった場合も、宣言と実態は何の兆候も無いままずれます。

したがってこのディレクトリが答えるのは「このリポジトリが何を強制するつもりか」であって、「このリポジトリが何を強制しているか」ではありません。後者は GitHub に直接問い合わせてください。

```sh
gh api /repos/{owner}/{repo}/rulesets
gh api /repos/{owner}/{repo}/rulesets/{ruleset_id}
```

いずれもリポジトリの通常の read 権限で参照でき、public リポジトリなら未認証でも参照できます。実態の確認に管理者権限のトークンは要りません。

## branch-protection.json

`conditions.ref_name.include` の対象は `production` / `staging` / `develop` / `release/**/*` / `hotfix/**/*` です。各ルールの宣言内容は次のとおりです。

| ルール | 宣言している内容 |
| --- | --- |
| `deletion` | 対象ブランチを削除できない。 |
| `non_fast_forward` | 対象ブランチへの force-push を拒否する。 |
| `pull_request` | 対象ブランチへの変更は PR 経由に限る。承認 1 件、push 時に既存の承認を破棄、最終 push 後の再承認、全レビュースレッドの解決、CODEOWNERS レビュー、マージ方法は merge commit か squash に限定（rebase merge を除外）。 |
| `copilot_code_review` | Copilot が各 PR を自動レビューする。push のたび、および draft に対しても実行する。 |
| `code_quality` | GitHub の code quality ルールを `errors` 深刻度でブロックする。 |
| `required_status_checks` | guard を持つ 7 件のチェックが成功してからマージする。 |

### 単独メンテナのリポジトリに `pull_request` を適用する場合

GitHub では自分の PR を自分で承認できません。`required_approving_review_count: 1` かつ `bypass_actors` が空の場合、参加者がオーナー 1 人だけのリポジトリではこのルールを満たせる人間が存在せず、保護ブランチ向けの PR がすべて恒久的にマージ不能になります。そうしたリポジトリへ適用する前に、メンテナを `bypass_actors` に列挙するか、承認を要求するパラメータを両方とも下ろしてください（`required_approving_review_count: 0` と `require_last_push_approval: false`）。スレッド解決とマージ方法の制限はそれ単体でも効きます。

### `code_quality` は裏側の機能が先に報告している必要がある

GitHub の案内は、ruleset に Code Quality の閾値を宣言する**前に** Code Quality のワークフローが動作し PR へ結果を報告していることを確認せよ、というものです。確認しないまま宣言すると、このルールが全 PR のマージをブロックし得ます。機能の有効化はこのディレクトリの外にあるリポジトリ単位の操作なので、機能が無効なら無害だと仮定せず、適用前に確認してください。

### Required status check はすべての PR で報告経路を持つ必要がある

宣言する required context は `trivy-fs-release`、`osv-release`、`trivy-config`、`sast`、
`lockfile-lint`、`openapi-security`、`osv-diff` です。それぞれの `*-guard.yaml` が、path または
branch filter で本体 workflow が skip されたときも同じ context を報告します。これが無いと、開始されない
check を PR が永久に待つことになります。設計は `docs/adr/0089-multi-layer-security-scanning.md` を参照してください。

## labels.json

`make create-default-labels` が作成するラベル定義（`name` / `description` / `color`）です。`make setup-repo` は先に `make delete-all-labels` を実行するため、このファイルは GitHub が既定で作るラベルへの追加分ではなく、意図するラベル集合そのものです。
