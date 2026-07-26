> **このファイルは `SKILL.md` の日本語訳です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# Dependency Vulnerability Upgrade

このスキルは **セキュリティ勧告リスト** を受け取り、名指しされた脆弱な依存だけを修正版へパッチする。対象は本リポジトリが使う 2 つのエコシステムにまたがる:

- **npm** — 各 `package-lock.json`（現状 `docker/tools/` と `docker/mock-auth-server/`）に記録された依存。`package.json` の `overrides` エントリで固定する必要がある **推移的（transitive）** 依存を含む。
- **Go** — `go.mod` / `go.sum` のモジュール。**間接（indirect）** 依存を含む。

このスキルは意図的に **対象を絞る**: 勧告で名指しされた package のみを変更し、「すべてを最新へ」の一括更新はしない。そうすることでセキュリティパッチをレビュー可能に保ち、無関係な変更と切り離す。一括更新には `/tools-upgrade`（mise ツール）や `make tidy-lib`（Go モジュール）を使うこと。

日本語参照訳は同じディレクトリの `SKILL.ja.md`（スキルとしては読み込まれない・人間参照用）。

## When to Use

次の場合に使う:

- ユーザーが脆弱性レポート（`npm audit` / Trivy / Dependabot alert / `govulncheck`、あるいは手書きの一覧）を貼り、指摘 package のパッチを望むとき。
- CVE / GHSA 勧告が `package-lock.json` や `go.mod` に存在する package を名指しし、最小の修正版バンプ + 検証を行いたいとき。
- ツール（`redocly` / `orval` 等）が引き込む推移的 npm 依存を、`overrides` でパッチ版へ強制する必要があるとき。

次には使わない:

- 「pin 済みツールをすべて最新へ」のルーチン監査 — それは `mise.toml` `[tools]` 向けの `/tools-upgrade`。
- Go 言語バージョンの引き上げ — それは `/go-upgrade`（downstream 同期が異なる）。
- 特定勧告と無関係な `go.mod` 依存の一般的リフレッシュ — `make tidy-lib` を使う。
- 実体が mise 管理ツールである npm パッケージ — それは `/tools-upgrade` の管轄。

## First Step: 勧告のパースと caution しきい値の解決

勧告リストをパースし、supply-chain caution しきい値 `<MIN_AGE_DAYS>` を解決する — **スキルとツールチェーンを一致させるため、リポジトリ自身の npm cooldown を優先** し、ディスク上に権威ある値が何も無いときだけユーザーに尋ねる。

手順:

1. スキル引数または直近のユーザーメッセージから勧告リストをパースする。各エントリから **package 名** / **現行 version**（あれば）/ **修正版候補**（major 系列ごとに複数あり得る）/ **CVE・GHSA id** / **深刻度** を取り出す。リストは自由形式でよい — よくある形（`- [HIGH] lodash 4.17.23 → 4.18.0 (CVE-...)`、`npm audit` ブロック、Trivy 行）を許容する。エコシステムや所在が曖昧なエントリは Step 1 で解決する（ここで推測しない）。
2. リポジトリの npm cooldown を検出する: lockfile ディレクトリ配下の各 `.npmrc`（例 `docker/tools/.npmrc`）を読み `min-release-age=N` 行を探す。これは npm 11+ ネイティブの supply-chain 隔離で、**依存解決**時（`npm install` / `npm install --package-lock-only`）に `before = now − N 日` のハード cutoff を適用する — cutoff より新しい版は **そもそも install できず**、単なるフラグでは済まない。見つかれば `N` をその lockfile の `<MIN_AGE_DAYS>` として採用し、スキルの caution しきい値をツールチェーンが実際に敷く壁に合わせる。
3. ある変更に `.npmrc` cooldown が効かない場合（Go モジュール、または `min-release-age` の無い lockfile）は、引数で値が渡されない限り `7` を既定 `<MIN_AGE_DAYS>` とする。しきい値の確認で `AskUserQuestion` を呼ぶのは真に曖昧なとき（`.npmrc` の値が競合、またはユーザーが override を求めた）だけ。単一の repo `.npmrc` 値や `7` 既定は質問不要 — 使った値を明記して進める。

しきい値はエコシステムにより 2 つの異なる役割を持つ:

- **`.npmrc` `min-release-age` 下の npm**: **ハードブロック**。cooldown 内の修正版は `npm install` を `ETARGET ... No matching version found ... with a date before <cutoff>` で失敗させる。リポジトリ自身のポリシーと戦わず、そうした版は **deferred（Step 4）** として扱い、適用しない。
- **それ以外（Go、cooldown 無しの npm）**: **caution フラグでありハードブロックではない** — 既知脆弱性を直すのが目的なので、新しすぎる修正版は握り潰さず提示・確認する。

`<MIN_AGE_DAYS>` が解決するまでレジストリ取得やファイル編集を行わない。

## AI Modification Scope

`AGENTS.md` は通常 `docker/**` とリポジトリ直下ファイルを AI 編集の対象外とする。**このスキルの起動がそれを緩める明示指示** であり、依存パッチが触れる特定ファイルに限り、この実行の間だけ緩和される。これは `AGENTS.md` の「Skills must not be a loophole」条項に沿った、文書化済みの非抜け穴的例外である — 下記が全変更面であり、どれが変わったかはユーザーへ報告する。

このスキル実行中に変更可:

- `**/package.json` — 承認済み package の `dependencies` version もしくは `overrides` エントリの追加/調整に限る（特に `docker/tools/package.json`、`docker/mock-auth-server/package.json`）。
- `**/package-lock.json` — 承認済み変更に対する当該 package ディレクトリでの `npm install` の決定的出力。
- `go.mod` / `go.sum` — 承認済み Go モジュールに対する `go get <module>@<version>` + `go mod tidy` の出力。
- `vendor/**` — `go mod vendor` の機械的出力として **のみ**、かつ **リポジトリが vendoring するとき（`vendor/modules.txt` が存在）のみ**。`go.mod` バンプは `vendor/modules.txt` を不整合のまま残し、ビルドが `inconsistent vendoring` で落ちる。よって再 vendoring は好みの編集ではなく必須の downstream 手順。vendored ファイルは手編集しない。
- これら依存が駆動する再生成物 — Step 6 の drift チェックで動いた場合 **のみ**（例: `make gen-mock-auth-oapi` 経由の `docker/mock-auth-server/openapi/openapi.gen.yaml` / `src/generated/**`）。リポジトリの `make` ターゲットで再生成する。生成物を手編集しない。

このスキル実行中でも保護（触れない）:

- `AGENTS.md` / `CLAUDE.md`
- `node_modules/**`（追跡対象外のビルド生成物）、および上記 `go mod vendor` 経由を **除く** `vendor/**`
- 手編集される生成ファイル（`**/*.gen.go`、`*.sql.go`、`*_mock.go`、`**/openapi.gen.yaml`、`docs/` 配下の生成物） — `make` 経由の再生成は可、手編集は不可。
- 勧告リストに名指しされていない package。このスキルは対象限定であり、隣の依存を便乗バンプしてはならない。

## Execution Steps

### 1. 各 package の所在特定

勧告エントリごとに、package が実際にどこに存在するかを見つけて分類する。推測せず、ツリーに対して検証する。

```sh
# npm: どの lockfile に package が含まれるか
find . -name package-lock.json -not -path '*/node_modules/*'
# 次に、候補 lockfile ごとに:
grep -n "\"node_modules/<pkg>\"" <lockfile>          # 存在 + インストール済み version
# 直接 vs 推移的: その package.json 自身の dependencies/devDependencies にあるか
grep -n "\"<pkg>\"" <dir>/package.json

# Go: モジュールが go.mod にあるか、直接か間接か
grep -n '<module-path>' go.mod
```

エントリごとに記録:

| 項目 | 方法 |
| --- | --- |
| ecosystem | `npm` か `go` |
| location | `package-lock.json` のディレクトリ、または `go.mod` |
| installed version | lockfile / `go.mod` から |
| direct / transitive | package.json の `dependencies`/`devDependencies` にある（npm）／`// indirect` が無い（go）→ direct、そうでなければ transitive/indirect |

いずれの lockfile / `go.mod` にも見つからない package は **not-present**（既に除去済み、または本リポジトリに無い lockfile）として報告しスキップする — 所在を捏造しない。

### 2. 修正版の選定

存在する package ごとに対象 version を選ぶ:

- **既定: 現在インストール済みの major 系列に留まる最小の修正版。** 複数候補の勧告（`brace-expansion 1.1.15 → 5.0.7 / 1.1.16 / 2.1.2`）からは、インストール済み major に一致するもの（`1.1.15` → `1.1.16`）を選ぶ。breaking リスクを最小化する。
- **major 跨ぎが必要**（修正が上位 major にしか無い、またはインストール系列に patch 版が無い — 例 `@hono/node-server 1.19.14 → 2.0.5`）: **breaking の可能性** として明示的にフラグする。per-package 確認（Step 4）とより入念な検証（Step 6）を要する。
- **downgrade ガード**: インストール済みより厳密に低い version は決して選ばない。唯一の「修正」がより低くパースされる場合は適用せず `needs-manual` として提示する。
- **勧告の番号ではなく、実際に解決される version をゲートする。** `package.json` の `^`/`~` レンジは一致する最新 patch へ浮くため、勧告の修正版よりずっと新しく `too-new` になり得る。lockfile が landing する version の日付を計算し（Step 5 で再確認）、レンジが too-new/未検証版へ解決してしまう（かつ `.npmrc` cooldown で抑えられないディレクトリ）場合は、caret を浮かせず **承認済みの exact 版を pin** する。

caution ゲート（Step 3）用に、選定各版の **公開日** を取得:

```sh
# npm
curl -fsSL https://registry.npmjs.org/<pkg> | jq -r '.time["<version>"]'
# Go（モジュール公開時刻）
curl -fsSL "https://proxy.golang.org/<module>/@v/<version>.info" | jq -r '.Time'
```

### 3. supply-chain caution ゲートの適用

選定各版について `now - publish_date` を計算し disposition を定める:

| フラグ | 条件 | 効果 |
| --- | --- | --- |
| **clear** | `>= MIN_AGE_DAYS` | eligible — 既定で適用（Step 4） |
| **too-new** | `< MIN_AGE_DAYS`、`.npmrc` cooldown 無しの npm、または Go | eligible だがフラグ付き。⚠️ 提示・既定では未適用 — ユーザーが opt-in する必要 |
| **blocked** | `.npmrc` `min-release-age` 下で `< MIN_AGE_DAYS` | **install 不可** — リポジトリ自身の npm cooldown がハード拒否。適用せず **deferred** とし、解除時期（`publish_date + N 日`）を報告 |

npm / Go proxy への悪意ある publish は通常、数時間〜数日で検知・撤回されるため caution を設ける。セキュリティ修正は急ぐので `too-new` はユーザーが override できる警告だが、`blocked` はリポジトリ自身のポリシーを npm 自体が強制するものであり、スキルは cooldown を無効化せず尊重する。境界は現実的である点に注意: N 日窓の内側に数分でも入って公開された版は、窓がその正確な publish 時刻を跨ぐまで `blocked`。

### 3.5. ゲートが捕捉したものをトリアージする

disposition が `too-new` または `blocked` になった全エントリについて、Step 4 が判断を提示する前に **`/supply-chain-triage`** を chain する（エントリごとに 1 回）。

窓は四つの問い——発行者は変わったか、artifact は source と一致するか、実際に何が変わったか、新しい依存が増えたか——の代理指標であり、待つ代わりに直接答えることができる（`docs/design/security.md` → 「Dependencies」）。この Step がその実行場所である。`too-new` の opt-in はまさに、ユーザーの日数に対する許容度ではなく証拠に基づくべき判断であり、`blocked` エントリについては、待つことが単に不便なのか実際に防御的なのかを知る必要がある。

エコシステム、パッケージ、候補バージョン、**lockfile が現在保持している baseline**（差分のもう一方の端）、`<MIN_AGE_DAYS>`、disposition、移動を強いている CVE を渡す。トリアージは報告のみ——tarball / module zip を実行せずに読み、0–12 のスコア・バンド・根拠を返す。何も変更せず、何も採用しない。判断は Step 4 に残る。

フラグ付きが何も無い（全て `clear`）ときは飛ばす。この run でユーザーが既に見送ったエントリも飛ばす。各エントリのバンドは Step 4 のサマリと選択肢の説明へ引き継ぎ、証拠を添えたうえで選択させる。

### 4. サマリ表示・clear は自動適用・フラグ付きのみ確認

日本語サマリを disposition ごとに出す。例:

```text
依存脆弱性パッチ監査（min_age_days = 7, docker/tools/.npmrc 由来）

✅ 適用（同一 major の最小修正版 / caution 通過 → 確認なしで適用）:
  - lodash 4.17.23 → 4.18.0  [docker/tools, 推移的]  (CVE-2026-4800, CVE-2026-2950 / HIGH)
  - js-yaml 4.2.0 → 4.3.0     [docker/tools, 直接]    (CVE-2026-59869, CVE-2026-53550 / HIGH)

⚠️ major 跨ぎ（breaking の可能性 / 別途確認）:
  - @hono/node-server 1.19.14 → 2.0.5  [docker/mock-auth-server, 直接]  (GHSA-frvp-7c67-39w9 / MEDIUM)

⚠️ too-new（公開が min_age 未満 / 別途 opt-in）:
  - fast-uri 3.1.3 → 3.1.4  [docker/tools]  (公開 3 日 / CVE-2026-16221 / HIGH)
      トリアージ: 1/12 LOW（発行者同一・provenance 一致・差分は URL parser のみ・新規依存なし）

⛔ deferred（repo の .npmrc min-release-age に阻まれ install 不可）:
  - brace-expansion 1.1.16  [docker/tools, spectral-core 内]  (公開が cooldown 内 / 2026-07-22 頃に解除)
      トリアージ: 2/12 LOW（ただし .npmrc の cooldown により install 自体が不可）

❓ 未検出 / 要手動:
  - （lockfile に見つからない等）
```

確認ポリシーは意図的に非対称にする — clear なパッチはスキルを回す目的そのものなので、ユーザーにクリックさせない:

- **clear かつ非 major → 確認なしで適用。** これは既定パッチであり、1 件ずつ確認するのは、勧告に対しスキルを起動した時点でユーザーが既に暗黙承認している摩擦にすぎない。
- **major 跨ぎ（インストール済みより上位 major）→ 常に別途報告・確認**。version 自体が `clear` でも同様。major はそれを import するコードを壊し得るので、ユーザーが承知のうえ判断する。Step 6 でより入念に検証。
- **too-new（cooldown 無しの caution）→ 報告・確認（opt-in）**、既定は未適用。
- **blocked / deferred → 決して適用しない**。報告と解除時期だけ示す。

`AskUserQuestion` を呼ぶのは、判断すべき **major 跨ぎ** および/または **too-new** エントリがあるときだけ（それらだけを並べた 1 回の `multiSelect: true`、既定は全て未選択）。各 `too-new` の選択肢には Step 3.5 のトリアージ・バンド（`1/12 LOW` / `7/12 HIGH` / `INSUFFICIENT-EVIDENCE`）を説明として付ける。opt-in するか待つかの理由はそのバンドなのだから、上のサマリだけでなくクリックする場所に置くべきである。eligible が全て `clear` かつ非 major なら、質問せず即適用する。eligible が無ければ書き込みなしで Step 7 へ。

### 5. 更新の適用

承認 package を location ごとにまとめて適用する。`package-lock.json` は決して手編集せず、`npm` に再生成させる。

ここで追跡されるのは `package.json` + `package-lock.json` のみ（`node_modules/` は git-ignore、toolbox イメージで再構築）なので、`--package-lock-only` で **lockfile のみ** 更新する — レジストリから解決し、ツリー全体をダウンロードせず lockfile を書き直す。`package.json` の編集は自分で行い、lockfile は `npm` に再生成させる。`package-lock.json` は手編集しない。

**npm — 直接依存**（その package.json の `dependencies`/`devDependencies` にある）: 宣言レンジをバンプして再生成。

```sh
# <dir>/package.json を編集: "<pkg>": "<new-range>"
cd <dir>
npm install --package-lock-only
```

再生成後、**lockfile が実際に pin した version を読み戻す**（`node -e '...node_modules/<pkg>...'`）。`^`/`~` レンジは一致する最新 patch へ解決するため、`^2.0.5` が昨日公開の `2.0.11` へ landing することがある — その解決版を Step 3 に対して再ゲートする。`too-new`（かつ `.npmrc` cooldown で抑えられないディレクトリ）なら、exact 承認版（`"<pkg>": "2.0.5"`）で pin して再生成し、最新版ではなく検証済み版へ landing させる。

**npm — 推移的依存**（他 package が引き込む・直接依存でない）: `overrides` エントリでパッチ版へ強制する。ここでは **specifier の書き方** と **スコープ** の 2 点が要点。

**specifier — exact pin ではなく同一 major の range フロアを優先する。** `overrides` エントリは権威的かつ *sticky* で、npm は書いたとおりを強制し、勝手に上へ流れることはない。したがって exact pin（`"<pkg>": "1.2.3"`）はその nested コピーを **凍結** する — 親が後に修正版を native に取り込んでも override は古い枝を保持し、さらに `1.2.3` 自体に後日 CVE が出れば override は今や脆弱な版を強制し続ける（pin 自体が脆弱性となり、しかも見落としやすい）。修正を *最低ライン* として強制しつつ、依存が親に追従して上へ動けるよう、**インストール済み major 内に留まるフロア** を書く:

```jsonc
// <dir>/package.json — スコープ付きフロア: 少なくともパッチ版・major 内でなお上へ流れる
"overrides": {
  "<vulnerable-parent>": { "<pkg>": ">=<fixed> <<next-major>" }   // 例 ">=1.2.3 <2"
}
// exact pin（"<pkg>": "<fixed>"）は、range 内のより新しい版が既知の壊れで留めざるを得ないときだけ
// 素の（スコープ無し）"<pkg>": "..." は、全コピーを動かす必要があるときだけ
```

**スコープ — 問題の親配下に pin する**（global ではない）。そうすれば同一 package の **既に修正済み top-level コピーを downgrade せずに** 脆弱な nested コピーを直せる。素の global override は全コピーを指定 specifier へ強制する。

その後 `cd <dir> && npm install --package-lock-only` で再生成。既存の `overrides` ブロックがあれば追記し、兄弟を潰さない。同一 package の承認済み変更は 1 回の編集 + 1 回の `npm install --package-lock-only` にまとめる。range フロアは npm が受理できる range 内の最新版に解決されるので、`.npmrc` `min-release-age` 下では修正版以上で最新の *枯れた* 版に着地する — 依存が動いても手動の再 pin は不要。フロアでも npm が `ETARGET ... date before <cutoff>` で拒否したら、修正版そのものがまだ cooldown 内（Step 3 `blocked`）— そのエントリを外し、当該 package は deferred のまま残して他を進める。

**Go モジュール**（直接 / 間接）:

```sh
go get <module>@<version>
go mod tidy
go mod vendor        # リポジトリが vendoring するとき（vendor/modules.txt が存在）のみ
```

承認済み Go モジュールは 1 回の `go get`（複数 `module@version` 引数）+ 1 回の `go mod tidy` にまとめ、`go.sum` を一度で収束させる。`vendor/modules.txt` が存在するなら後で `go mod vendor` を走らせる — これが無いと `go.mod` と `vendor/modules.txt` が食い違い、ビルドが `inconsistent vendoring` で落ちる。

### 6. 検証

実際に変わったものに合わせてチェックを走らせる — 依存パッチが一次ソースに触れることは稀なので、常にフルスイートを回すのではなく、動いたエコシステムに検証をスコープする。各々を OK / FAIL で報告し、失敗しても自動ロールバックしない — ユーザーが判断する。

Go 変更は最低限ビルド + vuln スキャンが clean であること（`go build ./...` + `govulncheck ./...`）。Go 変更がフルスイートに値するほど広ければ `make lint` / `make test` を回す。**major な npm バンプ** はより入念に検証する — その package 自身の `typecheck` + テスト（例 `docker/mock-auth-server/` で `npm run typecheck` + `npm test`）を回す。major は呼び出す API を変え得るため。

npm 変更 — 勧告が実際に解消されたか / lockfile が clean かを確認:

```sh
cd <dir> && npm audit            # パッチ対象 CVE が出なくなること
```

`npm audit` は **真の fix floor** の source of truth であり、ユーザーのリストと食い違い得る。黙って解決せず提示すべき 2 ケース:

- **勧告が挙げた版より高い fix floor。** その package が *別の* 勧告を抱え、初修正版がユーザーの貼った版より高いことがある（例: リストは「2.0.5 で修正」だが、`npm audit` は `2.0.0 - 2.0.9` に影響し `2.0.10` でのみ修正される別の moderate 勧告をなお指摘）。その高い版が `too-new` なら黙って飛びつかない — 衝突と、そもそも脆弱経路が到達可能か（例: WS ハンドラ未登録のサーバに対する WebSocket 限定の DoS は非該当の可能性が高い）を提示し、too-new な完全修正への opt-in か部分修正の受容かをユーザーに委ねる。
- **deferred/スキップした package に残る勧告** — 想定内。失敗ではなく、理由（cooldown で deferred / ユーザーがスキップ）付きで未解消と報告する。

Go 変更:

```sh
govulncheck ./...                # 利用可能なら。GHSA が解消すること
```

生成物 drift — これら npm 依存はコード生成器を駆動するため、バンプで生成出力が動き得る。変更した lockfile が生成器を養う package のものなら、その生成器を走らせて drift を確認する（CI と同じチェック）:

- `docker/mock-auth-server/**` 依存 → `make gen-mock-auth-oapi`（および `make gen-mock-auth-oapi-docs`）→ 生成パスを `git status`。再生成物をパッチの一部としてコミットする。
- `docker/tools/**` 依存（toolbox イメージ: redocly / orval / sqlc 周辺ツール）→ 依存が変わったツールがそれらであるときに限り、そのイメージが生成する成果物（`make gen-*-oapi`、`make gen-query` 等）を再生成し drift を確認。

再生成で変更が出たら含める — 生成出力を黙って変えるセキュリティバンプが、CI の drift チェックを落とす状態でツリーを残してはならない。

### 7. 最終報告

日本語で要約:

- パッチした package を ecosystem / location ごとにまとめ、version 差分と CVE id 付きで。
- 追加した `overrides` は、親が修正版を native に取り込んだら **回収すべき暫定 pin** として明示する（使った specifier — フロアか例外的 exact pin か、およびその理由）。
- 適用した `too-new` / `major-bump` エントリ（およびその確認）、または意図的にスキップしたもの。
- cooldown が捕捉した各エントリについて、そのトリアージ・バンドとそれを決めた軸。採用か待機かの判断が証拠に基づいたことを記録に残すため。答えられなかった軸があればそれも挙げる。
- ユーザーが別手段で対処すべき `not-present` / `needs-manual` エントリ。
- 検証結果（`make lint` / `make test` / `npm audit` / `govulncheck` / drift チェック）。
- 再生成物（あれば）。

commit / stage / push はしない。ユーザーがツリーをレビューし `/commit` を手動実行する。コミットを求められた場合、セキュリティパッチは `docker/**` + `go.mod` + 再生成物にまたがることが多い旨を伝え、CVE を説明する明確な `Build:` / `Fix:` コミットにまとめる。

## Notes

- **一括ではなく対象限定。** このスキルの本質は勧告で名指しされた package のみに触れること。実行中に無関係な古い依存に気づいても、言及はしてもここでバンプしない。
- **聞きすぎない。** `clear` かつ非 major のパッチは、まさにユーザーがスキルを起動した目的 — 確認クリックなしで適用する。`AskUserQuestion` は本当に重い判断のためだけに取っておく: **major 跨ぎ**（breaking リスク）と **too-new** の opt-in。それらは別途・明確に報告する。
- **リポジトリの npm cooldown を尊重し、無効化しない。** `.npmrc` `min-release-age=N` は意図的な supply-chain 制御。修正版をブロックしたら（`ETARGET ... date before <cutoff>`）、その版は `blocked`/deferred — 解除時期（`publish_date + N 日`）を報告し、`min-release-age` を下げたり `--before` を足したり迂回したりしない。LOW のトリアージ・バンドもこれを変えない。トリアージが供給するのは証拠であって許可ではなく、cooldown は証拠が何を言おうと npm が依存解決時に強制する。勧告の唯一の cooldown 内修正版が同日公開の新しい patch であることが多い。窓が明けるのを待つか、より古く既に枯れた修正版を選ぶ — *ただしユーザーがその版を承認した場合のみ*。
- **Vendoring。** `vendor/` があるなら、`go.mod` の変更は `go mod vendor` が `vendor/modules.txt` を再同期するまで完了しない。さもなくばビルドが `inconsistent vendoring` で落ちる。
- **推移的 vs 直接。** 修正を含む互換な直接依存版が既にあるなら直接バンプが望ましい。親がまだ release していない純粋な推移的依存にはスコープ付き `overrides` を使う。スコープ付き override（`"parent": {"pkg": ">=<fixed> <<next-major>"}`）は、既に修正済みの top-level コピーを downgrade せずに脆弱な nested コピーを直す。各 package がどちらを使ったか報告に明記する。
- **override は暫定的な負債 — フロアで書き、後で回収する。** `overrides` エントリは手動の sticky な pin で、npm は失効も通知もしないため、推移的依存への静かなキャップとして腐っていく。健全に保つ 2 つの規則: (1) exact version ではなく **同一 major のフロア**（`">=<fixed> <<next-major>"`）で書き、依存を凍結せずに修正を *最低ライン* として強制する — exact pin は range 内のより新しい版が既知の壊れで留めざるを得ないときだけに取っておく。(2) すべての override を **暫定** として扱う — **親** が修正版を native に引き込む release を出したら回収する: 親をバンプし、不要になった override を削除し、`npm install --package-lock-only`、そして `npm audit` を再実行して pin 無しでも修正が保たれることを確認する。腐った exact override は、pin した版自体が指摘されると脆弱性を再導入すらしうる。
- **複数 CVE の package。** 1 つの package が複数勧告行に出ることがある（例 `lodash` が 2 つの CVE 下）— すべてを満たす 1 回のバンプに dedup し、解消する全 CVE を挙げる。
- **lockfile のみ。** `node_modules/` は追跡されない。`npm install --package-lock-only` はフルインストールなしで `package.json` + `package-lock.json` を更新する。バンプは npm に lockfile 内の兄弟コピーを再配置させ得る — diff は確認するが、同一 package の `4.17.x → 4.18.x` 的な dedup churn の増加は想定内で無害。
- **冪等性。** 適用成功後の再実行は、package が既に修正版であること（npm audit / govulncheck clean）を示し、書き込みをしない。
- スキルは決して自動 push しない。ユーザーがレビューし、手動でコミット・push する。

## Checklist

完了報告の前に確認:

- [ ] 勧告リストを（package, current, 修正版候補, CVE/GHSA, ecosystem）にパースした
- [ ] `<MIN_AGE_DAYS>` を解決した（repo `.npmrc min-release-age` を優先 / Go・cooldown 無し npm は `7` 既定）。質問は真に曖昧なときだけ
- [ ] 各 package の所在（lockfile ディレクトリ / `go.mod`）を特定し direct vs transitive/indirect を分類、not-present を提示した
- [ ] 修正版を同一 major の最小として選び、major 跨ぎを breaking の可能性としてフラグ、downgrade ガードを適用した
- [ ] 公開日を取得し disposition を設定した（clear / too-new / blocked-by-cooldown）
- [ ] `too-new` / `blocked` の全エントリを `/supply-chain-triage` でトリアージした（baseline = lockfile の現行版）。バンドをサマリと `AskUserQuestion` の選択肢説明へ引き継いだ
- [ ] 日本語サマリを表示し、**clear 非 major は確認なしで適用**、`AskUserQuestion` は major-bump / too-new のみに使用、blocked は deferred（未適用）
- [ ] npm 直接は `npm install --package-lock-only`、推移的は **スコープ付き同一 major フロア** の `overrides`（`">=<fixed> <<next-major>"`。exact pin は既知の壊れた新版のときだけ）+ `npm install --package-lock-only`、lockfile は手編集しない
- [ ] 追加した override を報告で暫定と明示した（親が修正版を native に出したら回収 — 親バンプ・override 削除・再 audit）
- [ ] Go は 1 回にまとめた `go get module@ver ...` + `go mod tidy`、vendoring するなら `go mod vendor`
- [ ] npm 変更に `npm audit`、Go 変更に `govulncheck` + build、スコープに応じ `make lint` / `make test`（major バンプはより入念に — typecheck / package テスト）
- [ ] 生成器を養う依存には drift を確認し、再生成物を含めた
- [ ] 最終日本語報告: 適用したセット、major/too-new の判断、deferred/スキップ項目、検証結果
- [ ] `SKILL.md` 更新後に `SKILL.ja.md` を再同期した
- [ ] commit / stage / push を行っていない
