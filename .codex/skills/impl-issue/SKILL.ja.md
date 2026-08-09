> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Impl Issue

GitHub issue を、環境確保からマージ済み PR まで進める半自動パイプライン。進行・記録・検出は機械が持ち、判断はすべて人間が持つ。目的は速さではなく、**長い自律実行がブラックボックスにならないこと** — 承認した計画から外れた箇所が、最後ではなくその場で表に出る。

このファイルは意図的に自己完結させている。**別マシンでもこのファイルだけ読めば同じ実行を再現できる**ようにするため、具体的なコマンドはローカルのメモではなくここに置く。実際に drift するリポジトリ固有の詳細だけをポインタに残す: `.makefiles/README.md`（target 一覧）、`docs/maintenance/db-worktree-pool.md`（スロットプール）、`docs/development-flow.md`（変更種別ごとのフロー）。

## 使うとき

- issue の URL / 番号を渡され、マージ済み PR まで持っていってほしいとき。
- 決定点で止まった実行の再開を頼まれたとき。

以下には使わない: issue の無い変更（`commit` + `submit-pr` を直接）、既存 diff のレビュー（`impl-review` / `test-review`）、スキルの作成・編集（`manage-skill`）。

## このスキルが持たないもの

実装判断を一切持たない。どの設計を採るか、レビュアーが正しいか、その指摘が issue に値するかを決めない。ユーザーへ回して、答えを記録する。

| 作業 | 担当 |
| --- | --- |
| コミット分割と実行 | `commit` |
| push + PR 作成/更新 | `submit-pr` |
| 敵対的コードレビュー | `impl-review` |
| テスト品質レビュー | `test-review`（`impl-review` が連鎖） |
| 実装そのもの | 承認された計画に従うあなた |

## AI の変更範囲

`AGENTS.md` は AI の編集対象を `internal/` / `pkg/` / `database/` / `openapi/` に限り、それ以外 — `.github/workflows/`、`docker/`、`scripts/`、`docs/`、`.makefiles/`、ルートのドットファイル — をスコープ外としている。**このスキルの起動は、その制限を緩めるユーザーの明示指示にあたる。** このスキルは issue 汎用のドライバであり、対象面を決めるのは issue の側だからである。CI・ツールチェーン・コンテナイメージ・ドキュメントについての issue は、既定の 4 ディレクトリの中では解決できない。これは `AGENTS.md` の「Skills must not be a loophole」条項に対する、明示された抜け穴でない例外である。

緩和には境界があり、その境界は計画である:

- Step 3 の計画の **触るファイル一覧** が許可された対象面。既定の 4 ディレクトリの外にあるセンシティブなパスは、編集する**前**にそこへ現れていなければならない。glob で暗に含めるのではなく、明示的に名前を書く。
- 計画を提示するときにそれを言う。ユーザーは diff で気づくのではなく、承知のうえでセンシティブなパスを承認する — 黙ってスコープが広がる計画こそ、この条項が防ごうとしている失敗である。
- 計画に無いセンシティブなパスへ到達したら Step 4 の trip-wire 1。止まって聞く。対象面を広げてから事後報告しない。

このスキルの実行中も保護されるもの（issue が何を要求しても触らない）:

- `AGENTS.md` / `CLAUDE.md`
- 生成物: `**/*.gen.go`、`*.sql.go`、`*_mock.go`、`**/openapi.gen.yaml`、および `docs/` 配下の生成物（`docs/openapi/**`、`docs/coverage/**`、`docs/db-schema/**`、`docs/godoc/**`、`docs/portal/docs.json`、`docs/portal/guides/**`）。`make` ターゲット経由での再生成はよい。手編集は駄目。
- エージェント自身の権限設定の `permissions.deny` 配下
- `database/migrations/**` の既存ファイル（新規 migration ファイルのみ）

## Step 0 — 3 つのモードを確認する（`ask the user explicitly` の対話 1 回）

何より先に、3 問すべてを含む 1 回の `ask the user explicitly` 対話で聞く。既定は明示するが、ユーザーの選択が常に優先。別々の質問を連続して投げる形にしてはならない。

**レビューモード** — レビュー指摘をどうするか。

| モード | 確定した指摘 | それ以外 |
| --- | --- | --- |
| `all` | 大きく書き換わっても適用する | — |
| `harmful`（既定） | 変更範囲内で明確に有害なものだけ適用 | issue モードへ回す |
| `issues` | 何も適用しない | issue モードへ回す |

**issue モード** — 無関係な指摘をどう追跡に載せるか。

| モード | 挙動 |
| --- | --- |
| `search`（既定） | 既存 issue を先に検索し、重複ならそこへコメント |
| `file` | 検索せず起票 |

`issues` × `file` は起票数が最大になる組み合わせ。実行前に件数を出して確認する — レビューは容易に十数件出るし、十数件の新規 issue はそれ自体がノイズ。

**進行モード** — trip-wire に当たったときどうするか。

| モード | 挙動 |
| --- | --- |
| `interactive`（既定） | 止まって聞く |
| `delegated` | 判断を記録して続行し、最後に PR コメントへまとめて出す |

`delegated` は「全権委任 かつ 長期離脱」のための設定。許可を出す人が居ないのに許可待ちで止まると実行が死ぬ。速度設定ではない。

## Step 1 — 起動

```bash
printf '\033]0;%s\007' "<issue-number>-<slug>"   # 並行実行を見分けられるよう窓に印を付ける
gh issue view <n> --json number,title,body,labels,state,comments
```

**何か書く前に、issue 本文をベース実物と突き合わせる。** issue 本文は書かれた時点のスナップショットで、行番号・「X はまだ無い」・「Y に consumer が無い」は陳腐化する。これから切るベースに対して、本文の事実主張を 1 つずつ検証する。

その上で着手コメントを投稿する。ブランチ名・ベース commit・隔離方式、そして何より **上で見つけた食い違いすべて** を書く。issue を「着手済み」とマークすると同時に、まだ安いうちに訂正をユーザーへ渡すため。

```bash
gh issue comment <n> --body-file <file>
```

## Step 2 — 実行環境を確保する

コードに触れる前に済ませる。共有 checkout に何も落とさないため。

既存 worktree を再開する場合は、まず状態を変えずに確認する。worktree のパス、`vendor/` の有無、`.gobp-db-slot` の有無を確認する。slot が無い場合、DB 作業を始める直前に `make slot-acquire` を実行する必要がある。会話を再開するだけで取得してはならない。再開時の無条件な slot 取得や DB 再初期化は禁止する。

### [Codex側の差分]

このリポジトリでは Codex の session-start hook 契約を検証できていない。そのため再開確認は `.codex/hooks.json` ではなく Step 2 に置く。skill を同期するときもこの位置を保つこと。Claude 側の session hook が同じ観測を行っても、Codex 側で再開時の DB slot 取得や DB 再初期化を起こしてはならない。

```bash
# 1. origin の live state から現行のリリース線を解決する。
BASE=$(make -s base-branch)
test -n "$BASE" || { echo "ベースブランチを解決できませんでした"; exit 1; }

# 2. 古いローカル ref ではなく、今の origin から切る。
git fetch origin "$BASE"
git worktree add -b feature/<n>-<slug> ../go-boilerplate.worktrees/<n>-<slug> "origin/$BASE"

# 3. DB スロットをリース: 専用 DB（wt<N>_local / wt<N>_test）、API ポート 8080+N、mock-auth 4000+N。
cd ../go-boilerplate.worktrees/<n>-<slug> && make slot-acquire

# 4. 新規 worktree には vendor/ が無く、air は --mod=vendor でビルドするため、これが無いと serve が失敗する。
go mod vendor
```

`make base-branch` は `origin` の live state を読む。この用途で他の情報は使わない。ローカルの
`refs/remotes/origin/HEAD` は clone 時に一度だけ設定され、`git fetch` では更新されない。GitHub の
default branch は以前のリリース線に留まり、エージェントまたは環境が渡す「main branch」ヒントも、その古い
ローカル symref を報告しうる。いずれも警告なしに答えを返すため、世代の古いベースから分岐しても、期待される
ファイルが無いとエージェントが報告するまで見えない。その時点ではそのブランチ上の作業は無駄になっている。

`slot-acquire` が失敗を報告したら、再試行の前に `make slot-status` を見る — コマンドがエラーでもリースだけ成功していることが多い。

**スロットは自分から解放しない。片付けフェーズで解放を聞きもしない。** 握ったままのコストはほぼゼロ（リースは stale になれば自動回収される）だが、作業途中で失うと DB とポートを失う。本当に終わったかを知っているのはユーザーだけ。

ユーザーの指示が解決結果とは**異なる**リリースバージョンを指定した場合は、分岐前に聞く。意図的な
バックポート先だけは resolver には分からない。

## Step 3 — 計画を立て、そこで待つ

Codex のエージェント委譲機構で計画は**別モデル**に書かせる — 実装者自身の盲点をそのままコードへ持ち込むのを、second model が拾う。issue、Step 1 の訂正、既に読んだパスを渡す。「私の要約を鵜呑みにせず自分で検証しろ」と伝える。現在の Codex surface が別モデルのエージェントを起動できないなら、その制限を明示する。自分で計画を下書きした後、提示前に issue とコードに照らす独立した批評パスを行う。別モデル要件を黙って落としてはならない。

計画はチャットの発言ではなく**ファイル**にする。Step 5 が機械的に突合するため。repo の gitignore された `tmp/` 配下に置く（運用者の好みで repo 外ディレクトリへの symlink でもよい）。次のセクションが必須:

| セクション | なぜ必須か |
| --- | --- |
| 触るファイル一覧 | Step 5 が `git diff --name-only` と突合する |
| 各ステップの成果物 | 途中で止まった実行の再開・引き継ぎができる |
| 確定した選択肢と**棄却した選択肢（理由付き）** | 棄却した案を後から採ったとき trip-wire 2 が発火する |
| ゲート表 | runtime 検証の要否を計画時に固定し、後から黙って落とせなくする |

計画を提示し、**承認を待つ。承認前に実装しない。**

## Step 4 — 実装し、5 つの trip-wire を監視する

承認された計画に従う。以下は意図的に機械的なトリガーにしてある — 「判断が重要だったと**気づく**」ことに依存させるのが、まさに drift が報告されない原因だから。

| # | trip-wire | なぜ人間の判断か |
| --- | --- | --- |
| 1 | 計画に無いファイルを触った | スコープが伸びた。ユーザーが承認したのは別の形 |
| 2 | 計画が棄却した案、または第三の案を採った | 棄却には理由があった。黙って覆すとその理由が捨てられる |
| 3 | アーキ規約由来の lint/CI 失敗（`interfacebloat` / `gocognit` / `depguard` / architest ほか） | これらは書式ではない。満たすには設計が変わる |
| 4 | レビュー指摘を却下した、または提案と別の修正を採った | 指摘が正しくても提案された修正が有害なことがある。単独で決める判断ではない |
| 5 | ゲートを飛ばした | Step 6 参照 |

発火したら: `interactive` → 止めて状況と推奨を提示。`delegated` → 記録して続行し、最後に PR コメント 1 本へまとめる。

### コード生成が塞がれているとき

生成系 make target は `docker compose run … make <target>-ci` の薄いラッパで、`-ci` 側は mise が入れたツールで host 実行する前提で書かれている。コンテナランタイムが使えないときは直接叩く:

```bash
make merge-dml-ci work-dir="."     # DML 連結
make sqlc-generate-ci              # sqlc
make gen-bundle-oapi-ci            # OpenAPI bundle
make gen-api-docs-ci               # docs/openapi/index.html
cd <pkg> && mockgen -source=<f>.go -destination=mock/mock_<f>.go.gen.go -package=mock_<pkg>
```

本当にコンテナが要るのは `make dump-schema` だけで、しかも migration を足したときだけ。

罠が 2 つ。`merge-dml-ci` は `go run ./cmd/` を走らせるため、クエリ生成前に Repository メソッドを足すとビルドが詰まる — 生成の間だけ実装をスタブにし、後で戻す（先に `cp` で退避）。もう 1 つ、埋め込み spec の `//go:generate` はコンテナパスを指しているので `go generate` では回らない。実パスで直接叩く:

```bash
cd internal/controller/httpstack/oapi/validator \
  && oapi-codegen --package=gen --generate=spec -o ./gen/validate.gen.go <repo>/openapi/openapi.gen.yaml
```

OpenAPI の description を 1 行変えるだけでも生成物は 3 つ動く: bundle、`docs/openapi/index.html`、その埋め込み spec。1 つでも漏らすと CI の生成物チェックが落ちる。

## Step 5 — 計画と実物を突合する

ゲートの前に行う。突合するのは:

- `git diff --name-only` ↔ 計画のファイル一覧（増えた分・触られなかった分の両方を報告）
- 実際に採った選択肢 ↔ 計画の確定/棄却リスト
- ゲート表 ↔ 実際に走らせたもの

差分を提示する。長い実行は正当な理由で drift する。問題なのは**ユーザーが見ていない drift**。drift が無ければ 1 行そう言って進む。

## Step 6 — ローカルゲート

`make fix` → `make lint` / `make test`。worktree が多いときは CI へ委譲してよい（文書化されたトレードオフ）が、**ローカル未実行であることを PR に書く**。黙っていると「検証済み」と読まれる。

runtime 検証は意図的にここに置いていない。PR ができた後（Step 8）に置くことで、CI と並行して走らせられる。

## Step 7 — レビュー

`impl-review`（`test-review` を連鎖）を実行し、Step 0 のレビューモードに従って指摘を処理する。

自動適用してよいのは正しさを機械的に判定できるものだけ — 書式、lint 修正、コメント指摘、生成物の再生成。**設計を変える修正は、レビューモード `all` であっても必ず決定点**にする。`all` が許可しているのは「大きな書き換え」であって「未レビューの書き換え」ではない。`impl-review` がコード 5 レンズを報告のみに留めているのと同じ理由。

その後、trip-wire と保留した判断をまとめて 1 箇所で提示する。小出しより一括のほうが、実行全体の形が見える。

## Step 8 — PR → runtime 検証 → マージ

先に `submit-pr` で PR を開く。ローカル検証の間に CI が回り始める。

### runtime 検証 — マージのゲート

動いているシステムに対して実 HTTP 経路を通す。どのモードでも緩められない。

```bash
make serve                                    # API は 8080+N、mock-auth は 4000+N

TOKEN=$(curl -s -X POST http://localhost:400N/bypass/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"user-john-doe","profile":"valid"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["access_token"])')

curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" http://localhost:808N/v1/...
```

token の subject は `user_identities.subject` の文字列（`user-john-doe` 形式）でなければならない。ユーザー UUID ではない — シードの UUID 行はスロットのポートが発行する issuer とは別の issuer に属するため、UUID を渡すと紛らわしい 401 になる。実在する subject は DB から引く:

```bash
docker exec gobp-shared-database-1 psql -U postgres -d wt<N>_local -c \
  "select ui.subject, r.name from user_identities ui
     left join user_roles ur on ur.user_id = ui.user_id
     left join roles r on r.id = ur.role_id
   where ui.issuer = 'http://localhost:400N';"
```

正常系、変更が持ち込むエラー経路、そして保護された操作ならトークン無しで 401 になることを確認する。その上で **LGTM スタックのトレースを読み、リクエストが想定どおりの経路（controller → usecase → infrastructure、意図した SQL）を通ったことを確認する**。ステータスコードだけでは、そのリクエストが変更した層に届いたことの証明にならない — 間違ったが尤もらしい経路は、間違った理由で正しいステータスを返す。

**CI 緑はこの代替にならない。** 過去の実行で、ドキュメント上の API 契約が誤ったままマージされたことがある — 動いているシステムが 401 を返すところを 409 と書いていた。認証が usecase より手前で呼出元を弾いていたためである。5 つのレビューレンズと 29 の CI チェックが全て通っていた。どれもが静的解析か、DB 層で止まるテストだったから。実 HTTP リクエスト 1 本で即座に露見した。

runtime 検証がどうしても実行できないときの正直な選択肢は 2 つで、第三はない。

1. まだマージしない。
2. 同じ HTTP 経路を通す統合テストを足し、それを根拠にマージする。

どちらを採ったかを明言する。

### マージ

別々の foreground sleep 呼び出しで繰り返しポーリングせずに CI を待つ。以下を 1 本の長時間コマンドとして実行し、現在の surface が対応する場合は Codex の background/yield-and-notify 機構で完了通知を 1 回だけ受ける。その機構が無い場合は、この 1 本のループを維持して blocking であることを明示する。foreground sleep-poll の連続に置き換えてはならない:

```bash
until [ "$(gh pr checks <n> --json bucket --jq '[.[]|select(.bucket=="pending")]|length')" = "0" ]; do sleep 30; done
gh pr checks <n>
```

その上で `gh pr merge <n> --merge`。

## Step 9 — 締め

issue は**手動で**クローズする。PR のベースが default branch ではなくリリースブランチのとき、自動クローズのキーワードは発火しない:

```bash
gh issue comment <n> --body-file <handover> && gh issue close <n>
```

申し送りコメントには、決めたこと・計画から想定外だったこと・意図的に手を付けなかったことを書く。

変更範囲の外にある指摘は issue モードに従って追跡へ回す。`search` では、新規起票より既存 issue への申し送りコメントを優先する — issue 数それ自体がコストで、重複は元の issue を埋もれさせる。

**起票の前に、その指摘を動いているシステムで検証する。** コードを読んだだけで導いた指摘は、静的レビューでは捕まえられない形で誤りうる: ある実行で「退会済みユーザーが複数のエンドポイントを呼べる」という issue を起票したが、実際には middleware が、調べていたコードよりずっと手前で全て弾いていた。その issue は NOT_PLANNED でクローズする羽目になった。Step 8 の runtime 段階があれば大抵は確認できる。

最後に、コミットメッセージにも PR 本文にも現れていない自己判断を PR コメントへ記録する。`delegated` モードでは、記録した trip-wire がここに落ちる。

## 二重に聞かせない委譲

サブスキルはそれぞれ自前の質問を持つ。このスキルが既にユーザーと決着させている以上、答えを payload で渡してサブスキル側のゲートを飛ばす — `impl-review` が `scope` / `base_ref` / `reviewer_model` / `skip_verifier` を `test-review` へ渡すのと同じ形。

| サブスキル | 渡すもの | 抑止されるもの |
| --- | --- | --- |
| `commit` | 既に提示済みのコミット分割 | 分割の承認質問 |
| `submit-pr` | レビュー実行済みであること、進行モード由来の push 判断 | Phase 0 のレビュー確認と push 確認 |
| `impl-review` | スコープ、レビュアーモデル、テスト委譲の選択 | Step 0 |

同じことを 2 回聞くと、ユーザーは読まずに承認する癖がつく。それはこのスキルが作ろうとしている決定点そのものを無効化する。

## やること / やらないこと

- ✅ コードに触れる前に worktree とスロットを確保する。
- ✅ issue の主張をベース実物で検証し、食い違いを着手コメントに書く。
- ✅ 実装前に計画の承認を取り、Step 5 が突合できるようファイルとして残す。
- ✅ 5 つの trip-wire を「気づくもの」ではなく機械的トリガーとして扱う。
- ✅ どのゲートを走らせ、どれを走らせなかったかを明言する。
- ✅ ステータスコードだけでなくトレースを読む。
- ✅ issue を起票する前に runtime で指摘を検証する。
- ❌ HTTP 経路を一度も通していない実装コードの変更をマージする。
- ❌ CI 緑を runtime 検証として提示する。
- ❌ 設計を変える修正を、どのモードであれ自動適用する。
- ❌ issue モードが指示しない限り、既存を確認せず起票する。
- ❌ 頼まれてもいないのに DB スロットを解放する、または解放を聞く。
- ❌ foreground の sleep ループで CI をポーリングする。

## チェックリスト

- [ ] `ask the user explicitly` の対話 1 回で 3 モードを確認した。
- [ ] 着手コメントを投稿した（issue ↔ ベースの食い違いを含む）。
- [ ] fetch 済みのベースから worktree を作成。DB slot は DB 作業開始時にのみリースし、必要なら `go mod vendor` を実行した。
- [ ] 別モデルが計画を作成、必須 4 セクションが揃い、実装前に承認された。
- [ ] trip-wire を進行モードどおりに処理した。黙って吸収したものが無い。
- [ ] 計画と実差分を突合した。
- [ ] ローカルゲートを実行した、または CI へ委譲したことを PR に書いた。
- [ ] レビューを実行し、自動適用は機械的に判定できるものに限定した。
- [ ] 決定点をまとめて提示した。
- [ ] PR を開いた上で runtime 検証（curl + トレース）を完了した — または未実施であることと、2 択のどちらを採ったかを明言した — 上でマージした。
- [ ] issue を申し送り付きで手動クローズ。無関係な指摘は issue モードどおりに処理し、各々起票前に検証済み。残った自己判断を PR コメントへ記録した。
