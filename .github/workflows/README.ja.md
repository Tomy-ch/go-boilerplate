# GitHub Actions Workflows

[English](README.md) | 日本語

このディレクトリには CI/CD 用の GitHub Actions ワークフロー定義を格納しています。ワークフローは目的別にグルーピングされており、PR ゲート（lint / test / セキュリティスキャン）、push 起点のデプロイ、リリースブランチ起点のドキュメント再生成という構成です。

## トリガー戦略

| グループ | 発火タイミング | 目的 |
| --- | --- | --- |
| CI チェック | 全 PR | lint / test / 生成物整合性が失敗したらマージブロック |
| セキュリティ | ツールごとのマトリクス（後述） | コード / 依存 / イメージ / ワークフロー定義 / コミット済みシークレットの問題を surface |
| デプロイ | `production` / `staging` / `develop` への push | 成果物ビルド、マイグレーション実行、アプリ / docs portal をデプロイ |
| ドキュメント | `release/*` への push | OpenAPI / ER / portal ドキュメントを再生成し auto-sync PR を作成 |
| アシスタント | プルリクエストでの `@claude` メンション | オンデマンドで回答・調査する。書き込み権限を持つアカウントに限定 |

`gen-*-artifacts-check` 系のワークフローが守る不変条件は「コミットされた生成物が生成器から再現できる」ことです。この不変条件は「入力が変わったのに生成物が再生成されていない」「生成物が直接書き換えられた」の 2 方向で崩れるため、`on.pull_request.paths` には**入力と生成物の双方**を列挙する必要があります。入力側だけを見張ると、生成物のみを触った PR に対して構造的に盲目になり、壊れた生成物がそのままベースブランチへ入って次の無関係な PR が代わりに赤くなります。

これらのワークフローは生成器のバージョンを `mise.toml` で固定しているため、同ファイルは多くの検査にとって入力です。`paths` フィルタはファイル単位でしか判定できないので、この共有ロックファイル内の無関係なツールを更新しただけでも検査が起動します（Postgres を伴う `gen-db` ジョブを含む）。この過剰起動は意図的に受け入れています。生成器のバージョン更新こそ再検証すべき変更であり、トリガを絞るために `mise.toml` を分割するコストの方が、たまの余分な実行より大きいためです。

## 結果コメント

このリポジトリの PR では 30 件ほどの検査が走り、そのほとんどがコメントを投稿しえます。通過した検査ごとのコメントは誰も尋ねていないことを述べるだけで、言うべきことがある唯一のコメントを埋もれさせます。したがって結果コメントは**その検査に報告すべきことがあるときにだけ作成します**。この判定を運ぶのが `upsert-pr-comment` の `status` 入力で、コメントを抑止する値はリテラル `success` ただ 1 つです。それ以外の値は、打ち切られたジョブが残す空文字も含めて、すべて投稿します。

修正されたことは沈黙で伝えません。`success` が抑止するのはコメントの *作成* だけで、更新は抑止しません。以前の push で失敗した検査は、自分の赤いコメントを緑の結果でその場に上書きします。PR にはもう成り立たない失敗が残らず、読み手は失敗が報告された場所で解決を知ることができます（不在に気付く必要がありません）。削除にすると修正の記録が残らず、再発時にはコメントが、既に読み進められたスレッドの末尾へ移動してしまいます。

**上書きではなく削除する例外が 1 つだけあります。打ち切り（CUT OFF）通知です。** これは判定を記録しておらず「ある実行が判定に到達しないまま終わった」ことしか述べていないため、後続の `success` は所見を上書きしているのではなく、開いたままの問いに答えているだけで、消しても失われる情報がありません。上書きすると、キャンセル 1 回と引き換えに恒久的な緑コメントが PR に残ります。しかもキャンセルは日常的です — `cancel-in-progress` は 2 回目の push のたびに前の実行を打ち切り、出力を流しながら打ち切られたジョブは書きかけの本文を残すので、通知が作成されてそのまま緑で固定されます。打ち切り自体はどちらにせよ実行履歴に残ります。判定は `CUT OFF (no result produced)` の見出しで行います。本文を書きながら打ち切られたジョブは本文を持つため action 自身の通知文には到達せず、共通するのは呼び出し側が渡す見出しだけで、それを必須にしているのが `make actions-cutoff-lint` です。トークンが削除を拒否した場合は上書きへフォールバックするので、緑の実行に打ち切り表示が残ることはありません。

**判定は肯定形で導きます — 明示的にクリーンな信号のときだけ黙ります。** ステップが既に `status` 出力を持つ場合は呼び出し側がそれをそのまま渡し、クリーン状態が件数やフラグで表される場合は所見側ではなくその値そのものを検査します（`steps.<id>.outputs.count == '0' && 'success' || 'findings'`）。差が出るのは本文を作るステップが走らなかったときです。出力は空になり、それはクリーンな値ではないので検査は報告側へ倒れます。逆向きに書くと、完走しなかった実行がすべて合格に見えてしまいます。title の `✅` と `success` は構造上同じことを意味します — 非ブロッキングな所見を `✅` とする側（`osv-scan`）はそれについて黙り、`⚠️` とする側（`sast`）は黙りません。

定数 `report` を渡す呼び出しは `image-scan.yaml` の 2 件だけです。SBOM のインベントリと Trivy の表は判定ではないので「言うべきことが無い」状態を持たず、またこのジョブはデプロイブランチ向けの PR でしか走らず、そこではイメージの中身こそがレビュー対象だからです。

コメントはそれを書いた実行より長く残ることがあります。`paths` フィルタにより後続の push でワークフローが走らないことがあり、赤いコメントが、検査していない head の上に立ったまま残ります。各コメントが持つ Commit / UpdatedAt のフッタが現行のものと区別する手がかりであり、権威ある状態はチェックラン側にあります。

## ジョブの打ち切り

ジョブは判定に到達しないまま止まることがあります（タイムアウト、キャンセル、ランナー障害）。そのとき PR 上で何が読めるかは、走っていたツールの性質ではなく、ジョブとコメントステップをどう宣言したかで決まります。しかもここでの既定値はどれも誤った側に倒れています。以下の規約は `make actions-cutoff-lint` で機械的に守っています。このディレクトリの全コメントステップと全ジョブを目視で維持することはレビューに要求できないためです。

**`upsert-pr-comment` を呼ぶステップは、キャンセル後にも到達できなければなりません。** Actions はステータスチェック関数を含まないカスタム `if:` に暗黙で `success() &&` を前置するため、打ち切られたジョブではコメントステップがスキップされ、PR には何の痕跡も残りません。一方で `Fail if …` 側は `always()` を持つことが多く、チェックは赤くなります。理由の読めない赤は、どちらか片方の欠落より悪い状態です。したがって条件には `always()` か `cancelled()` が要ります。`failure()` は**該当しません** — キャンセルされたジョブでは false になるためです。現在はすべての呼び出しが素の `always() && github.event_name == 'pull_request'` です。コメントすべきかどうかはステップをスキップすることではなく action 内の `status` が決めます。走らなかったステップは、以前の push が残したコメントを訂正できないためです。

**本文ファイルの不在は、ステップの失敗ではなく打ち切りとして報告します。** 早期に打ち切られたジョブはファイルを書くステップに到達しないので、不在はまさにコメントが生き残るべきケースの通常形です。`upsert-pr-comment` は打ち切りの通知を投稿し、呼び出し側の見出しを置き換えます — ジョブが走らなかったと書く本文に対して、呼び出し側が設定した title はもう何も説明していないためです。通知は原因を名指ししません。本文が無い理由には前段ステップがそのまま失敗した場合もあり、両者を区別できるのは実行ログだけだからです。代償として `body-file` のパスを誤配線すると、失敗ではなく緑の実行に打ち切り通知が出ます。これは気付ける程度に喧しく、打ち切り時の沈黙はそうではありません。`cancel-in-progress` のもとでは、打ち切られた実行がこの通知を投稿した直後に新しい実行が同じ marker を上書きすることがあります。これは機構が働いている姿であって、直すべき不具合ではありません。

**不在だけでは判定の半分にしかならないので、呼び出し側も打ち切り時の見出しを渡します。** 多くの検査ステップは出力を `tee` でそのまま本文ファイルへ流し、`title` は終了コードを見たあとに出力します。検査の途中で打ち切られると、そのジョブは**書きかけのファイル**を残すため、action からは完成した本文と区別がつきません。一方 title は出力されないままです。そこで呼び出し側が判定のもう半分を `${{ steps.<id>.outputs.title || '## ⚠️ <検査名>: CUT OFF (no result produced)' }}` の形で持ちます。フォールバックは本文を作るステップが自らの結論に到達しなかったときにちょうど発火し、書きかけのログはその下にそのまま残ります。見出しがステップ出力ではなくリテラルの箇所（`image-scan.yaml` / `sync-versions-check.yaml`）では、同じ判定をそのステップの `outcome` / 出力への条件として書きます。書くときは GitHub の式の罠に注意してください。`cond && '' || X` は空文字が falsy なので常に `X` になります。見出しは真の側の枝に置く必要があります。

**全ジョブに `timeout-minutes` を置きます。** 無いと GitHub 既定の 360 分まで走るため、1 つのハングがランナーを 6 時間占有します。値は「実測の最大所要 × 3 を 5 分単位で切り上げ、下限 10 分」です。下限は混雑したランナーでのセットアップ変動を吸収するためで、直近に完了実行が無いジョブは 15 分とします。この式から外れる値だけを以下に挙げます。他は全て下限であり、値は一覧を引くのではなく式から再導出できます。

| ジョブ | 分 | 式から外した理由 |
| --- | --- | --- |
| `auto-generate-docs.yaml` `generate-docs` | 25 | 実測 約 7 分 |
| `go-test.yaml` `go-test` | 20 | 実測 約 5 分 |
| `image-scan.yaml` `build`、`deploy-app.yaml` `build` | 15 | レイヤキャッシュが冷えたイメージビルドは実測を大きく超えて振れる |
| `deploy-app.yaml` `deploy` | 30 | 現状はプレースホルダ。フォークが実デプロイを配線したときに 10 分の上限へ当てない |
| `fuzz.yaml`、`scorecard.yaml`、`notify.yaml`、`osv-release-gate.yaml` | 15 | 直近に完了実行が無く実測できない |
| `zap-api-scan.yaml` `dast` | 30 | 完了実行が無く実測できないうえ、スキャンの前にアプリケーションをビルドして起動し、スキャン自体の長さは OpenAPI 定義の規模で決まる <!-- dast:line --> |
| `code-ql.yaml` `codeql` | 30 | 上限は matrix の最も遅い leg に掛かるが、`go` 以外の leg には完了実行が無く実測できない。加えて `security-extended` は従前の値を測ったスイートより大きい |
| `secret-scan.yaml`、`trufflehog.yaml` | 15 | 実測は差分を見る PR 実行のみ。週次は全履歴を走査するが、その完了実行が一度も無く実測できない |
| `app-di-startup-check.yaml`、`gen-go-artifacts-check.yaml` | 15 | 式より前から存在する値。動いている上限を下げてもリスクしか増えないためそのまま |
| `claude.yaml`、`go-lint.yaml`、`sample-removal-check.yaml` | 30 | 同上。`go-lint` は golangci-lint 自身の timeout を無効化して走らせているため、これがそのジョブ唯一の打ち切り点でもある |

上限に当たり始めたジョブは実測を追い越しているということなので、数字を小突くのではなく測り直して式に掛け直します。reusable workflow を呼ぶジョブには `timeout-minutes` を書けません（invalid key）。検査はそれらを除外し、上限は呼ばれる側のジョブが持ちます。

3 つの規約を 3 本ではなく 1 本の検査に置いているのは、これらが 3 つのポリシーではないからです。上限の無いジョブが打ち切りを生み、コメント側の 2 つがそれを読めるようにします。どれか 1 つだけを直しても PR 上の状況は改善しないので、単独で走らせる理由がありません。

## ワークフロー一覧

### CI チェック（Pull Request）

|ワークフロー|ファイル|説明|
|---|---|---|
|Go Lint|`go-lint.yaml`|golangci-lint による Go コードの静的解析|
|Go Test|`go-test.yaml`|Go テスト実行とカバレッジレポート、およびカバレッジゲート対象外の `scripts/` ツールテスト|
|Module Tidy Check|`tidy-check.yaml`|go.mod / go.sum の整合性検証|
|SQL Lint|`sql-lint.yaml`|sqlfluff による migration / DML / seed SQL の検証|
|Actions Lint|`actions-lint.yaml`|actionlint によるワークフロー定義の検証、composite action の `run:` スクリプトの shellcheck 検査、PR コメント本文への secret 混入・固定長フェンスの検査、ジョブ打ち切り時の振る舞いの検査|
|Migration Check|`migration-check.yaml`|マイグレーションファイルの検証（重複、欠番、up/down ペア）|
|Sync Versions Check|`sync-versions-check.yaml`|mise.toml のバージョンが go.mod / 各 Dockerfile / README へ伝播済みか検証|
|Generated Go Artifacts Check|`gen-go-artifacts-check.yaml`|生成済み Go コードとコミット済み成果物の一致検証|
|Generated Database Artifacts Check|`gen-db-artifacts-check.yaml`|生成済み sqlc コードとコミット済み成果物の一致検証|
|Generated OpenAPI Artifacts Check|`gen-oapi-artifacts-check.yaml`|OpenAPI バンドルとドキュメントの一致検証|
|Generated Mock-Auth OpenAPI Artifacts Check|`gen-mock-auth-oapi-artifacts-check.yaml`|mock-auth-server の OpenAPI バンドル / zod スキーマ / ドキュメントとコミット済み成果物の一致検証|
|Mock-Auth Server Check|`mock-auth-server-check.yaml`|mock-auth-server の型検査、ユニット / インテグレーションテスト、golden JWKS フィクスチャのドリフト検出|
|Portal Check|`portal-check.yaml`|ドキュメントポータルのビューアー（`docs-viewer/`）の型検査とテスト|
|Scripts Check|`scripts-check.yaml`|リポジトリの TypeScript 補助スクリプト（`scripts/**/*.ts`）の型検査と、判定ロジックを覆う単体テストを実行します|
|OpenAPI Lint|`oapi-lint.yaml`|OpenAPI 定義を `redocly lint` で検証（命名 / casing / description / 未使用コンポーネント）|
|App Boot Check|`app-di-startup-check.yaml`|DB 付きでアプリケーションサーバが正常に起動するか検証|
|Job Boot Check|`job-boot-check.yaml`|ジョブのエントリポイントが起動し、未知のジョブを拒否するか検証|
|Worker Boot Check|`worker-boot-check.yaml`|worker のエントリポイントが起動（DI / DB）し、未知の worker を拒否するか検証|
|Dockerfile Lint|`docker-lint.yaml`|hadolint による Dockerfile の検証（go_tool_runner 経由）|
|Markdown Lint|`md-lint.yaml`|markdownlint による Markdown 体裁の検証、実 mermaid パーサによる ` ```mermaid ` フェンスの構文検証、`.claude/**` のスキル / エージェント定義の実態一致と `.codex/**` との存在対応の検証|
|Commitlint|`commitlint.yaml`|PR が base ブランチへ加えるコミットのメッセージを検証（`commit-msg` フックが覆えない経路を担う）|
|Pin Actions Check|`pin-actions-check.yaml`|GitHub Actions が SHA でピン留めされているか検証（サプライチェーン対策）|
|Pin Images Check|`pin-images-check.yaml`|Docker base image が lockfile 通り digest でピン留めされているか検証（サプライチェーン対策）|

### セキュリティ

|ワークフロー|ファイル|説明|
|---|---|---|
|CodeQL Scan|`code-ql.yaml`|`security-extended` スイートでの CodeQL 解析。言語ごとに matrix を分け、`go` / `javascript-typescript`（mock-auth-server / docs-viewer / scripts）/ `actions`（ワークフロー定義そのもの）を対象とする|
|Dependency Scan|`trivy-fs.yaml`|Trivy によるライブラリ脆弱性スキャン(開発者向け)|
|Release Dependency Scan|`trivy-release-gate.yaml`|develop/staging/production 向け PR での Trivy 依存スキャン|
|Image Scan|`image-scan.yaml`|Docker イメージビルド + SBOM 生成（SPDX-JSON と CycloneDX-JSON の両形式）+ Trivy スキャン|
|Vulnerability Scan|`vulnerability-check.yaml`|govulncheck による Go パッケージ脆弱性検出|
|OSV Scan|`osv-scanner.yaml`|OSV データベースによる Go モジュール / npm lockfile 横断の脆弱性スキャン|
|Release OSV Scan|`osv-release-gate.yaml`|develop/staging/production 向け PR での OSV スキャン。HIGH 以上で fail|
|Secret Scan|`secret-scan.yaml`|ワーキングツリーに対する 2 系統の独立したシークレット検出。gitleaks（正規表現 / エントロピーの広い網）と Trivy（誤検知の少ない固定ルール）を、判定を分けた別ジョブとして実行する|
|Secret Scan (TruffleHog)|`trufflehog.yaml`|TruffleHog による**検証済み**シークレット（実際に有効なクレデンシャル）の検出|
|Actions Static Analysis|`zizmor.yaml`|zizmor によるワークフロー / composite action 定義自体の静的解析（pre-commit フックと同じ `make` ゲートを共有）|
|Dependency Review|`dependency-review.yaml`|PR が新たに持ち込む脆弱な依存をマージ前にブロック|
|OpenSSF Scorecard|`scorecard.yaml`|リポジトリのセキュリティ姿勢のスコアリングと結果の公開|
|npm Cooldown Audit|`npm-cooldown-audit.yaml`|lockfile が `.npmrc` の供給網 cooldown を満たしているかを報告（ブロックはしない）|
|Go Cooldown|`go-cooldown.yaml`|cooldown 窓の内側で公開された direct Go モジュールを足す / 上げる PR をゲート|
|Tool Cooldown|`tool-cooldown.yaml`|cooldown 窓の内側で公開された CLI ツール版（`mise.toml` / `python/*.in` の宣言）を pin する PR をゲート|
|Config Scan|`trivy-config.yaml`|Trivy による Dockerfile の設定不備スキャン（HIGH 以上でゲート）|
|SAST|`opengrep.yaml`|Opengrep（Semgrep 互換）による自前の Go / TypeScript ソースの解析（taint 追跡あり）|
|Lockfile Integrity|`lockfile-integrity.yaml`|npm の `resolved` URL が正規レジストリかつ HTTPS であることの検証|
|OpenAPI Security|`openapi-security.yaml`|Spectral + OWASP API Security ルールセットによる OpenAPI 定義の検証|
|Fuzz|`fuzz.yaml`|外部入力を受けるパーサに対する Go ネイティブ fuzzing|
|DAST|`zap-api-scan.yaml`|ランナー内で起動したアプリケーションに対する、OpenAPI 定義を入力とした OWASP ZAP の API スキャン（報告専用のサンプル。[DAST](#dast) を参照） <!-- dast:line -->|
|Capability Diff|`capability-diff.yaml`|capslock による Go 依存グラフの capability 差分報告（report-only）|
|Notify|`notify.yaml`|定期実行の失敗、および非ブロッキングなスキャナの検出を人へ届ける `workflow_call` の再利用ワークフロー|

各スキャナは可能な限り SARIF を GitHub code scanning へ送り、検出は共通の `upsert-pr-comment` アクションで PR にコメントします（そもそもコメントを書くのがどういうときかは [結果コメント](#結果コメント) を参照）。

#### セキュリティのトリガーマトリクス

各ツールは「結果が実際に変わりうる場所」で走らせています。PR はその変更自身が持ち込むリスクを surface し、protected branch への push はブランチ保護が判断材料にする code scanning のベースラインを残し、定期実行は「コードが変わらなくても結果が変わる」種別（新規公表 CVE / 新規クエリ）にだけ設けます。

| 種別 | PR | protected branch への push | 定期 |
| --- | --- | --- | --- |
| gitleaks | 全 PR | 不要 | 週次で履歴全体 |
| Trivy secret | 全 PR・作業ツリー | 不要 | 週次 |
| TruffleHog | 全 PR の差分 | 不要 | 週次で履歴全体 |
| zizmor | Actions 関連ファイル変更時 | `develop` / `staging` / `production` / `release/*` | 週次（オンライン監査） |
| Dependency Review | 依存関係変更 PR | 不要 | 不要 |
| govulncheck | Go・依存変更 PR | 同上 | 週次 |
| Trivy FS | Go・依存変更 PR | 同上 | 週次 |
| OSV-Scanner | 依存関係変更 PR | 同上 | 週次 |
| CodeQL | Go / TypeScript / Actions 定義の変更 PR | 同上 | 週次 |
| OpenSSF Scorecard | 不要 | 既定ブランチのみ | 週次 |
| Image Scan | デプロイ先ブランチへの PR | 不要 | 週次 |
| リリースゲート（Trivy FS / OSV） | デプロイ先ブランチへの PR | 不要 | 不要 |
| npm cooldown 監査 | lockfile / `.npmrc` 変更時 | 同上 | 週次 |
| Trivy config（設定不備） | Dockerfile 変更 PR | 同上 | 不要 |
| Trivy ライセンス | Trivy FS と同一トリガー | 同上 | 週次 |
| OSV diff | 依存関係変更 PR | 不要 | 不要 |
| Opengrep（SAST） | Go / TypeScript・依存・spec 変更 PR | 同上 | 週次 |
| lockfile-lint | lockfile 変更 PR | 不要 | 不要 |
| Spectral（OpenAPI） | spec 変更 PR | `release/*` / デプロイ先ブランチ | 不要 |
| capslock | `go.mod` 変更 PR | 不要 | 不要 |
| Go fuzzing | 不要 | 不要 | 週次 |
| OWASP ZAP（DAST） | `zap-api-scan.yaml` / `.github/zap/**` 変更時 | `develop` / `staging` / `production` / `release/*` | 週次 <!-- dast:line --> |

週次実行は月曜内で 1 時間ごとにずらしています（`0 0` Trivy FS、`0 1` govulncheck、`0 2` TruffleHog、`0 3` OSV-Scanner、`0 4` Scorecard、`0 5` CodeQL、`0 6` Image Scan、`0 7` gitleaks（全履歴）、`0 8` zizmor（オンライン監査）、`0 9` npm cooldown 監査、`0 10` Opengrep、`0 11` fuzz）。同一時刻に全スキャナが並ぶのを避けるためです。

DAST は次の枠 `0 12` に入ります。スキャンの前にアプリケーションをビルドして起動する唯一のワークフローで、いちばん長く、他の前に並べても得るものが無いため、順番の末尾に置いています。 <!-- dast:line -->

週次スケジュールを持つスキャナは、ジョブが `failure` または `cancelled` で終わったときに `notify.yaml` を呼び出します。PR の失敗は作成者に見えていますが、定期実行の失敗は誰にも見えないためです。`cancelled` を含めるのは、タイムアウトやランナー障害で打ち切られたジョブが `failure` ではなくこちらになるからです。

押し出す価値があるのは失敗だけではありません。報告専用のスキャナは検出してもジョブが green で終わるため、失敗モードは検出に対して決して発火しません。それらは代わりに `notify.yaml` を検出モードで呼び出し、actor / ref / commit と検出内容そのものを添えて通知します。どちらのモードも webhook secret が未設定なら送信をスキップして run を green のままにするため、送信先を持たない fork が通知のせいで落ちることはありません。

検出通知をどのトリガーで発火させるかは、届けるべき相手が誰かで決まります。脆弱性スキャナは定期実行のみです。PR では検出内容が既に PR コメントとして依存を持ち込んだ作成者宛に出ている一方、週次の検出は「変わっていないコードに対して新たに公開された advisory」であり誰にも届かないからです。npm cooldown audit だけが例外で全トリガーで発火します。cooldown をバイパスする判断はテックリード / アーキテクトの所管であり、その人が PR の参加者とは限らないためです。

| ワークフロー | 発火条件 | トリガー |
| --- | --- | --- |
| `npm-cooldown-audit.yaml` | cooldown 違反の検出 | 全て |
| `trivy-fs.yaml` | 修正版のある CRITICAL / HIGH / MEDIUM | schedule |
| `vulnerability-check.yaml` | 到達可能な脆弱性 | schedule |
| `osv-scanner.yaml` | 昇格をブロックする検出 | schedule |

他の定期実行スキャナに検出通知は不要です。gitleaks / Trivy secret / TruffleHog / Opengrep / zizmor（high）/ image-scan のゲート / fuzzing はいずれも検出時にジョブが落ちるため、失敗モードが既に届けています。意図的に未接続のものが 3 つあります。Trivy のライセンス集計は「まだ誰も問題だと合意していないライセンス」を並べるもので（SARIF を書かないのと同じ理由）、CodeQL と Scorecard は結果を code scanning ダッシュボードへ publish するだけでワークフロー側に検出件数が出てきません。Scorecard の「スコア低下」通知には加えて前回スコアの保持が要りますが、それを持つ仕組みはここにありません。

#### 検知が重なる面

複数のツールが同じ種類の指摘を出せます。1 つの問題が二重にゲートされ二重に抑止されることを避けるため、面ごとに担当を 1 つに決めています。

| 面 | 担当 | 検知可能だがここでは使わない |
| --- | --- | --- |
| Dockerfile のセキュリティポリシー | `trivy-config.yaml` | Opengrep（`opengrep.yaml` で Dockerfile ルールを除外） |
| Dockerfile のスタイル / 正しさ | `docker-lint.yaml`（hadolint） | —（層が違い重複ではない） |
| 自前の Go ソース | `opengrep.yaml`（Opengrep）+ golangci-lint の `gosec` | — |
| OpenAPI の規約 / 命名 | `oapi-lint.yaml`（redocly） | Spectral |
| OpenAPI のセキュリティ姿勢 | `openapi-security.yaml`（Spectral） | redocly |

#### リリースゲート

依存スキャナは二段構えです。通常の PR では報告のみに留めます。既存の依存ツリーから受け継いだ脆弱性はその PR が持ち込んだものではなく、そこでブロックしても更新作業が別途進む間、無関係な作業が止まるだけだからです。ブロックの判定は `develop` / `staging` / `production` 向けの PR で行います。そこでレビュー対象になっている依存の状態が、まさに昇格されようとしている状態だからです。

| ゲート | fail する条件 |
| --- | --- |
| `trivy-release-gate.yaml` | Trivy の全 finding（修正版が出ていないものを含む） |
| `osv-release-gate.yaml` | HIGH / CRITICAL 判定の OSV finding（修正版の有無を問わない）と、判定を持たないが修正版が存在する finding |

OSV ゲートの深刻度は advisory 自身の評価を使い、無ければ osv-scanner がグループ単位で集約する CVSS スコアへフォールバックします。Go 脆弱性データベース由来の advisory はそのどちらも公開しないため HIGH 閾値では測れず、修正版が存在する場合にのみゲート対象とします。評価もできず更新もできない advisory が、昇格のたびに恒久的な赤を生むのを避けるためです。両ゲートとも意図的に `paths` フィルタを持ちません。昇格 PR はマニフェストを一切変更しないことが多く、required check はまず実行されなければブロックできないからです。

#### npm cooldown の監査

各 `.npmrc` は `min-release-age` を宣言しています。これは npm 自身の供給網隔離で、窓内に公開されたバージョンはそもそも依存解決できません。落とし穴は、これが依存**解決**時にしか効かないことです。本リポジトリの CI とイメージビルドは全て `npm ci` で、lockfile を再現するだけで解決を行わないため、cooldown を外して作られた lockfile はそのまま install でき、CI のどこにも症状が出ません。

`npm-cooldown-audit.yaml` はこの死角を埋めます。窓の長さは各 lockfile 自身の `.npmrc` から読み（ハードコードしない）、窓内のエントリを報告します。シグナルはほぼノイズを含みません。cooldown が有効なら npm は窓内バージョンの解決を拒否するので、それが lockfile に載る経路は意図的な解除以外に無いためです。

#### Go モジュールの cooldown

Go には `min-release-age` に当たるものがありません。`go get` に「新しすぎるから採るな」と言わせる手段が無い、ということです。これがツールと防御の関係を逆にします。上の npm 監査は「既にある防御が迂回された証拠」を探すので報告で足りますが、こちらは検査そのものが防御であり、報告に留めれば窓はどこにも存在しないままになります。

そこで `go-cooldown.yaml` は PR でゲートし、対象はその変更が追加 / 更新した require だけに絞ります。既に `go.mod` にあるものは grandfather するので、窓はこれから入るものに効き、引き継いだ状態で全ブランチが人質になることはありません。落とすのは **direct** だけです。indirect の版は MVS が選び、direct が要求する下限より上に固定されることがあります。それを下げるのは PR にできる操作ではないので、落としても打つ手の無い赤になります。よって indirect は報告に留めます。

窓は **7 日**で、この数字は npm からではなく本リポジトリの実績から採りました。Go モジュールには install script が無く `go mod download` は何も実行しないため、公開直後のバージョンが install 時点でマシンを奪うクラスは成立しません。窓が買うのは「悪意あるコードがビルドされ出荷されるまでの時間」です。履歴に当てると、7 日は `go.mod` を触った 47 コミットのうち 12 件を止め、14 日にしても 3 件しか増えないので両者の間に崖がありません。加えて「cooldown を満たす」と明示してバージョンを選んだ唯一のコミットが、実際には 7.4 日待っていました。

緊急の解除は [`go-cooldown-bypass.toml`](../go-cooldown-bypass.toml) が受け、エントリは必ず期限を持ちます。期限切れ・3 ヶ月より先の期限・`go.mod` の何にも当たらないエントリは検査を落とします。無効なエントリは効力も失うので、失効したバイパスが黙ってモジュールを通し続けることはありません。期限は `go.mod` が変わらなくても訪れます。定期実行があるのはそのためで、PR トリガーだけでは失効を一度も見られません。

**このワークフローは決してビルドを落としません。** これは既定値ではなく設計上の決定です。cooldown の解除はテックリード / アーキテクトの判断であり、CRITICAL への即応こそがその存在理由なので、ハードゲートは正当な行使をこそ塞いでしまいます。ブロックしない性質はワークフローの設定ではなくツール自身に持たせてあり、YAML の編集だけではゲートに変えられません。

守備範囲は正直に狭く、**方針のドリフト**（事故・規約の風化・npm 側の挙動変化）に限ります。commit 権限を持つ攻撃者は同じ変更でワークフローごと削除できるため、技術的な防止手段ではありません。そこで成立するのは検知と attribution までで、抑止は組織側の運用に委ねます。強制の担保は [`CODEOWNERS`](../CODEOWNERS) で、`**/.npmrc` / `**/package-lock.json` / `**/pnpm-lock.yaml` / `**/pnpm-workspace.yaml` と pin lockfile のレビューを所管ロールに限定します。

npm で解決するのは `mock-auth-server/` だけです。`scripts/` と `docs-viewer/` は pnpm で解決し、`minimumReleaseAge` は窓内のバージョンを記録して後から警告するのではなく、解決の時点で拒否します（`minimumReleaseAgeStrict` により、窓が黙って外れるのではなくハード失敗になります）。事後に監査する対象が残らないため、この半分には監査ツールがありません。そのぶん重みは `pnpm-workspace.yaml` 自体のレビューへ全て乗るので、CODEOWNERS に登録しています。

Pull request ではその base との差分を監査するので、検出はその変更が持ち込んだエントリだけを名指しし、対象バージョンが窓から出た後も PR コメントが記録として残ります。週次実行は全エントリを走査する二重の網です。

<!-- dast:begin -->
#### DAST

`zap-api-scan.yaml` は、ここで唯一「動いているアプリケーション」を走査するワークフローです。他のセキュリティ検査がファイルを読むのに対し、これはサーバーをビルドし、シードを入れた Postgres に対して起動し、バンドル済みの OpenAPI 定義から得たエンドポイント一覧をもとに OWASP ZAP から HTTP を投げます。

ツールの選定はこの形から決まりました。GitHub の code scanning テンプレート一覧にある DAST 6 件のうち 4 件はベンダー側でスキャンが走り、GitHub-hosted runner の内部にしか存在しない API へは到達できません。runner 内で走る残り 2 件はいずれも有償トークンが必須です。ZAP は資格情報を要さず、ジョブの内部からスキャンできる唯一の選択肢で、そもそも短命な対象を見られるのはこの性質によります。

**スキャンは認証済みで走ります。そしてここが最も壊れやすい部分です。** 未認証のスキャンは保護されたオペレーションから 401 を集めて表層で止まり、完走したように見えて完走していません。`ci` の環境プロファイルは [`docs/design/auth.md`](../../docs/design/auth.md) が述べる dev 限定スタブ認証を配線するため、資格情報はシード済みアイデンティティを指す `debug:<subject>` の bearer トークンで、ジョブは ZAP を起動する前にその資格情報が通ることを確認します。この確認を失うと検査が赤くなるのではなく、スキャンの守備範囲だけが黙って縮みます。

**報告専用であることは、決めた結果であって省略ではありません。** [`.github/zap/rules.tsv`](../zap/rules.tsv) のしきい値は、この API が現に何を返すかから導いたものです。これでマージをゲートすれば、しきい値が想定していない検出で pull request を落とすことになります。検出は `zap-dast` カテゴリで code scanning と artifact に上がり、ジョブが落ちるのは「スキャン自体が実行できなかった」ときだけです。ZAP は SARIF を出力しないため、JSON レポートをワークフロー内で SARIF へ写像しています。各検出を OpenAPI バンドルへ紐づけるのは、そのファイルこそ検出対象の面を記述したものであり、実在するファイルを指すことが code scanning 上の辿りやすさになるからです。

しきい値とスキャン対象の面は、向ける先の API に対して導き直される前提です。撤去は `scripts/setup/remove-dast-setting` が一括で行います（[セットアップ手順の Phase 17](../../docs/get-started/setup-repository.md)）。
<!-- dast:end -->

#### ランナーのハードニング

このディレクトリの全ジョブは `step-security/harden-runner` を `egress-policy: block` と、そのジョブ専用の `allowed-endpoints` とで先頭に置いています。外向き接続はすべてこの一覧と照合され、載っていない宛先は遮断されます。侵害されたアクションやツールの推移的ダウンロードは、そのジョブが本来必要としない宛先へ持ち出すことができません。ファイル改変の記録は従来どおり併走します。

このステップは**全ジョブにインラインのまま置きます**。composite action へ括り出すと重複の除去に見えますが、重複しているその部分こそ共有してはならないものです。全ジョブに足りる 1 本の allowlist は全ジョブの和集合であり、それはもはや allowlist ではありません。何処へ到達してよいかはジョブ固有の性質です。

各一覧は、以下の base に「そのジョブが実際に行うこと」を足したものです。

| 集合 | エンドポイント | 対象 |
| --- | --- | --- |
| base | harden-runner 自身の agent、GitHub の API / web / codeload、`objects` / `raw` / `release-assets.githubusercontent.com`、`*.actions.githubusercontent.com`、`*.blob.core.windows.net` | 全ジョブ（checkout、action の取得、artifact のアップロード） |
| Go | `proxy.golang.org`、`sum.golang.org`、`index.golang.org`、`storage.googleapis.com`、`dl.google.com` | `setup-go`、`go build` / `go test` / `go run` |
| mise | `mise.jdx.dev`、`mise-versions.jdx.dev`、`aquaproj.github.io`、および Go・Sigstore の集合 | 各 `mise install`。aqua backend のツール自体は GitHub リリース経由（base に含まれる）ですが、mise はその後 GitHub artifact attestation を Sigstore で検証し、`go:` backend のツールは module proxy 経由で解決するため、どちらの集合も省けません |
| Node | `nodejs.org`、`registry.npmjs.org`、`get.pnpm.io` | `npm ci` / `pnpm install` |
| Python | `astral.sh`、`pypi.org`、`files.pythonhosted.org` | `uv` で解決するツール（`sql-lint.yaml`） |
| レジストリ | Docker Hub の各ホスト、`mirror.gcr.io`、`ghcr.io`、`pkg-containers.githubusercontent.com` | イメージの build / push、service container、Trivy の DB と checks bundle |
| スキャナのデータ | `semgrep.dev`（Opengrep のルールセット）、`api.osv.dev` / `api.deps.dev`（OSV）、`vuln.go.dev`（govulncheck）、Scorecard の各データソース | それを読むスキャナ |
| ZAP | `zaproxy.org` とそのサブドメイン、およびスキャナのイメージ取得のためのレジストリ集合 | `zap-api-scan.yaml`（ZAP は起動時に add-on のマニフェストを解決します） <!-- dast:line --> |
| Sigstore | `fulcio` / `rekor` / `tuf-repo-cdn` / `oauth2.sigstore.dev` | `deploy-app.yaml`（cosign keyless 署名、attestation）**および各 `mise install`** — mise は artifact attestation の検証に Sigstore の TUF root を取得するため、`tuf-repo-cdn.sigstore.dev` へ到達できないジョブはツールを 1 つも実行する前に落ちます |

一覧は監査データの実測ではなく、各ジョブの動作から導いたものです。したがって当初の実行では取りこぼしが出る前提です。遮断された宛先は harden-runner の実行サマリに拒否接続として現れます。ジョブ自身のログからは理由が読めない失敗のとき読むべきはそこで、対処はそのジョブの一覧を広げることであり、`audit` へ戻すことではありません。

**1 つだけ失敗の出方が逆転しているジョブがあり、その旨は当該ファイルに書いてあります。** `trufflehog.yaml` は候補の資格情報を発行元サービスへ問い合わせて検証しますが、その発行元の集合には上限がありません。ここでのエンドポイント漏れはジョブを赤くせず、本物の漏洩を「未検証」に変えます。このワークフローは検証済みのみを報告するため、結果として黙って緑になります。TruffleHog の検出が消えたときは allowlist の漏れをまず疑ってください。

fork 時に読み直す価値のある前提が 2 つあります。`deploy-app.yaml` の build ジョブは `ghcr.io` と公開 Sigstore インスタンスを前提にしており、deploy ジョブは placeholder なので一覧は base のみです。実際のデプロイを結線する際は、その環境の control plane のホストを足す必要があります。`id-token` の交換も外向き通信だからです。

### デプロイ（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Deploy App|`deploy-app.yaml`|production/staging/develop への push|Docker イメージのビルド・プッシュ（cosign による image 署名 + provenance / SBOM attestation）、マイグレーション実行、デプロイ|
|Deploy Docs|`deploy-docs.yaml`|production への push（docs 変更時）|ドキュメントポータルを GitHub Pages にデプロイ|

### ドキュメント生成（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Auto-generate Docs|`auto-generate-docs.yaml`|release/* への push|`release/vX.Y.Z` のブランチ名から OpenAPI `info.version` を同期し、OpenAPI バンドル / 埋め込み spec / ドキュメント、ER 図、ポータルドキュメントを自動生成|

### アシスタント（コメント）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Claude|`claude.yaml`|プルリクエストのコメント / レビューでの `@claude`|オンデマンドでプルリクエストに対して Claude Code を実行|

## 共通 Composite Action

再利用可能な composite action は [`.github/actions/`](../actions/) に配置しています：

|アクション|目的|
|---|---|
|`setup-postgres`|Postgres サービスコンテナの待機・初期化（DB 依存ジョブで使用）|
|`upsert-pr-comment`|マーカーで既存コメントを検出して update / create する PR コメントの upsert。Commit / UpdatedAt フッターを共通付与し、結果コメント系ワークフローで使用。`status: success` は既存コメントを更新するが新規作成はしない|
|`osv-scan`|osv-scanner を実行し、各 finding をリリースゲートの深刻度ポリシーで分類する。OSV の報告用ワークフローと OSV リリースゲートで共用|

## 補足

- `.github/workflows/**` と `.github/actions/**` のコメントおよびログ文言は **英語**で書く（`echo` の出力と
  `::error::` アノテーションを含む）。このリポジトリの日本語コメント規則は Go コード・テスト名・PR・応答を
  対象とし、読み手が workflow ログと Actions エコシステムである CI 定義には及ばない。内容基準
  （[`docs/rules.md`](../../docs/rules.md) § Comment Rules）はそのまま適用される — 手順のナレーション・
  開発経緯・言い換えは書かず、非自明な Why は残す
- `auto-generate-docs.yaml` は `auto/docs-update/<base>` というブランチ名で auto-PR を作成（release base ごとに 1 ブランチを `delete-branch: true` で再利用）。再帰実行を避けるため自己ブランチでは workflow をスキップ
- デプロイ系 workflow の target ブランチ（`production` / `staging` / `develop`）はすべてブランチ保護を有効化。マージは必ず PR レビュー経由
- セキュリティスキャンのトリガーは上記「セキュリティのトリガーマトリクス」でツールごとに定義。CodeQL / Trivy で high-severity が出るとブランチ保護ルールでマージブロック
- `trivy-fs.yaml` と `osv-scanner.yaml` は**チェックを落とさない**。修正版の有無に関わらず全 finding を code scanning と PR コメントへ載せ、ブロックの判定は上記のリリースゲートに委ねる。これにより、既知の脆弱性が黙って昇格に載ることはなく、かつ通常の PR がその PR の持ち込みでない脆弱性に足止めされることもない
- `trufflehog.yaml` は**検証済み**シークレットのみを報告し、生のシークレット値をジョブログ / PR コメント / artifact のいずれにも出さない。正規表現ベースの検知は `--redact` 付きの gitleaks が担当
- **PR にコメントするジョブには secret を渡さない**。シークレットマスキングが効くのは、ランナーがジョブ出力をログ表示用に捕捉する経路だけ。ステップが `tee` でファイルへ落としたバイトはそこを通らず、`upsert-pr-comment` は本文をまさにそのファイルから読む。つまりログ上はマスク済みに見える値が、公開コメントには生のまま載る。現状どの検査ステップにも secret は渡っていないが、それを維持するのが `make actions-comment-secret-lint` で、当該アクションを使うジョブに `GITHUB_TOKEN` 以外が渡ると失敗する。検査ステップに secret が要るなら、コメントしないジョブへ分離する。なお検査が読むのは secrets の直接参照だけなので、`needs.<job>.outputs` を経由すればすり抜ける。支えているのは lint ではなく規約の側
- **`upsert-pr-comment` は「投稿者が bot」かつ「本文冒頭がマーカー」で自分のコメントを同定する**。公開リポジトリではマーカー入りのコメントを誰でも投稿でき、しかも本リポジトリのワークフローはすべて同じ bot で投稿するため、マーカーだけでも投稿者だけでもコメントは同定できない。PR 提出者が、あるワークフローのログに別のワークフローのマーカーを混ぜられれば、その別ワークフローを誤ったコメントへ誘導できてしまう。アクションが書いた本文は必ずマーカーで始まるが、混入させたマーカーはそうならない。したがって `github-token` は bot として投稿するトークン（`GITHUB_TOKEN` か GitHub App トークン）である必要がある。PAT はユーザーとして投稿するので自分のコメントを見つけられず、実行のたびに新規コメントが増える
- **攻撃者が内容を制御できるテキストを囲むフェンスは、そのテキストから長さを決める。固定長にはしない**。`upsert-pr-comment` は本文中の最長バッククォート連 + 1 をフェンス長とするが、これが働くのは `details-summary` 経路だけである。この入力が無い呼び出しでは本文を素通しする — 見出し・表・自前の `<details>` をそのままレンダリングさせたい呼び出しが複数あるためで、一律フェンスは表示を壊す。したがって素通し経路で本文の一部を自前でフェンスする呼び出しは、フェンスの責任を自分で負う。固定 3 連は、ソース行をそのまま再現する本文には閉じられる — lint が PR 提出者の書いたファイルを引用すればブロック内に 3 連が入り、以降が bot 名義の生 Markdown としてレンダリングされる。`sql-lint.yaml` は自前でフェンスを組むためログごとに長さを計算し、`capability-diff.yaml` はフェンスをアクションへ委ね step summary だけを包む。あわせて本文はアクションの `max-length` を下回るよう呼び出し側で切り詰める。この切り詰めはフェンスより**前**に適用されるため、そこで削られた本文は閉じフェンスを失う。機械的に判定できる 3 点 — `run:` からリテラルのフェンスを出さないこと、複製された `fence_for` が同一であること、素通しのワークフローが inline code span へ値を補間しないこと — は `make actions-comment-fence-lint` が見るが、「その本文が攻撃者制御か」は判定できないので、支えているのは規約の側
- **同じ規則は inline code span にも及ぶ。span は長さ 1 のフェンスでしかない**。補間した値にバッククォートが 1 つあれば span はそこで閉じ、以降は生 Markdown に戻る。刺さるのはパスの場合である。パスに使えない文字は NUL と `/` だけなので、バッククォート・`@`・リンク構文はいずれも使え、`git diff --name-only` で得たファイル名は手を加えられないままコメントへ届く（`core.quotePath` がエスケープするのは非 ASCII と制御文字であって、これらではない）。したがって素通しの呼び出しは、リポジトリ由来のパスを span にも裸の Markdown にも置かず、上と同じく一覧全体をその一覧から決めた長さのフェンスで包む。`gen-*-artifacts-check.yaml` 4 本と `sync-versions-check.yaml` のファイル一覧はこの形。本文をレンダリングさせ続ける必要がある場合、本文全体を包む解は採れない。`image-scan.yaml` の SBOM summary が見出しと強調ラベルから始まるのは、この本文がログではなくレビュアーが読むインベントリだからである。そこではテンプレートを生の Markdown のまま残し、代わりに値ごとに長さを値自身から決める。スカラーは自身の最長バッククォート連 + 1 の span へ入れ（CommonMark が剥がす空白で両端を埋めるので、値がバッククォートで始まっても終わってもよい）、一覧は上と同じくその一覧から決めた長さのフェンスで包み、digest は `^[0-9a-f]{64}$` に一致しなければ `unknown` へ落とす。SBOM 由来の文字列はパスと違って何にも有界でないため、長い連を潰し、値ごとに長さの上限を掛けてからフェンス長を決める。この上限こそが仕組み全体の土台である。フェンスから閉じ行を奪う `max-length` の切り詰めは、span からも同じように閉じデリミタを奪い、閉じられなかった span は以降の本文を生 Markdown へ戻すからである。したがって本文に寄与するものはすべて、合計が上限の内側に収まるよう有界にしてある。lint はこれらのいずれにも乗っていない本文のために、ファイル単位の除外機構を残している。運用は `.github/zizmor.yml` と同じで、エントリは追跡 issue を明記し、その指摘を直したら消す。恒久的な allowlist ではない。変数経由や `jq` の連結で組んだ span は検査から見えないので、ここでも規約が lint を上回る。なお検査は `details-summary` のキーの有無ではなく**値**を読む。この入力が空だとアクションは本文を素通しへ落とすためで、したがって `details-summary` には静的な非空文字列を渡す。空になり得る式は素通し扱いとして検査対象に含める
- zizmor の例外設定は `.github/zizmor.yml`。`ignore` はファイル単位であり、同じ audit を踏む新規ワークフローは意図どおり落ちる。恒久的な allowlist ではなく、元の指摘を直したらエントリを消す運用
- **`run:` 本文へ展開された式はコードであり、それを見ているのは zizmor だけ**。`${{ }}` の置換はシェルが構文を解釈するより前に起きるため、未クオートの `github.event.*` はコマンドを終わらせて攻撃者のコマンドを始められる。shellcheck を通すゲート群は構造的にこれを見られない（理由は [`scripts/README.md`](../../scripts/README.md) の `actions-shellcheck/` の行を参照）。zizmor の `template-injection` は代わりに展開位置そのものを判定し、式の出自が攻撃者制御可能かで重み付けする。`make actions-zizmor` が `make actions-lint` の中ではなくその隣に pre-commit フックへ載っているのはこのため。式を `env:` へ束ねてシェル側では `"$VAR"` を読む形にすれば、値はデータとして届く
- **キャッシュの安全性**。キャッシュは branch-scoped であり、run が復元できるのは自分の ref とデフォルトブランチのキャッシュだけなので、pull request の run が後続の `release/*` push が読むキャッシュを書くことはできない。通常の CI ワークフローでキャッシュを有効なままにしているのはこのため。汚染が成立する経路は 2 つある。1 つは、信頼できない PR のコードを信頼された scope で実行しつつキャッシュを保存する場合。`pull_request_target` と `workflow_run` は base ref の scope で動くため、そこで PR head を checkout するワークフローを書くと、そのキャッシュが特権 run の読む場所に残る。この 2 つを組み合わせてはならない。信頼できないコードを扱うワークフローではキャッシュを無効にする。もう 1 つは、**同じ branch scope を共有しながら権限が異なるワークフロー間**。protected branch への push で走る通常のワークフローが複数あるため、そのどれかが侵害されると `security-events: write` を持つジョブが復元・実行するツールキャッシュを残せてしまう。そのため当該権限を持つジョブはすべて `cache: false` とし、インストールが遅くなる代わりに、低権限の run が書きえた成果物を引き継がないようにしている
- `auto-generate-docs.yaml` の `Detect changes` ステップはカバレッジ HTML / SchemaSpy のタイムスタンプ揺れを除外し、無意味な PR が発火しないよう設計
- GitHub は 60 日コミットが無いとスケジュール実行のワークフローを自動的に、しかも黙って無効化する。これを回避し続けることは本テンプレートの責任範囲外であり keepalive ジョブは用意しない。動きが止まった fork では Actions タブから再有効化が必要になる前提で扱う
- fork / テンプレート由来のリポジトリは全ワークフローが `disabled_fork` 状態で作られ、この状態では何も動かない。`make enable-workflows` が列挙して一括で有効化する（冪等なので再実行して差し支えない）
- **`claude.yaml` の認可**。誰が Claude を呼べるかはワークフロー側の allowlist ではなく action 自身の書き込み権限チェックで決まる。代替案はいずれも fork で破綻する。アカウントをワークフローに直書きすると fork 先のオーナーが自分のリポジトリで締め出され、リポジトリ変数に持たせても変数は fork に引き継がれないため空に解決されて誰も呼べない。権限チェックはワークフローが動いているリポジトリに対して解決されるので、どこでも設定なしで正しく振る舞う。これを無効化する 2 つの input は意図的に未設定である。`allowed_non_write_users` はチェック自体をバイパスし、`allowed_bots` はインストールも書き込み権限も不要な App を通す。ワークフローの `if:` は `github` コンテキストしか読まず、無関係なコメントで runner を起動させないためのものであって権限を与えない。なお「誰が呼べるか」を絞っても fork PR に仕込まれたプロンプトインジェクションは防げない（呼ぶ人間は信頼できても Claude が読む diff は信頼できない）。`contents` を read のまま据え置いているのはそのためである
- `.spectral.yaml` と `.trivyignore.yaml` は `.github/zizmor.yml` と同じ方針。一括無効化はせず、各エントリに根拠となる ADR か実装を書き、抑止はパス（または JSON ポインタ）単位に閉じる。これにより同じルールを踏む新規ファイルは引き続き落ちる
- `fuzz.yaml` は PR ではなく定期実行。fuzz はランダムな corpus を探索するため、マージ可否をそれに賭けさせないための判断。クラッシュの再現入力は `testdata/fuzz/` へコミットされ、通常の回帰テストとして再生される
