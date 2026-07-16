---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, ci]
---

# ADR-0070: mise.toml を単一の情報源とし、バージョンを下流に伝播させ CI でドリフトを検知する

English canonical: [0070-mise-ssot-drift-gate.md](../../adr/0070-mise-ssot-drift-gate.md)

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

## 決定

`mise.toml` をプロジェクトで使用するすべてのツールおよびランタイムバージョンの単一情報源とする。

**言語ランタイムのバージョン伝播:** `[tools]` に宣言された `go`、`node`、`python` の値は
`go.mod`（`go` ディレクティブ）および `docker/server/Dockerfile`・`docker/tools/Dockerfile` の
該当 `FROM` 行へ `make sync-versions` を通じて伝播する。このコマンドは `scripts/sync-versions/main.go`
を実行する。その Go プログラムは事前条件（バージョンの存在確認、ファイルの存在確認、期待される一致数の確認）を
検証したうえで各ファイルをアトミックに書き込むため、失敗しても中間状態が残らない。

**mise のツール解決モデルの外にある Docker イメージバージョン**は `mise.toml` の `[env]` セクションに
宣言する（例: `OTEL_LGTM_VERSION`）。これにより `[tools]` テーブルを汚染せず、同一ファイル内に収める。

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
- CI ドリフトゲートは意図的なガードレールとして機能する。誤ったバージョンが開発ブランチに
  サイレントに入り込むことを防ぐ。`make sync-versions` を実行して結果をコミットする手順は
  想定された意図的なプロセスであり、むしろこのゲートがないほうが危険である。

### ネガティブな影響

- Docker イメージバージョン用の `[env]` セクションは `[tools]` テーブルとはパターンが若干異なる。
  コントリビューターはどちらのセクションがどの種類のバージョンに対応するかを把握しておく必要がある。

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

## 補足

- コンテナベース実行に関する関連決定:
  [ADR-0069](0069-containerized-pinned-toolchain.ja.md)。
- SSOT コメントと `[env]` セクションを持つ `mise.toml`:
  [`mise.toml`](../../../mise.toml)。
- CI ドリフトゲートワークフロー:
  [`.github/workflows/sync-versions-check.yaml`](../../../.github/workflows/sync-versions-check.yaml)。
- 同期スクリプト: [`scripts/sync-versions/`](../../../scripts/sync-versions/)。
