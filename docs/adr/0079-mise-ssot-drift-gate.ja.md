---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, ci]
---

# ADR-0079: mise が解決するバージョンは mise.toml を単一の情報源とし、下流に伝播させ CI でドリフトを検知する

English canonical: [0079-mise-ssot-drift-gate.md](0079-mise-ssot-drift-gate.md)

## ステータス

accepted

## 背景

複数のファイルが同じ言語ランタイムバージョンに一致していなければならない。

- `mise.toml` — 開発者プロビジョニングおよび Docker イメージビルドのバージョンを宣言する。
- `go.mod` — `go` ディレクティブは `mise.toml` の Go バージョンと一致していなければならない。
- `docker/server/Dockerfile` および `docker/tools/Dockerfile` — `FROM golang:X`、`FROM node:X`、
  `FROM python:X` タグは `mise.toml` と一致していなければならない。

これらのファイルを手動で同期し続けることはエラーが起きやすい。`mise.toml` で Go をアップグレードした
コントリビューターが `go.mod` や Dockerfile の `FROM` 行を更新し忘れることがある。検証ゲートがなければ、
その乖離がサイレントにメインブランチに入り込む。

一部の Docker イメージバージョン（例: `grafana/otel-lgtm`）は、mise レジストリからインストールできないため
`[tools]` テーブルで管理できない。それらにも単一の権威ある場所が必要である。

PyPI で公開されているツールも、`[tools]` エントリだけでは足りない 2 つ目のケースである。mise はバージョンを
固定するが、その下は何も固定しない。Python ツールの推移依存は install 時に解決され、pin もハッシュ検証も
一切されないため、同じ `[tools]` エントリでも日によって違う依存ツリーが入る。供給網の中で汚染が最も気づかれ
にくく入り込む部分であり、同時にどの脆弱性スキャナからも見えない部分でもある。`mise.toml` は `osv-scanner`
が読める lockfile ではないからである。

## 決定

`mise.toml` を、mise が解決するすべてのツールおよびランタイムバージョンの単一情報源とする。

**PyPI のツールは mise の外で宣言し固定する。** バージョンは `python/<tool>.in` で宣言し、
`make py-lock`（`uv pip compile --generate-hashes --universal`）が生成する `python/<tool>.txt` が
推移依存のツリー全体を sha256 付きで固定する。install は `uv pip install --require-hashes` を通す。
このオプションはバージョンかハッシュを欠いた要求を拒否するため、ハッシュ検証は省略できる別の検査ではない。
ツールごとに `.in`/`.txt` を 1 組ずつ持つことで、各ツールの解決は互いに独立に保たれる。`uv` 自体と、
lockfile の解決対象となる Python ランタイムは引き続き `mise.toml` が宣言する。

**言語ランタイムのバージョン伝播:** `[tools]` に宣言された `go`、`node`、`python` の値は
`go.mod`（`go` ディレクティブ）および `docker/server/Dockerfile`・`docker/tools/Dockerfile` の
該当 `FROM` 行へ `make sync-versions` を通じて伝播する。このコマンドは `scripts/sync-versions/main.go`
を実行する。その Go プログラムは事前条件（バージョンの存在確認、ファイルの存在確認、期待される一致数の確認）を
検証したうえで各ファイルをアトミックに書き込むため、失敗しても中間状態が残らない。

**mise のツール解決モデルの外にある Docker イメージバージョン**は `mise.toml` の `[env]` セクションに
宣言する（例: `OTEL_LGTM_VERSION`）。これにより `[tools]` テーブルを汚染せず、同一ファイル内に収める。

**宣言と lockfile のドリフトゲート:** `scripts/tool-cooldown` は `mise.toml` と併せて `python/*.in` を
読み、宣言したバージョンが lockfile の固定と食い違っていれば失敗する。この検査が無いと、`.in` を上げて
`.txt` を再生成し忘れたときに、実際には入らないバージョンに対して cooldown ゲートが通ってしまう。

**CI ドリフトゲート:** `sync-versions-check` ワークフローは `mise.toml`・`go.mod`・Dockerfile・
同期スクリプト自体に触れるプルリクエストでトリガーされる。ブランチに対して `go run ./scripts/sync-versions`
を実行し、その後 `git diff --quiet` を確認する。差分が生じた場合はワークフローが失敗し、
ローカルで `make sync-versions` を実行して結果をコミットするよう作成者に指示する。

## 影響

### ポジティブな影響

- `mise.toml` を 1 か所変更するだけで言語ランタイムをあらゆる場所でアップグレードできる。
  残りは同期スクリプトが処理する。
- ドリフトはマージ前の PR レビュー時に検出される。後になってビルドが壊れて発見されることがない。
- Go ベースの同期スクリプトは強力なエラーハンドリングを備え、失敗しても中間状態を残さない。
- Docker イメージバージョンは `[tools]` エントリでなくても単一の権威ある場所を持つ。
- Python のツールはどこでも同じ依存ツリーが入り、ハッシュで検証される。`osv-scanner` も
  `python/*.txt` を lockfile として読める。`[tools]` エントリでは得られなかった範囲である。
- CI ドリフトゲートは意図的なガードレールとして機能する。誤ったバージョンが開発ブランチに
  サイレントに入り込むことを防ぐ。`make sync-versions` を実行して結果をコミットする手順は
  想定された意図的なプロセスであり、むしろこのゲートがないほうが危険である。

### ネガティブな影響

- Docker イメージバージョン用の `[env]` セクションは `[tools]` テーブルとはパターンが若干異なる。
  コントリビューターはどちらのセクションがどの種類のバージョンに対応するかを把握しておく必要がある。
- PyPI のツールは 1 行ではなく 2 ファイルになり、アップグレードは `mise.toml` ではなく
  `python/<tool>.in` の編集と lockfile の再生成になる。この 2 手目が黙って忘れられないようにするのが、
  上記のドリフトゲートである。

## 検討した代替案

### Renovate / Dependabot によるファイルごとの PR

自動依存関係 PR は個々のファイルを最新状態に保つが、各ファイルを独立して扱う。ファイル間の整合
（例: `go.mod` と `Dockerfile` がともに `mise.toml` で宣言された同一の Go バージョンを追跡すること）は
依然として調整が必要になる。単一ソースアプローチはその関係を明示的かつ機械的に検証可能にする。

### シェルベースの同期スクリプト

シェルスクリプトは記述が簡単だが、エッジケース（単語分割、ツールの欠如、移植性）で脆弱になる。
Go プログラムは追加の依存なしに `go run` で実行でき、適切なエラーハンドリングとアトミック書き込みを提供する。

### 同期なし・各ファイルを独立して管理

非常に小規模なチームでは許容できるが、スケールしない。マルチコントリビュータプロジェクトでは
ドリフトインシデントが繰り返しコストとなる。

### PyPI のツールも `[tools]` に置き、推移依存が固定されないことを受け入れる

1 ツール 1 行という形は単純で、lockfile を導入するまではこのリポジトリもそうしていた。却下したのは、
その単純さが「誰も見ていない依存ツリー」から買われているためである。ツール自身の依存は浮いたままで、
1 週間空けた 2 回のビルドは違うコードを install し、どのスキャナもその宣言を読めない。

### `pyproject.toml` + `uv.lock`

uv 本来のプロジェクト lockfile もアーティファクトごとにハッシュを記録し、狙い撃ちの更新もできる。
却下したのは、これが「1 つのプロジェクト・1 つの解決」をモデルにしているためである。ここにあるのは
エコシステムを共有するだけの無関係な CLI であり、本来は起こらないはずの依存衝突を宣言して回避する
必要が出てくる。Go リポジトリのルートに Python のプロジェクトマニフェストを置くことにもなる。

## 補足

- コンテナベース実行に関する関連決定:
  [ADR-0078](0078-containerized-pinned-toolchain.ja.md)。
- SSOT コメントと `[env]` セクションを持つ `mise.toml`:
  [`mise.toml`](../../mise.toml)。
- PyPI ツールの宣言と lockfile: [`python/`](../../python)。
- lockfile 再生成のターゲット: [`.makefiles/python/lock.mk`](../../.makefiles/python/lock.mk)。
- CI ドリフトゲートワークフロー:
  [`.github/workflows/sync-versions-check.yaml`](../../.github/workflows/sync-versions-check.yaml)。
- 同期スクリプト: [`scripts/sync-versions/`](../../scripts/sync-versions)。
- 宣言と lockfile のドリフトゲート: [`scripts/tool-cooldown/`](../../scripts/tool-cooldown)。
