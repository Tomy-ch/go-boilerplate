> **このファイルは `SKILL.md` の日本語訳です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# Docker base image の pin 更新

このスキルは `docker/*/Dockerfile` の `FROM` base image **および `docker-compose*.yaml` のサービス `image:`** の **digest 固定**を監査・更新する。**サプライチェーン cooldown ゲート**を備え、image-config の `created` が除外期間（`PIN_IMAGES_MIN_AGE_DAYS`、既定 14 日）より新しい digest は採用しない。公開直後の（侵害されている可能性のある）再ビルドを、upstream が検知・取り下げる前に取り込まないため。

`actions-pin` の姉妹スキル。あちらは GitHub Actions の `uses:` を commit SHA へ、こちらは Docker image を digest へ固定する。cooldown の思想は共通だが SSOT が異なる。

**tag はこのスキルが触るものではない。** `golang:1.26.5-alpine` / `node:24.18.0-alpine` 等のバージョンは `make sync-versions`（`mise.toml` 由来）で pin され、`go-upgrade` / `tools-upgrade` が bump する。compose のサービス image の tag（`postgres:18.3-bookworm` / `grafana/otel-lgtm:0.28.0` …）は upstream / compose サービスを設定した者が管掌する。このスキルは Dockerfile・compose いずれの image でも、tag に続く `@sha256:...` digest のみを管理する。ここで tag を編集しないこと——バージョン bump が必要なら止めて管掌スキルへ委ねる。

## このリポジトリでの pin の仕組み

着手前に必ず読むこと。以降の全手順はこの仕組みに依存する。実物の `scripts/pin-images/main.go` と `.makefiles/docker/pin.mk` を実行時に読むこと。本節は要約であり、コードが正である。

- 管理対象は 2 種類の参照で、いずれも `image:tag@sha256:<hex>` へ固定される：Dockerfile の `FROM image:tag@sha256:<hex> [AS <stage>]` と、compose サービスの `image: image:tag@sha256:<hex>`（各行の `AS <stage>` / 末尾コメントは保持される）。**バージョンの正は tag**、`@sha256` digest は不変性のロック。
- `docker/images-pin.toml` が lockfile: `"image:tag" = "sha256:<hex>"`（`apply` の SSOT・`resolve` が再生成）。**単一のフラットな map** であり、Dockerfile 由来と compose 由来の image が、どのファイル出身かに関わらず `image:tag` をキーに同居する。ヘッダは `Docker base image / docker compose image の pin 対象 digest（SSOT）`。
- `make pin-images-resolve` — `docker/*/Dockerfile` の各 `FROM` **と `docker-compose*.yaml` の各 `image:`** を集め、各 `image:tag` を `docker buildx imagetools inspect` で現在 digest へ解決し、cooldown を適用して lockfile を書き換える。環境変数 `PIN_IMAGES_MIN_AGE_DAYS`（既定 14）がゲートを制御。
- `make pin-images-apply` — lockfile を元に各 `FROM` / compose `image:` を `image:tag@<digest>` へ固定する。**fail-closed**：lockfile 非登録の image は tag のみへ正規化*しない*（digest を剥がさない）——`未登録` エラーとして報告し非ゼロ終了する。lockfile が唯一の正で、参照は lockfile に登録済みの image しか指せない。
- `make pin-images-check` — 書き換えずに、全 `FROM` / compose `image:` が digest 固定済みかつ lockfile 登録済みかを検証する（CI / hook）。network 不要（lockfile・Dockerfile・compose ファイルのみ読む）。fail-closed：`未固定`（tag のみ・digest 無し）・`未登録`（digest はあるが lockfile 未登録）・drift（digest ≠ lockfile）で非ゼロ終了し、`❌ image 参照が未固定か lockfile と不一致です …` として報告する。CI gate は `.github/workflows/pin-images-check.yaml`。

## cooldown & quarantine ルール（このスキルの核）

各 `image:tag` について（`N` = 除外日数、cutoff = `now - N 日`）：

1. **十分に古い → 採用。** 現在 digest の image-config `created` が cutoff より古ければ → lockfile に書き出し、`apply` が pin する。
2. **新しすぎる・前回 lock あり → 前回を維持。** 現在 digest が窓の内側だが、lockfile に当該 `image:tag` の既存エントリがあれば → 既存（より古く検証済み）の pin を維持し、quarantine として報告する。
3. **新しすぎる・前回 lock なし（初回）→ 拒否（非ゼロ終了）。** 現在 digest が窓の内側で前回 lock も無ければ、退行できる検証済みの古い digest が無い——新しいものを採るのも、tag のまま（未固定）残すのも、未検証の base image を出荷することになる。よって `resolve` は tag のまま残さず、当該 image と採用可能日（`created + N 日`）を報告して非ゼロ終了する。採用は、古くなってから意図的に行うか、`days=0` でブートストラップする（リスクを受容する）場合のみ。

`actions-pin` と異なる理由：git tag/release は列挙可能で不変な履歴を持つため、`actions-pin` は「1つ前の exact version」へ step-back できる。**mutable な image tag には問い合わせ可能な履歴が無い**——レジストリは「その tag が*今*指す先」しか答えない。よって step-back 先はツール自身の前回 lock（ルール 2）のみ。初回は step-back 先が無く、退行先の無い出来立て image こそが危険なケース（未検証・侵害されている可能性のある再ビルド）なので、ルール 3 は pin も tag 出荷もせず fail-closed で失敗させる。

公式イメージは頻繁に再ビルドされる（base OS の CVE パッチ）ため、現在 digest が新しいことは普通で、ルール 2（前回の古い pin を維持）は想定内でありエラーではない——退行先の無いルール 3 のみが run を失敗させる。

## いつ使うか

- base image digest pin の定期リフレッシュ。
- base image / レジストリのサプライチェーン advisory の後。
- 前回 quarantine された image が窓を越えて古くなったので pin する。

以下には使わない：

- image の**バージョン/tag**（Go/node/python ランタイム）の bump — `/go-upgrade` または `/tools-upgrade`（+ `make sync-versions`）。
- GitHub Actions `uses:` の pin — `/actions-pin`。
- Dockerfile lint の指摘 — `make docker-lint`。

## 引数

呼び出し引数を解析する（順序非依存）：

| トークン | 意味 | 既定 |
| --- | --- | --- |
| 裸の整数、または `days=N`（`--days N`） | 除外期間（日）= `PIN_IMAGES_MIN_AGE_DAYS`。 | `14` |

例: `/images-pin`（14日）・`/images-pin 30`（30日）・`/images-pin days=7`。

除外日数は非負整数。`0` は cooldown を無効化（新品 digest も採用）——ユーザーが明示的に `0` を渡したときだけ従い、サプライチェーンリスクを明示すること。

## AI 変更スコープ

`CLAUDE.md` の「Exception: Skill Execution」条項により、本スキル実行中は以下を変更してよい（本スキルは機微な `docker/` を触る——これは意図的で、pin に限定される）：

- `docker/*/Dockerfile` — `FROM` 行の `@sha256:...` digest（`make pin-images-apply` が書き込む）
- `docker-compose*.yaml` — サービス `image:` 行の `@sha256:...` digest（`make pin-images-apply` が書き込む）
- `docker/images-pin.toml` — lockfile（`make pin-images-resolve` が書き込む）

実行中も保護されるもの：

- `AGENTS.md` / `CLAUDE.md`
- 生成ファイル（`**/*.gen.go`、`*.sql.go`、`*_mock.go`、`**/openapi.gen.yaml`、`docs/` 配下の生成物）
- digest pin と無関係な全ファイル。`FROM` / compose `image:` の **tag**、`RUN`/`COPY` 手順やその他の compose キー、`scripts/pin-images`（ツール本体——既に両方のファイル種を走査する。拡張は pin 実行ではなく別個の開発タスク）を変更しないこと——tag bump が必要なら明示して止まる。

## 実行手順

### 0. 事前確認: vendor 整合 + レジストリアクセス

`make pin-images-*` は `go run ./scripts/pin-images` を実行し、`vendor/` に対してコンパイルする。`vendor/` は gitignore されるため、並行 checkout があると現ブランチの `go.mod` と不整合になり `vendor/modules.txt: ... inconsistent` で失敗しうる。その場合は一度 `go mod vendor` を実行してから進む。

`resolve` は `docker buildx imagetools inspect` を呼びレジストリへアクセスする。Docker Hub の**匿名 pull レート制限**は burst 後に `429 Too Many Requests` を返す。`resolve` が 429 で失敗したらユーザーに認証してもらう（上限が上がる）——プロンプトで `! docker login` の入力を提案——後に再実行する。`apply` / `check` は network 不要で影響なし。

### 1. 引数解析とインベントリ

引数を `<N>`（除外日数）へ解析する。次に：

- `docker/images-pin.toml` を読み、現在の `image:tag → digest` セットを把握。
- `docker/*/Dockerfile` の `FROM` **と `docker-compose*.yaml` の `image:`** を grep し、各 image のファイル位置と tag を対応づける（複数ステージ／ファイルに現れる image に注意——例: `golang:*-alpine` は複数ステージに登場。compose サービスは各々 `image:` を 1 つ持つ）。

### 2. Resolve

```sh
make pin-images-resolve PIN_IMAGES_MIN_AGE_DAYS=<N>   # 解析した除外日数
```

`resolve` は全 `image:tag` を再解決する。ルール 2 は `⚠️ ... のため既存ピンを維持` を出力し前回の古い pin を維持する——想定内でありエラーではない。ルール 3（新しすぎる・前回 lock なし）は今や**ハード失敗**：`resolve` は `❌ 退行先の無い出来立て image は採用できません ...` を出力し、該当 `image:tag` を列挙して非ゼロ終了する。ルール 2 の quarantine 対象は記録し、ルール 3 の失敗は採用可能日（`created + N 日`）とともにユーザーへ提示して止まる——無理に通さない。ユーザーは古くなるのを待って再実行するか、採用が意図的でリスクを受容するなら `days=0` で再実行する。

### 3. Apply

```sh
make pin-images-apply
```

lockfile を元に全 `FROM` / compose `image:` を固定する（古い image は `@sha256:...` を付与/更新）。fail-closed：lockfile 未登録の image は tag のみへ剥がさず、`未登録` として報告し非ゼロ終了する。これが出たら、`resolve` を通さずに `FROM` / `image:` へ image が到達した状態——resolve を再実行する（または不要な参照を除く）。

### 4. 検証

```sh
make pin-images-check     # FROM + compose image: が lockfile と一致（network 不要）
make docker-lint          # docker/*/Dockerfile に hadolint
```

各コマンドの OK / FAIL を報告する。失敗時に自動ロールバックしない——ユーザーが判断する。

### 5. 最終報告

日本語で要約する：新規 pin / digest 更新した image（digest の経過日数つき）、前回の古い pin を維持した image（ルール 2、理由つき）、ルール 3 のハード失敗があればそれ（新しすぎる・前回 lock なし——採用可能日と、ユーザーが待機を選んだか `days=0` でブートストラップしたか）、検証結果。ルール 2 で維持した image を列挙し、古くなったら再実行すべきことを示す。commit / stage / push はしない——ユーザーが `/commit`（これらは `CI:` または `Build:` 接頭辞）を手動実行する。

## 補足

- **tag は対象外。** 本スキルは digest のみ動かす。バージョン bump は `go-upgrade` / `tools-upgrade` + `make sync-versions`。
- **ルール 2 の quarantine は正常だがルール 3 は失敗。** 公式 base image は頻繁に再ビルドされるため現在 digest が新しいことは普通で、前回の古い pin があればゲートはそれを維持する（ルール 2）——想定内。だが前回 pin の*無い*新しい image（ルール 3）は検証済みの退行先が無いため、`resolve` は tag 出荷も採用もせず fail-closed で失敗させる。
- **速く動く tag は停滞しうる。** tag が `N` 日より短い周期で再ビルドされると現在 digest が常に新しく、pin が最後の古い digest から進まない。これは cooldown の受容コスト。セキュリティパッチを早く採る必要があるときだけ意図的に `N` を下げる（リスクを明示）。
- **マルチアーキ digest。** `resolve` は最上位の image-index digest を pin し（Docker がそこから各プラットフォームの manifest を解決）、経過日数判定には各プラットフォームで最も古い `created` を読む（最も保守的）。
- **`check` は network 不要。** lockfile・Dockerfile・compose ファイルのみ読むため、レジストリアクセス無しでも CI / pre-commit gate として安全。
- **強制のレベル。** ローカルゲートは lefthook の `pin-images` pre-commit フック（glob は `docker/**/Dockerfile` + `docker-compose*.yaml` + `docker/images-pin.toml`）＋ `pin-images-check.yaml` の CI workflow（paths フィルタも同様に `docker-compose*.yaml` を含む）——いずれも `pin-actions` と対称。branch ruleset の *required* status check（マージのハードブロック）まで強めるかは**意図的に template 利用者（downstream）の判断に委ねる**：boilerplate は check を提供するが強制はしない。なお paths フィルタ付き workflow を required 化すると、当該 paths を触らない PR がブロックされるため always-run 化の調整も要る——これも利用者へ委ねる理由。
- **冪等性**: 2 回目の `apply` は no-op で `pin-images-check` は通る。
- スキルは自動 push しない。

## チェックリスト

完了報告の前に確認：

- [ ] 除外日数 `<N>` を解析（既定 14）
- [ ] `vendor/` 整合を確保（必要なら `go mod vendor`）; レジストリ到達可（429 なら `docker login`）
- [ ] 現在の pin を `images-pin.toml` + `FROM` / compose `image:` grep からインベントリ化
- [ ] `make pin-images-resolve PIN_IMAGES_MIN_AGE_DAYS=<N>` 実行; ルール 2 で維持した image を理由つきで記録; ルール 3 のハード失敗（image + 採用可能日）は提示し無理に通さない
- [ ] `make pin-images-apply` 実行（`未登録` エラー無し; 全 `FROM` / compose `image:` が lockfile 登録済み）
- [ ] `make pin-images-check` + `make docker-lint` 実行・報告
- [ ] `FROM` / compose `image:` の **tag** は不変（digest のみ）; 全 `FROM` / compose `image:` が digest 固定済み（tag のみ無し）かつ lockfile 登録済み
- [ ] ルール 2 で維持した image を後日再実行用に列挙
- [ ] `SKILL.md` 更新後、`SKILL.ja.md` も同期
- [ ] commit / stage / push を行っていない
