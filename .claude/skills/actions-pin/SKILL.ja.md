> **このファイルは `SKILL.md` の日本語訳です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# GitHub Actions の pin 更新

このスキルは `.github/workflows/**` と `.github/actions/**` で SHA 固定された GitHub Actions を監査・更新する。**サプライチェーン隔離ゲート**に加え、**自動ステップバック**を備える。除外期間（`PIN_ACTIONS_MIN_AGE_DAYS`、既定 14 日）より新しいリリースは採用せず、代わりに「除外期間より前に公開済みの最新版」を固定する。公開直後の（侵害されている可能性のある）バージョンを、upstream が検知・取り下げる前に取り込まないため。

`tools-upgrade` の姉妹スキル。あちらは `mise.toml` の `[tools]` を、こちらは GitHub Actions の pin を対象とする。隔離の思想は共通だが SSOT が異なる。

## このリポジトリでの pin の仕組み

着手前に必ず読むこと。以降の全手順はこの仕組みに依存する。

- 各外部参照は `uses: owner/repo[/sub]@<40桁hex sha> # <tag>` で固定される。**バージョンの正は末尾コメントの tag** であり、`@<sha>` 側ではない。
- `.github/actions-pin.toml` が lockfile: `"owner/repo@<tag>" = "<sha>"`（`apply` の SSOT・`resolve` が再生成）。
- `make pin-actions-resolve` — 各 `uses:` のコメント tag を読み、`git ls-remote` で commit SHA へ解決（annotated tag は commit へ deref）し、隔離を適用して lockfile を再生成。env `PIN_ACTIONS_MIN_AGE_DAYS`（既定 14）でゲートを制御。moving タグの最新 SHA が除外期間内のときは**既存ピンを維持**する（moving タグに対する組み込みのステップバック）。
- **`resolve` がゲートに使う経過日数は、リリースの `published_at` と解決先 commit の日時のうち新しい方**（理由は `docs/design/security.md` の Build inputs 節）。本スキルへの影響: **`published_at` が古いことは、候補が aged であることを意味しなくなった** — リリース日は古くても head commit が新しい moving タグは、それでも隔離される。
- `make pin-actions-apply` — lockfile を元に各 `uses:` の `@<sha>` を書き換え（`# <tag>` は保持）。全対象ファイルを読み切って可否を確定させて**から**書き込むため、中断しても作業ツリーは変更されない。
- `make pin-actions-check` — 書き換えなしで lockfile と一致するか検証（CI / hook 用）。drift 以外にも fail-closed で落ちる: lockfile の解釈できない行、キーの重複、孤児エントリ、pin できない記法で書かれた `uses:` はいずれもエラー（手順 7 参照）。
- 走査対象は `.github/workflows/*.{yml,yaml}` に加え `.github/actions/**/action.{yml,yaml}` を**再帰的に**辿るため、通常の配置より 1 階層深く置かれた composite action も他と同様に固定される。
- **moving major タグ（`# v6`）は次回 `resolve` で当該メジャー内の最新へ自動追従**する。よって同一メジャーのリフレッシュは `resolve` + `apply` のみ。**メジャー更新はコメント tag の編集（`# v6` → `# v7`）が必要**。**exact 版コメント（`# 0.35.0`）は `resolve` で動かない**ため、更新はコメント編集が必要。

## 使用タイミング

以下のような場合に使用する。

- 固定済み Actions の SHA を定期的にリフレッシュ（既定のマイナーのみモード）
- Actions を新しいメジャーへ更新（`major` 引数）
- GitHub Actions のセキュリティ勧告が出たとき

以下には使用しない。

- `mise.toml` のツールバージョン → `/tools-upgrade`
- Go 本体 → `/go-upgrade`
- Go モジュール依存 → `make tidy-lib`
- ローカル複合アクション（`uses: ./...`）→ `@ref` を持たず固定対象外

## 引数

起動引数を解析する（順不同）。挙動を引数で決めるため、戦略は対話で訊かない。

| トークン | 意味 | 既定 |
| --- | --- | --- |
| `major`（または `--major`） | **メジャー**更新も対象にする。無ければ**マイナーのみ**（現行メジャー維持）。 | マイナーのみ |
| 裸の整数、または `days=N`（`--days N`） | 除外期間（日）= `PIN_ACTIONS_MIN_AGE_DAYS`。スキルのステップバック計算と `make pin-actions-resolve` の両方に使う。 | `14` |

例: `/actions-pin`（マイナー・14日）・`/actions-pin major`（マイナー+メジャー・14日）・`/actions-pin major 30`（メジャー・30日）・`/actions-pin 21`（マイナー・21日）。

除外日数は非負整数。`0` は隔離を無効化（公開直後でも採用）— ユーザーが明示的に `0` を渡した場合のみ尊重し、サプライチェーンのリスクを提示する。

## AI 変更スコープ

`CLAUDE.md` の「Exception: Skill Execution」条項により、本スキル実行中は以下の編集が許可される。

- `.github/workflows/*.{yml,yaml}` — `uses:` のコメント tag + `@<sha>`（`make pin-actions-apply` が書き込む）
- `.github/actions/*/action.{yml,yaml}` — 同上
- `.github/actions-pin.toml` — lockfile（`make pin-actions-resolve` が書き込む）

実行中も保護されるもの。

- `AGENTS.md` / `CLAUDE.md`
- 生成ファイル（`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, `docs/` 配下の生成物）
- pin 更新と無関係なファイル全般。`with:` 入力・ステップ処理・`scripts/pin-actions` は変更しない。更新に入力変更が必要なら、提示して停止する。

## ターゲット選択規則（このスキルの中核）

各アクションについて、**対象メジャー** `M` は、マイナーのみモードでは現行メジャー、`major` モードでは利用可能な最新メジャー。`M` の pin を以下で決める（`N` = 除外日数、カットオフ = `now - N日`）。

1. **moving タグが aged** — moving major タグ `vM` が存在し、その最新解決 SHA がカットオフより古い → `# vM` で固定（優先。今後の実行でも自動追従する）。
2. **一つ前の aged exact へステップバック** — それ以外（`vM` の head が除外期間内、または moving `vM` タグが無い）→ `published_at` がカットオフより古い最新の exact 版 `vM.x.y` を選び `# vM.x.y` で固定。これが「一つ前のバージョンを入れる」挙動で、隔離を守りつつ `M` に今到達できる。
3. **保留** — `M` 内にカットオフより古いリリースが一つも無い（例: `M` が新規で `vM.0.0` のみ・まだ新しい）→ そのアクションは変更せず、保留として報告。

規則の補足:

- **マイナーのみ**モードでは `M` は現行メジャーなので通常は手順1が成立し、`make pin-actions-resolve` が処理する（within-major head が新しいときは既存ピンを維持＝手順2相当）。スキルがコメント tag を編集するのは、exact 版へのステップバックを強制する必要があるとき（直近の patch リリースで head が新しい）か、exact 固定アクションの patch を上げるときのみ。
- **major** モードでは `M` は lockfile キーの無い新メジャーなので、`vM` head が新しいと `resolve` に**skip**され（→ `apply` で missing）。手順2（exact ステップバック）が新メジャーを今採用可能にし、無ければ手順3で保留。
- exact ステップバック（手順2）は moving-major 慣習から外れる。`vM` が aged になれば `# vM` へ戻せる旨をコミットに記す。`sigstore/cosign-installer`（moving `v4` タグ無し）は恒久的な手順2ケース。
- **候補の選定は `published_at`、採否は `resolve`。** リリースを `published_at` で並べて候補を選ぶ（手順2）のは引き続き正しいが、その後に掛かるゲートは `published_at` と解決先 commit 日時の新しい方を使う。リリースは古いが head commit が新しい moving `vM` は、リリース一覧上は aged に見えても `resolve` の時点で手順1に失敗し、既存ピンを維持して `⚠️ ... 既存ピンを維持` を出す。エラーではなく、手順1→手順2 のフォールバックが遅れて発火したものとして扱う。

## 実行ステップ

### 0. 事前確認: vendor 整合性 + トークン

`make pin-actions-*` は `go run ./scripts/pin-actions` を実行し `vendor/` に対してビルドする。`vendor/` は gitignore 管理のため、並行 checkout（例: `go.mod` 更新ブランチ）の状態が残ると現ブランチの `go.mod` と不整合になり `go run` が `vendor/modules.txt: ... inconsistent` で落ちる。その場合は `go mod vendor` を一度実行して再同期してから進める。また `resolve`（リリース日取得で GitHub API を呼ぶ）のレート制限回避にトークンを export する。

```sh
export GITHUB_TOKEN="$(gh auth token)"
```

### 1. 引数の解析と棚卸し

引数（上記）を `<MODE>`（minor / major）と `<N>`（除外日数）へ解析する。続いて:

- `.github/actions-pin.toml` から現行の `tag → sha` を読む。
- `.github/workflows/` と `.github/actions/` で `uses:` を grep し、各外部アクションのファイル箇所と現行コメント tag を対応付ける（複数ファイル参照に注意）。

### 2. リリース取得とターゲット pin の算出

各外部アクションについてリリース一覧と日付を取得（`gh api repos/<owner>/<repo>/releases -q '.[] | "\(.tag_name)\t\(.published_at)\t\(.prerelease)"'`・pre-release は除外）。`<MODE>` に応じて対象メジャー `M` を決め、ターゲット選択規則を適用して `# vM` 固定 / exact `# vM.x.y` 固定（ステップバック）/ 保留 のいずれかを算出する。メジャー間の **tag 形式変化**（例: `aquasecurity/trivy-action` は `0.35.0` → `v0.36.0`）に注意 — コメント tag は upstream のタグ文字列と完全一致が必要で、ずれると `resolve` が `ref not found` で落ちる。手順1候補は moving `vM` タグの実在も確認（`git ls-remote … vM`）、無ければ手順2へ。

### 3. メジャー更新の `with:` 検証

`resolve` / `apply` / `actionlint` は構文を見るが**入力のセマンティクス変更は見ない**。**メジャーが変わる**全アクションについて、リリースノート / `action.yml` を読み、当repo が使う全 `with:` ブロックと突き合わせる。実例: `peter-evans/create-pull-request` は v8 で `git-token`→`branch-token` 改名、`actions/upload-pages-artifact` v5 は dotfile 除外 + `deploy-pages@v4+` 要求。実入力が互換なら更新を維持。破壊的な入力変更があれば**そのアクションは保留し、必要な変更を報告**する（自動適用しない）。（メジャー内のマイナーリフレッシュはこの検証を省略。） <!-- skill-lint-ignore -->

### 3.5. ステップバックが使えない場合のトリアージ

ここでの隔離への通常の応答は**ステップバック**（手順 2）——既に aged な最新の exact 版を固定する。これは証拠収集を要さない。窓が既に通したものを採用するだけである。トリアージが受け持つのは、ステップバックが選択肢にならない場合である。

- **手順 3 の保留** — 対象メジャー内に cutoff より古いリリースが無く、待つか出来立てを取るかの選択しかない。退行先として検証済みの代替が存在しない。
- **修正が出来立ての head にしか存在しない勧告**。待つことが脆弱なままでいることを意味する場合。
- **`with:` レビュー（手順 3）で保留になったアクション**について、出来立て版がそれ自体安全かをユーザーが後から尋ねた場合。

これらについては、エコシステムに `github-actions`、アクション、候補 tag、そして baseline に **`.github/actions-pin.toml` に現在記録されている SHA** を渡して **`/supply-chain-triage`** を chain する。lockfile の SHA が今日 workflow が実行しているものなのだから、それが差分の正しいもう一方の端である。Actions のトリアージは 4 エコシステム中で最も強力である——artifact が実物の commit range なので、発行者・tag と commit の対応・差分・入力面のすべてが直接答えられる。

トリアージは報告のみ。コメント tag・lockfile・`uses:` 行を編集せず、`resolve` / `apply` も実行しない。バンドは手順 4 の計画へ引き継ぎ、保留か採用かの選択を証拠とともに可視化する。判断は `AskUserQuestion` に残る。

### 4. 計画の提示と確認

日本語サマリを表示する: 適用する更新（moving `# vM` / exact ステップバック `# vM.x.y`・各々の選定版と経過日数）、保留（理由: 新メジャーがまだ新しい / `with:` 破壊的 / aged 版なし）、変更なし。その上で具体的なセットを `AskUserQuestion` で確認する（独立した複数更新がある場合は `multiSelect: true`）。書き込み前にステップバックと保留の判断を可視化する。

### 5. コメント tag の編集

承認された更新ごとに、該当 `uses:` 行の末尾コメント tag を算出ターゲット（`# v7` または exact `# v4.1.0`）へ編集する。`@<sha>` はそのまま（`apply` が書き換える）。同一の `uses:` 行が複数ファイルに現れる場合はファイル単位の replace-all が適切。1ファイル内で別アクションが同じ旧コメントを共有する場合（例: 3つの `# v3`）は、意図した行だけが変わるよう完全な一意 `uses:` 行でマッチする。保留 / 変更なしのアクションは触らない。

### 6. resolve → apply

```sh
export GITHUB_TOKEN="$(gh auth token)"
make pin-actions-resolve PIN_ACTIONS_MIN_AGE_DAYS=<N>   # 解析した除外日数
make pin-actions-apply
```

`resolve` は参照中の全 tag を再解決する（現行メジャーのアクションもメジャー内の最新 aged SHA に更新）。within-major head が除外期間内のものは `⚠️ ... 既存ピンを維持` と表示されるが想定どおりで失敗ではない。`ref "vN" が見つかりません` で中断したら moving-major タグが無い → そのアクションは手順2の exact 固定にすべき。修正して再実行。vendor 不整合で中断したら手順0の `go mod vendor`。

**lockfile の diff で tag の付け替えを見張る。** moving major タグ（`# v6`）が新しい SHA へ進むのは正当である。**exact** 版コメント tag（`# v4.1.0`）はそうではない。exact tag に対して `resolve` が lockfile 記録済みと異なる SHA を返したなら、バージョン参照は同じままで下のコードが変わったということである。これは tag の付け替えであり、`tj-actions/changed-files` 侵害の形であり、pin のリフレッシュではなくセキュリティ事象である。止まり、`apply` せず、書き込みの前にトリアージする（`/supply-chain-triage`、baseline = lockfile の従前 SHA）——上流への通報を実行可能にするのは旧新の SHA である。

### 7. 検証

```sh
make pin-actions-check     # pin が lockfile と一致
make actions-lint          # actionlint で workflow 検証
```

コマンドごとに OK / FAIL を報告。失敗時に自動ロールバックしない（ユーザー判断）。

`pin-actions-check`（および同じ判定を共有する `apply`）は、通常の drift 以外に次の4条件で fail-closed になる。いずれも upstream ではなくリポジトリ側の状態の問題であり、ローカルで解消できる。

| 失敗 | 意味 | 対処 |
| --- | --- | --- |
| `lockfile に解釈できない行があります`（行番号付き） | `.github/actions-pin.toml` の行が、空行でもコメントでも `"key" = "<40桁hex>"` の代入でもない。 | `make pin-actions-resolve` を実行するか、報告された行を削除する。 |
| `lockfile にキーの重複があります` | 同じ `owner/repo@tag` が2回代入されている。lockfile のマージコンフリクトを機械的に解消した残骸として発生しやすい。 | `make pin-actions-resolve` を実行するか、重複行を削除する。 |
| `lockfile に参照されていないエントリがあります` | lockfile のキーがどの `uses:` とも対応しない。典型的には workflow を削除した際の残骸。lockfile が現用インベントリの鏡でなくなり、「差分を読めば足りる」という前提が崩れる。 | `make pin-actions-resolve`（走査結果からファイルを再生成する）を実行するか、孤児の行を削除する。 |
| `固定対象として解釈できない記法の uses: があります` | pin ツールが書き換えられない記法の `uses:` で、そのままでは固定されない。該当するのは 4 形: YAML flow mapping（`- {name: Checkout, uses: actions/checkout@v4}`）、クオートしたキー（`"uses": ...`）、値を次行へ送るブロックスカラー（`uses: >-`）、YAML alias（`uses: *anchor`）。メッセージには該当する値が出る。 | その step を素のブロック記法（`- uses: owner/repo@sha # tag` を独立行に）へ書き換える。検査の抑止で済ませない。 |

ブロックスカラーの中身は対象外なので、`run:` スクリプトが `uses: owner/repo@ref` という文字列を出力するだけでは検査に引っかからない。

### 8. 最終報告

更新したアクション（moving / exact ステップバック）、SHA リフレッシュしたアクション、保留（理由付き・トリアージのバンドとそれを決めた軸があればそれも）、手順 6 で見つかった付け替え済み exact tag、検証結果をまとめる。導入した exact 版 pin は、aged 後に見直せるよう一覧化する。commit / stage / push は行わない — ユーザーが手動で `/commit`（`CI:` プレフィックス）を実行する。

## 補足

- **隔離への既定の応答は保留ではなくステップバック。** 保留は対象メジャー内に aged 版が一つも無いときだけ。引数設計はこの挙動を前提にしている。ここでトリアージが規則ではなく例外である理由もこれである——aged な exact 版は窓が既に供給した証拠であり、検証済みの代替がコメント tag の 1 行編集で手に入るなら、出来立ての head についてさらに証拠を集めても何も買えない。
- **tag は名前であって同一性ではない。** lockfile が存在するのは tag が付け替えられ得るからであり、exact tag の SHA が動くのはそれが捕まえるために作られたケースそのものである（手順 6）。リフレッシュではなくセキュリティ事象として扱う。
- **隔離 vs 新メジャー**: ゲートは SHA の経過日数で判定し、新メジャーには既存 lockfile エントリが無いため、新メジャーの moving タグは aged になるまで `resolve` に skip される。だからこそ手順2で aged な exact 版を固定する。
- **すべてのアクションが moving major タグを持つわけではない。** `# vM` が解決できると仮定する前に必ず `vM` タグを `git ls-remote` する（`sigstore/cosign-installer` は既知の例外 → 恒久 exact 固定）。
- **`actionlint` ≠ セマンティクスの安全。** workflow 構文は検証するが、更新後アクションの入力・挙動が用途と整合するかは見ない。メジャー更新では手順3 の `with:` レビューが必須。
- **annotated tag の deref**: `resolve` は deref した commit SHA（`refs/tags/vM^{}`）を採用するため、素朴な `git ls-remote vM` の行と lockfile の SHA が異なる。
- **リリース日時は、それが指すコードの新しさではない。** 隔離は `published_at` と解決先 commit 日時の新しい方を採る。これは自動化された乗っ取りに対して時間を稼ぐものであり、日時が正直であることの証明ではない。付け替えそのものの検知は引き続き手順6の役割。論拠は `docs/design/security.md`。
- **`GITHUB_TOKEN`**: `resolve` はリリース日取得で GitHub API を叩く。トークン無しだと匿名 60 req/h 制限に当たる。`gh auth token` を export する。
- **冪等性**: 再実行すると全て固定済みと表示され、`pin-actions-check` は通る。
- このスキルは自動 push しない。

## チェックリスト

完了報告の前に以下を確認する。

- [ ] 引数を mode（minor / major）と除外日数 `<N>`（既定 14）へ解析
- [ ] `vendor/` の整合性を確保（必要なら `go mod vendor`）し `GITHUB_TOKEN` を export
- [ ] `actions-pin.toml` + `uses:` grep で現行 pin を棚卸し
- [ ] アクションごとに mode で対象メジャーを決め、ターゲット選択規則を適用（moving `# vM` aged → exact ステップバック → 保留）
- [ ] tag 形式変化（`v` 接頭辞の有無等）と moving タグの実在を考慮
- [ ] メジャーが変わる各アクションで `with:` 互換を検証し、破壊的なものは保留 + 報告
- [ ] 手順 3 の保留（および勧告起因の出来立て head）を lockfile の SHA を baseline として `/supply-chain-triage` でトリアージ。ステップバックは設計どおりトリアージ対象外
- [ ] lockfile の diff で **exact** tag の SHA が動いていないか確認（付け替え → `apply` 前に停止・トリアージ・報告）
- [ ] リリースが aged に見えた tag に `resolve` の隔離ノートが出た場合、commit 日時ゲートの発火でありエラーではないと認識している
- [ ] 計画（更新 / ステップバック / 保留と理由、トリアージのバンドがあればそれも）を提示し `AskUserQuestion` で確認
- [ ] 承認した更新のみコメント tag 編集・保留/変更なしは不変
- [ ] `make pin-actions-resolve PIN_ACTIONS_MIN_AGE_DAYS=<N>` + `make pin-actions-apply` を実行
- [ ] `make pin-actions-check` + `make actions-lint` を実行し報告
- [ ] 導入した exact 版ステップバック pin を見直し用に一覧化
- [ ] `SKILL.md` 更新後、`SKILL.ja.md` も同期
- [ ] commit / stage / push を行っていない
