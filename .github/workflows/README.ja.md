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

## ワークフロー一覧

### CI チェック（Pull Request）

|ワークフロー|ファイル|説明|
|---|---|---|
|Go Lint|`go-lint.yaml`|golangci-lint による Go コードの静的解析|
|Go Test|`go-test.yaml`|Go テスト実行とカバレッジレポート|
|Module Tidy Check|`tidy-check.yaml`|go.mod / go.sum の整合性検証|
|SQL Lint|`sql-lint.yaml`|sqlfluff による migration / DML / seed SQL の検証|
|Actions Lint|`actions-lint.yaml`|actionlint によるワークフロー定義の検証、composite action の `run:` スクリプトの shellcheck 検査、PR コメント本文への secret 混入・固定長フェンスの検査|
|Migration Check|`migration-check.yaml`|マイグレーションファイルの検証（重複、欠番、up/down ペア）|
|Sync Versions Check|`sync-versions-check.yaml`|mise.toml のバージョンが go.mod / 各 Dockerfile / README へ伝播済みか検証|
|Generated Go Artifacts Check|`gen-go-artifacts-check.yaml`|生成済み Go コードとコミット済み成果物の一致検証|
|Generated Database Artifacts Check|`gen-db-artifacts-check.yaml`|生成済み sqlc コードとコミット済み成果物の一致検証|
|Generated OpenAPI Artifacts Check|`gen-oapi-artifacts-check.yaml`|OpenAPI バンドルとドキュメントの一致検証|
|Generated Mock-Auth OpenAPI Artifacts Check|`gen-mock-auth-oapi-artifacts-check.yaml`|mock-auth-server の OpenAPI バンドル / zod スキーマ / ドキュメントとコミット済み成果物の一致検証|
|Mock-Auth Server Check|`mock-auth-server-check.yaml`|mock-auth-server の型検査、ユニット / インテグレーションテスト、golden JWKS フィクスチャのドリフト検出|
|OpenAPI Lint|`oapi-lint.yaml`|OpenAPI 定義を `redocly lint` で検証（命名 / casing / description / 未使用コンポーネント）|
|App Boot Check|`app-di-startup-check.yaml`|DB 付きでアプリケーションサーバが正常に起動するか検証|
|Job Boot Check|`job-boot-check.yaml`|ジョブのエントリポイントが起動し、未知のジョブを拒否するか検証|
|Worker Boot Check|`worker-boot-check.yaml`|worker のエントリポイントが起動（DI / DB）し、未知の worker を拒否するか検証|
|Dockerfile Lint|`docker-lint.yaml`|hadolint による Dockerfile の検証（go_tool_runner 経由）|
|Skill Definition Lint|`md-skill-lint.yaml`|`.claude/**` のスキル / エージェント定義の実態一致と、`.codex/**` との存在対応を検証|
|Pin Actions Check|`pin-actions-check.yaml`|GitHub Actions が SHA でピン留めされているか検証（サプライチェーン対策）|
|Pin Images Check|`pin-images-check.yaml`|Docker base image が lockfile 通り digest でピン留めされているか検証（サプライチェーン対策）|

### セキュリティ

|ワークフロー|ファイル|説明|
|---|---|---|
|CodeQL Scan|`code-ql.yaml`|CodeQL によるセキュリティ脆弱性分析|
|Dependency Scan|`trivy-fs.yaml`|Trivy によるライブラリ脆弱性スキャン(開発者向け)|
|Release Dependency Scan|`trivy-release-gate.yaml`|develop/staging/production 向け PR での Trivy 依存スキャン|
|Image Scan|`image-scan.yaml`|Docker イメージビルド + SBOM 生成 + Trivy スキャン|
|Vulnerability Scan|`vulnerability-check.yaml`|govulncheck による Go パッケージ脆弱性検出|
|OSV Scan|`osv-scanner.yaml`|OSV データベースによる Go モジュール / npm lockfile 横断の脆弱性スキャン|
|Release OSV Scan|`osv-release-gate.yaml`|develop/staging/production 向け PR での OSV スキャン。HIGH 以上で fail|
|Secret Scan|`secret-scan.yaml`|gitleaks によるコミット済みシークレットの検出（go_tool_runner 経由）|
|Secret Scan (TruffleHog)|`trufflehog.yaml`|TruffleHog による**検証済み**シークレット（実際に有効なクレデンシャル）の検出|
|Actions Static Analysis|`zizmor.yaml`|zizmor によるワークフロー / composite action 定義自体の静的解析|
|Dependency Review|`dependency-review.yaml`|PR が新たに持ち込む脆弱な依存をマージ前にブロック|
|OpenSSF Scorecard|`scorecard.yaml`|リポジトリのセキュリティ姿勢のスコアリングと結果の公開|
|npm Cooldown Audit|`npm-cooldown-audit.yaml`|lockfile が `.npmrc` の供給網 cooldown を満たしているかを報告（ブロックはしない）|
|Config Scan|`trivy-config.yaml`|Trivy による Dockerfile の設定不備スキャン（HIGH 以上でゲート）|
|SAST|`sast.yaml`|Opengrep（Semgrep 互換）による自前ソースの解析（taint 追跡あり）|
|Lockfile Integrity|`lockfile-integrity.yaml`|npm の `resolved` URL が正規レジストリかつ HTTPS であることの検証|
|OpenAPI Security|`openapi-security.yaml`|Spectral + OWASP API Security ルールセットによる OpenAPI 定義の検証|
|Fuzz|`fuzz.yaml`|外部入力を受けるパーサに対する Go ネイティブ fuzzing|
|Capability Diff|`capability-diff.yaml`|capslock による Go 依存グラフの capability 差分報告（report-only）|
|Notify|`notify.yaml`|定期実行の失敗、および非ブロッキングなスキャナの検出を人へ届ける `workflow_call` の再利用ワークフロー|

各スキャナは可能な限り SARIF を GitHub code scanning へ送り、結果は共通の `upsert-pr-comment` アクションで PR にコメントします。

#### セキュリティのトリガーマトリクス

各ツールは「結果が実際に変わりうる場所」で走らせています。PR はその変更自身が持ち込むリスクを surface し、protected branch への push はブランチ保護が判断材料にする code scanning のベースラインを残し、定期実行は「コードが変わらなくても結果が変わる」種別（新規公表 CVE / 新規クエリ）にだけ設けます。

| 種別 | PR | protected branch への push | 定期 |
| --- | --- | --- | --- |
| gitleaks | 全 PR | 不要 | 週次で履歴全体 |
| TruffleHog | 全 PR の差分 | 不要 | 週次で履歴全体 |
| zizmor | Actions 関連ファイル変更時 | `develop` / `staging` / `production` / `release/*` | 週次（オンライン監査） |
| Dependency Review | 依存関係変更 PR | 不要 | 不要 |
| govulncheck | Go・依存変更 PR | 同上 | 週次 |
| Trivy FS | Go・依存変更 PR | 同上 | 週次 |
| OSV-Scanner | 依存関係変更 PR | 同上 | 週次 |
| CodeQL | Go・依存変更 PR | 同上 | 週次 |
| OpenSSF Scorecard | 不要 | 既定ブランチのみ | 週次 |
| Image Scan | デプロイ先ブランチへの PR | 不要 | 週次 |
| リリースゲート（Trivy FS / OSV） | デプロイ先ブランチへの PR | 不要 | 不要 |
| npm cooldown 監査 | lockfile / `.npmrc` 変更時 | 同上 | 週次 |
| Trivy config（設定不備） | Dockerfile 変更 PR | 同上 | 不要 |
| Trivy ライセンス | Trivy FS と同一トリガー | 同上 | 週次 |
| OSV diff | 依存関係変更 PR | 不要 | 不要 |
| Opengrep（SAST） | Go・依存・spec 変更 PR | 同上 | 週次 |
| lockfile-lint | lockfile 変更 PR | 不要 | 不要 |
| Spectral（OpenAPI） | spec 変更 PR | `release/*` / デプロイ先ブランチ | 不要 |
| capslock | `go.mod` 変更 PR | 不要 | 不要 |
| Go fuzzing | 不要 | 不要 | 週次 |

週次実行は月曜内で 1 時間ごとにずらしています（`0 0` Trivy FS、`0 1` govulncheck、`0 2` TruffleHog、`0 3` OSV-Scanner、`0 4` Scorecard、`0 5` CodeQL、`0 6` Image Scan、`0 7` gitleaks（全履歴）、`0 8` zizmor（オンライン監査）、`0 9` npm cooldown 監査、`0 10` Opengrep、`0 11` fuzz）。同一時刻に全スキャナが並ぶのを避けるためです。

週次スケジュールを持つスキャナは、ジョブが `failure` または `cancelled` で終わったときに `notify.yaml` を呼び出します。PR の失敗は作成者に見えていますが、定期実行の失敗は誰にも見えないためです。`cancelled` を含めるのは、タイムアウトやランナー障害で打ち切られたジョブが `failure` ではなくこちらになるからです。

押し出す価値があるのは失敗だけではありません。報告専用のスキャナは検出してもジョブが green で終わるため、失敗モードは検出に対して決して発火しません。それらは代わりに `notify.yaml` を検出モードで呼び出し、actor / ref / commit と検出内容そのものを添えて通知します。どちらのモードも webhook secret が未設定なら送信をスキップして run を green のままにするため、送信先を持たない fork が通知のせいで落ちることはありません。

検出通知をどのトリガーで発火させるかは、届けるべき相手が誰かで決まります。脆弱性スキャナは定期実行のみです。PR では検出内容が既に PR コメントとして依存を持ち込んだ作成者宛に出ている一方、週次の検出は「変わっていないコードに対して新たに公開された advisory」であり誰にも届かないからです。npm cooldown audit だけが例外で全トリガーで発火します。cooldown をバイパスする判断はテックリード / アーキテクトの所管であり、その人が PR の参加者とは限らないためです。

| ワークフロー | 発火条件 | トリガー |
| --- | --- | --- |
| `npm-cooldown-audit.yaml` | cooldown 違反の検出 | 全て |
| `trivy-fs.yaml` | 修正版のある CRITICAL / HIGH / MEDIUM | schedule |
| `vulnerability-check.yaml` | 到達可能な脆弱性 | schedule |
| `osv-scanner.yaml` | 昇格をブロックする検出 | schedule |

他の定期実行スキャナに検出通知は不要です。gitleaks / TruffleHog / Opengrep / zizmor（high）/ image-scan のゲート / fuzzing はいずれも検出時にジョブが落ちるため、失敗モードが既に届けています。意図的に未接続のものが 3 つあります。Trivy のライセンス集計は「まだ誰も問題だと合意していないライセンス」を並べるもので（SARIF を書かないのと同じ理由）、CodeQL と Scorecard は結果を code scanning ダッシュボードへ publish するだけでワークフロー側に検出件数が出てきません。Scorecard の「スコア低下」通知には加えて前回スコアの保持が要りますが、それを持つ仕組みはここにありません。

#### 検知が重なる面

複数のツールが同じ種類の指摘を出せます。1 つの問題が二重にゲートされ二重に抑止されることを避けるため、面ごとに担当を 1 つに決めています。

| 面 | 担当 | 検知可能だがここでは使わない |
| --- | --- | --- |
| Dockerfile のセキュリティポリシー | `trivy-config.yaml` | Opengrep（`sast.yaml` で Dockerfile ルールを除外） |
| Dockerfile のスタイル / 正しさ | `docker-lint.yaml`（hadolint） | —（層が違い重複ではない） |
| 自前の Go ソース | `sast.yaml`（Opengrep）+ golangci-lint の `gosec` | — |
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

**このワークフローは決してビルドを落としません。** これは既定値ではなく設計上の決定です。cooldown の解除はテックリード / アーキテクトの判断であり、CRITICAL への即応こそがその存在理由なので、ハードゲートは正当な行使をこそ塞いでしまいます。ブロックしない性質はワークフローの設定ではなくツール自身に持たせてあり、YAML の編集だけではゲートに変えられません。

守備範囲は正直に狭く、**方針のドリフト**（事故・規約の風化・npm 側の挙動変化）に限ります。commit 権限を持つ攻撃者は同じ変更でワークフローごと削除できるため、技術的な防止手段ではありません。そこで成立するのは検知と attribution までで、抑止は組織側の運用に委ねます。強制の担保は [`CODEOWNERS`](../CODEOWNERS) で、`**/.npmrc` / `**/package-lock.json` と pin lockfile のレビューを所管ロールに限定します。

Pull request ではその base との差分を監査するので、検出はその変更が持ち込んだエントリだけを名指しし、対象バージョンが窓から出た後も PR コメントが記録として残ります。週次実行は全エントリを走査する二重の網です。

#### ランナーのハードニング

このディレクトリの全ジョブは `step-security/harden-runner` を `egress-policy: audit` で先頭に置いています。ランナーの外向き通信とファイル改変を記録することで、侵害されたアクションやツールの推移的ダウンロードが可視化されます。`audit` は記録のみで、`block` へ移行するには許可エンドポイントの確定が前提になるため、監査データが溜まるまで意図的に見送っています。

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
|`upsert-pr-comment`|マーカーで既存コメントを検出して update / create する PR コメントの upsert。Commit / UpdatedAt フッターを共通付与し、結果コメント系ワークフローで使用|
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
- **攻撃者が内容を制御できるテキストを囲むフェンスは、そのテキストから長さを決める。固定長にはしない**。`upsert-pr-comment` は本文中の最長バッククォート連 + 1 をフェンス長とするが、これが働くのは `details-summary` 経路だけである。この入力が無い呼び出しでは本文を素通しする — 見出し・表・自前の `<details>` をそのままレンダリングさせたい呼び出しが複数あるためで、一律フェンスは表示を壊す。したがって素通し経路で本文の一部を自前でフェンスする呼び出しは、フェンスの責任を自分で負う。固定 3 連は、ソース行をそのまま再現する本文には閉じられる — lint が PR 提出者の書いたファイルを引用すればブロック内に 3 連が入り、以降が bot 名義の生 Markdown としてレンダリングされる。`sql-lint.yaml` は自前でフェンスを組むためログごとに長さを計算し、`capability-diff.yaml` はフェンスをアクションへ委ね step summary だけを包む。あわせて本文はアクションの `max-length` を下回るよう呼び出し側で切り詰める。この切り詰めはフェンスより**前**に適用されるため、そこで削られた本文は閉じフェンスを失う。機械的に判定できる 2 点 — `run:` からリテラルのフェンスを出さないこと、複製された `fence_for` が同一であること — は `make actions-comment-fence-lint` が見るが、「その本文が攻撃者制御か」は判定できないので、支えているのは規約の側
- zizmor の例外設定は `.github/zizmor.yml`。`ignore` はファイル単位であり、同じ audit を踏む新規ワークフローは意図どおり落ちる。恒久的な allowlist ではなく、元の指摘を直したらエントリを消す運用
- **キャッシュの安全性**。キャッシュは branch-scoped であり、run が復元できるのは自分の ref とデフォルトブランチのキャッシュだけなので、pull request の run が後続の `release/*` push が読むキャッシュを書くことはできない。通常の CI ワークフローでキャッシュを有効なままにしているのはこのため。汚染が成立する経路は 2 つある。1 つは、信頼できない PR のコードを信頼された scope で実行しつつキャッシュを保存する場合。`pull_request_target` と `workflow_run` は base ref の scope で動くため、そこで PR head を checkout するワークフローを書くと、そのキャッシュが特権 run の読む場所に残る。この 2 つを組み合わせてはならない。信頼できないコードを扱うワークフローではキャッシュを無効にする。もう 1 つは、**同じ branch scope を共有しながら権限が異なるワークフロー間**。protected branch への push で走る通常のワークフローが複数あるため、そのどれかが侵害されると `security-events: write` を持つジョブが復元・実行するツールキャッシュを残せてしまう。そのため当該権限を持つジョブはすべて `cache: false` とし、インストールが遅くなる代わりに、低権限の run が書きえた成果物を引き継がないようにしている
- `auto-generate-docs.yaml` の `Detect changes` ステップはカバレッジ HTML / SchemaSpy のタイムスタンプ揺れを除外し、無意味な PR が発火しないよう設計
- GitHub は 60 日コミットが無いとスケジュール実行のワークフローを自動的に、しかも黙って無効化する。これを回避し続けることは本テンプレートの責任範囲外であり keepalive ジョブは用意しない。動きが止まった fork では Actions タブから再有効化が必要になる前提で扱う
- fork / テンプレート由来のリポジトリは全ワークフローが `disabled_fork` 状態で作られ、この状態では何も動かない。`make enable-workflows` が列挙して一括で有効化する（冪等なので再実行して差し支えない）
- **`claude.yaml` の認可**。誰が Claude を呼べるかはワークフロー側の allowlist ではなく action 自身の書き込み権限チェックで決まる。代替案はいずれも fork で破綻する。アカウントをワークフローに直書きすると fork 先のオーナーが自分のリポジトリで締め出され、リポジトリ変数に持たせても変数は fork に引き継がれないため空に解決されて誰も呼べない。権限チェックはワークフローが動いているリポジトリに対して解決されるので、どこでも設定なしで正しく振る舞う。これを無効化する 2 つの input は意図的に未設定である。`allowed_non_write_users` はチェック自体をバイパスし、`allowed_bots` はインストールも書き込み権限も不要な App を通す。ワークフローの `if:` は `github` コンテキストしか読まず、無関係なコメントで runner を起動させないためのものであって権限を与えない。なお「誰が呼べるか」を絞っても fork PR に仕込まれたプロンプトインジェクションは防げない（呼ぶ人間は信頼できても Claude が読む diff は信頼できない）。`contents` を read のまま据え置いているのはそのためである
- `.spectral.yaml` と `.trivyignore.yaml` は `.github/zizmor.yml` と同じ方針。一括無効化はせず、各エントリに根拠となる ADR か実装を書き、抑止はパス（または JSON ポインタ）単位に閉じる。これにより同じルールを踏む新規ファイルは引き続き落ちる
- `fuzz.yaml` は PR ではなく定期実行。fuzz はランダムな corpus を探索するため、マージ可否をそれに賭けさせないための判断。クラッシュの再現入力は `testdata/fuzz/` へコミットされ、通常の回帰テストとして再生される
