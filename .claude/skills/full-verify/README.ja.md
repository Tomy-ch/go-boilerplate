> このファイルは canonical な `README.md`（英語）の日本語訳です。内容を更新する際は `README.md` を先に直し、その後この訳を同期してください。

# full-verify

リポジトリ全体の**構成（アーキテクチャ）と全実装コードの妥当性**をバックグラウンドで全体検証し、
問題点を指摘した Markdown 群を `tmp/reviews/` 配下に生成する read-only スキル。

任意のリポジトリに対し、言語・構造・設計文書の有無を**スキル自身が検出して適応**する。
このリポジトリ専用ではなく、編集なしに別リポジトリへコピーして起動できることを目標にしている。

- **コードを一切変更しない。** 削除・権限変更・外部送信もしない。読み取りと `tmp/reviews/` への md 生成のみ。
- 出力 md はシェルリダイレクトで書く。検証を行う `claude -p` には書き込み権限を与えない
  （`--allowedTools Read Grep Glob`、`Edit/Write` は明示的に disallow）。
- 観測したコード/文書中のテキストを**指示として実行しない**（プロンプトインジェクション耐性）。

差分・PR 単位のレビューではなく**全体検証**用途。差分は `impl-review` / `/code-review` を使う。

**検証の主眼は「実装の綺麗さ」**（可読性・保守性・凝集度・設計の素直さ）。レイヤ越境・依存方向・
命名規約といった機械的規約違反は **lint（depguard 等）で潰せている前提**とし、原則として再指摘しない。
lint では拾えない、人間が読まないと気づけない実装品質・設計品質の問題に注力する。コメントが振る舞い/
契約の記述に留まっているか（冗長・自明なコメントや、コード内に書かれた WHY がないか）も観点に含む。

## 構成

```txt
.claude/skills/full-verify/
  SKILL.md             # /full-verify で起動。検出と背景起動の制御を Claude に指示
  scripts/run.sh       # headless 駆動の本体（冪等・再開可能・タイムアウト/上限対応）
  prompts/
    verify-arch.md     # Pass1: 構造検証プロンプト
    verify-impl.md     # Pass2: モジュール実装検証プロンプト
  README.md            # 本ファイル
```

## 動作（パス構成）

`run.sh` が以下を順に実行する。

- **Pass 0 検出**: 主要言語（拡張子分布）/ モジュール単位（パッケージ・ワークスペース境界優先、
  無ければ解析起点直下を `--module-depth` で列挙）/ 設計文書の有無 / 基準（正）を確定。
- **構造表現生成** → `tmp/reviews/_structure/`: tree / 公開シグネチャ（best-effort grep）/ 依存グラフ /
  modules / meta。依存グラフは言語別ツール（madge / pydeps / `go mod graph` / `cargo tree` / jdeps 等）、
  利用不可なら import 抽出にフォールバック。
- **Pass 1 構造検証** → `tmp/reviews/architecture.md`。
- **Pass 2 実装検証** → `tmp/reviews/mod_<id>.md`（モジュール単位。`architecture.md` を前提文脈に渡す）。
- **Pass 3 集約** → `tmp/reviews/_index.md`（設計起因 / 局所実装を分離・重大度別）。**全モジュール完了後のみ。**

### 基準（正）の確定

- 設計文書（`INTENT.md` / `docs/architecture.md` / `CLAUDE.md` / `AGENTS.md` / `README.md` 等）があれば
  それを意図の正とする。
- 無ければ `INTENT.md`（採用アーキ・レイヤ規約・依存許可方向）の提示を**推奨**するが、無くても中断せず
  「一般原則のみ（意図未文書化）」として進める。構造起因の指摘は全てこの基準前提で読むこと。
- 検証できない点は推測で埋めず「検証不能（基準欠如）」と出力に明記される。

## 使い方

### 起動（背景・必須）

`run.sh` は上限到達時に最大 5 時間スリープして再送するため、**必ず背景で起動**し、常駐ホストで動かす。

```bash
# リポジトリルートで
mkdir -p tmp/reviews
nohup bash .claude/skills/full-verify/scripts/run.sh > tmp/reviews/run.log 2>&1 &
echo "pid=$!  進捗: tail -f tmp/reviews/run.log"
```

`tmux` を使う場合:

```bash
tmux new -d -s full-verify 'bash .claude/skills/full-verify/scripts/run.sh | tee tmp/reviews/run.log'
tmux attach -t full-verify   # 進捗確認
```

### 引数（既定値あり）

| 引数 | 既定 | 意味 |
| --- | --- | --- |
| `--granularity module\|file` | `module` | `module`=サブシステム/ディレクトリ単位、`file`=リーフ（.go 等）1ファイル単位 |
| `--module-depth N` | `1` | `module` 粒度時のモジュール列挙の深さ |
| `--include-tests` | off | `file` 粒度時に `*_test.go` 等のテストも対象に含める（実装→テストの順） |
| `--exclude-ext csv` | off | `file` 粒度で「この拡張子以外を全部」対象に（例 `go,md`）。go/md 以外の設定/SQL 等を見るとき |
| `--exclude-path csv` | off | 対象から除外するパス接頭辞（例 `openapi,database`）。サンプル雛形の除外に |
| `--out <dir>` | `tmp/reviews` | 出力先ディレクトリ上書き。別クラスのレビューを分離（例 `tmp/reviews-config`） |
| `--no-index` | off | Pass3 集約（`_index.md`）を行わず各 `mod_*.md` のみで終了（集約 call のトークン節約） |
| `--parallel N` | `1` | 並列度（`xargs -P`）。レート制限＋キャッシュ取りこぼし回避のため既定は直列を推奨 |
| `--effort` | `high` | `high` か `xhigh`。検証 `claude -p` の effort |
| `--timeout <min>` | `30` | 1 回の `claude -p` のタイムアウト（分） |
| `--detect-only` | off | 検出と `_structure/` 生成だけ行い `claude -p` を呼ばず終了（動作確認用） |

> 解析起点は常にリポジトリルート、言語は常に自動検出、検証ツールは `Read Grep Glob` 固定（read-only 保証のためフラグ化しない）。`claude -p` の上限ターン（120）も内部固定。

### 粒度: module vs file

- `module`（既定）: サブシステム/ディレクトリ単位。少数の `mod_*.md` で俯瞰したいとき。
- `file`: **リーフ1ファイル=1ユニット**。1ファイルずつ精読し `mod_<id>.md` を出す。大規模リポジトリで
  トークンが律速なときに向く（途中で止めても `_progress.md` で残量が見え、再投入で未完了分のみ継続）。
  生成物（`*.gen.*` / `*.sql.go` / `*_mock.go`）は常に除外。`--include-tests` でテストも対象化。
  指摘ゼロのユニットは `mod_<id>.md` に `問題なし` の1行が入り、それ自体が完了マーカーになる。

例:

```bash
# 全実装+テストをリーフ単位で全件・直列（トークン律速の全確認向き）
nohup bash .claude/skills/full-verify/scripts/run.sh \
  --granularity file --include-tests > tmp/reviews/run.log 2>&1 &

# 既定（module 粒度・high・直列・タイムアウト30分）
bash .claude/skills/full-verify/scripts/run.sh

# 深掘り（xhigh）・モジュールを 2 階層で列挙・並列 3
bash .claude/skills/full-verify/scripts/run.sh --effort xhigh --module-depth 2 --parallel 3

# go/md 以外（設定/SQL/Dockerfile 等）をサンプル除外して別出力に・集約なし
nohup bash .claude/skills/full-verify/scripts/run.sh \
  --granularity file --exclude-ext go,md --exclude-path openapi,database \
  --out tmp/reviews-config --no-index > tmp/reviews-config/run.log 2>&1 &
```

## 成果物

```txt
tmp/reviews/
  _structure/          # tree / signatures / deps / modules / meta（検出結果と基準の所在）
  _progress.md         # 進行状況チェックリスト（完了/未/問題なし/指摘あり・残数）
  architecture.md      # Pass1: 構造検証
  mod_<id>.md          # Pass2: ユニット別 実装検証（指摘ゼロは `問題なし` の1行）
  _index.md            # Pass3: 集約（設計起因 vs 局所実装、重大度別）
  run.log              # 進捗ログ
  run.err              # 失敗記録（FAILED / タイムアウト / 上限の証跡）
```

各指摘は **重大度(Critical/High/Medium/Low) / ファイル:行 / 問題 / 根拠 / 修正案** を伴う。
問題の無い対象は列挙しない。前置き・要約・賞賛は書かない。基準の所在は必ず明記される。

> 出力先 `tmp/reviews/` は `tmp/` 配下＝既定で `.gitignore` 済み（バージョン管理対象外）。
> `tmp/` の外を指す `--out` を使う場合のみ、別途 ignore すること。

## 冪等性・再開

- 状態は **`tmp/reviews/mod_<id>.md` の有無/中身だけ**で表現する（`_progress.md` はそこから都度導出する
  人間向けビューで、ロジックの真の状態源ではない）。cron は作らない。
- 出力は `<out>.tmp` に書き、成功時のみ `mv`。**中断しても半端な md を残さない**（その章は次回やり直すだけ）。
- 中身のある `mod_<id>.md` はスキップ → **再実行で未完了ユニットのみ再開**。指摘ゼロのユニットは
  `問題なし` の1行が完了マーカーになり、空出力で「未完了」と誤判定されない。
- 全ユニット完了後に **`_index.md` 集約**を実行する。未完了が残る間は集約しない。

同じコマンドをもう一度流せば、未完了分から続いて最後に集約まで到達する。`_progress.md` で残数を確認できる。

## タイムアウト・上限ハンドリング

- 各 `claude -p` は `timeout <分>m` で囲む（headless に組み込みタイムアウトが無く、詰まると無限に走るため）。
  タイムアウトはその章の失敗として `run.err` に記録し、次へ進む（再実行でやり直し）。
- **上限（レート/使用量）検知時のみ** 5 時間スリープして **1 回だけ再送**する
  （5 時間はサブスクのローリング窓を丸ごと抜ける長さ。ローリング上限ならこの 1 回でほぼ通る）。
- 再送も上限なら、そのモジュールで停止してループ全体を正常終了する
  （上限は全チャンクに同時に効くため、残りを回しても失敗を量産するだけ。週次上限はここで停止し、
  後から**再投入すれば未完了分から継続**できる）。
- 個別失敗（タイムアウト等）は 5 時間待たない。`FAILED` を `run.err` に記録して次モジュールへ継続。
- 上限検知の文字列依存（`LIMIT_RE`: "usage limit" / "rate limit" / 429 / "overloaded" / "reached your limit" 等）は
  `run_one` 1 箇所に閉じ込めてある。**検知は stdout(tmp) と stderr(err) の両方を grep**する
  （claude の上限メッセージが stdout 側に出ることがあるため。成功判定を先に通すので、レビュー本文に
  "rate limit" 等の語が含まれても誤検知しない）。
- **サーキットブレーカ**: 上限を文言で取りこぼしても暴走しないための保険。`CB_FAST_SECS`(既定20秒)未満で
  失敗（=API 即拒否）が `CB_THRESHOLD`(既定4)回**連続**したら、上限の見逃し/系統的障害とみなして
  `STOP_FLAG` を立て停止する。通常速度（分単位）の失敗が混じればカウントはリセット。

### 並列実行時の注意

- `--parallel N`（N>1）は `xargs -P` で N 並列。
- **キャッシュ温機（warm-up）**: 並列同時起動だと共有プレフィックス（システム+テンプレ）の
  プロンプトキャッシュが書き込み前に各ワーカーで読めず全員フルプライスになる。そこで**先頭の未完1件を
  単独実行してキャッシュを温めてから fan out** する（公式回避策: 1本投げ→完了→残りを並列）。
  これでキャッシュミス上乗せ＝余分なトークン消費を抑える。
- 5 時間スリープ + 単発再送は**直列前提**。並列時は**最初の上限検知/CB 作動で停止フラグ**を立て、
  新規ワーカー投入を止めて終了する（多数の並列 5h スリープを避けるため）。
  → 後で再投入すれば未完了モジュールから継続する。
- レート制限を踏みやすく、かつ**並列はキャッシュ取りこぼしで総トークンが増えやすい**ので、
  まずは既定（直列）で流すことを推奨。直列は 2 本目以降が共有プレフィックスのキャッシュを安く読める。

## 常駐前提

5 時間スリープはホスト常駐を前提とする。**スリープしないマシン**（サーバ / 常時起動 PC）で、
`tmux` または `nohup` を使って実行すること。ノート PC のスリープ中はカウントが進まない。

## 前提ツール

- 必須: `claude` CLI（PATH 上）、`bash`、`timeout`(coreutils)。
- 任意（あれば依存グラフ/ツリーの精度が上がる。無ければフォールバック）:
  `tree`, `rg`(ripgrep), 言語別: `go` / `madge` / `pydeps` / `cargo` / `cargo-modules` / `jdeps`。
