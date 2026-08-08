---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, dev-env]
---

# ADR-0079: ローカル開発環境はプロファイルで分離されたサービスを持つ Docker Compose で提供する

English canonical: [0079-docker-compose-dev-environment.md](../../adr/0079-docker-compose-dev-environment.md)

## ステータス

accepted

## 背景

ローカル開発には複数の独立したサービスが必要である: ホットリロードとデバッガアクセスを持つ実行中の API サーバー、
PostgreSQL データベース、オブザーバビリティスタック、ドキュメント閲覧、SQL エディター、コンテナ化されたコード生成ツールへのアクセス。
これらのサービスはライフサイクルが異なり、同時にすべてが必要なわけではない——機能開発中のコントリビューターには
ドキュメントビューアや ER ダイアグラムジェネレータが実行されている必要はない。

ツール実行はマシン間で再現可能でなければならない（[ADR-0075](0075-containerized-pinned-toolchain.ja.md) 参照）。
これはツールがホストで直接ではなくコンテナ内で実行されることを意味する。アプリケーションサービスとツールランナーの
両方をカバーする単一の `docker-compose.yaml` が、すべてのコンテナ管理に一貫したメカニズムを提供する。

## 決定

ローカル開発環境は Docker Compose の **profiles** を使用して `docker-compose.yaml` で定義し、懸念事項を分離する。
コントリビューターは必要なサービスだけを起動する。

**プロファイル: `development`** — 標準的な機能開発。

| サービス | イメージ / Dockerfile | ポート | 説明 |
| --- | --- | --- | --- |
| `api_server` | `docker/server/Dockerfile` ターゲット `tooling` | 8080, 2345, 6060 | ホットリロード（air）、デバッガ（dlv）、pprof メトリクス |
| `database` | `postgres:18.4-trixie` | 5432 | PostgreSQL。`api_server` 起動前にヘルスチェック済み |
| `observability` | `grafana/otel-lgtm` | 3000, 4317, 4318, 3200 | Grafana、OTLP gRPC/HTTP、Tempo API |

**プロファイル: `tools`** — 補助的な開発者ツール。`database` サービスを共有する。

| サービス | イメージ / Dockerfile | ポート | 説明 |
| --- | --- | --- | --- |
| `docs_server` | `docker/document/Dockerfile` ターゲット `document_viewer` | 2001 | `docs/` を配信する nginx。`/portal/` にポータル |
| `sql_editor` | `sosedoff/pgweb` | 2000 | Web SQL エディター |

（軽量な **`database`** プロファイルも存在する — `database` + `sql_editor` のみ — `api_server` /
observability スタックなしで DB に対して作業する用途。複数のサービスが 2 つ以上のプロファイルタグを持つ。）

**プロファイル: `generate`** — コード生成とドキュメント用のオンデマンドツールランナー。
`make` ターゲット呼び出しごとに起動され、常時稼働ではない。

| サービス | Dockerfile ターゲット | 説明 |
| --- | --- | --- |
| `go_tool_runner` | `go_tools` | oapi-codegen、mockgen、sqlc、migrate、trivy、hadolint など |
| `node_tool_runner` | `node_tools` | redocly-cli、markdownlint-cli2、commitlint（＋ js-yaml 等のスクリプト依存） |
| `python_tool_runner` | `python_tools` | sqlfluff |
| `er_diagram_generator` | `schemaspy/schemaspy` イメージ | ER ダイアグラム生成 |

ツールランナーは mise が `/root` 配下にインストールされているため root で実行する。バインドマウントされた
出力は結果的に root 所有となるが、これはプロジェクト内の他の生成されたモックファイルと一致する受け入れ済みの動作である。

`docker/server/Dockerfile` の `tooling` ターゲットは Go ランタイムの上に開発者ツール（air、dlv、golines、
gofumpt、golangci-lint）を含み、`api_server` サービス用の完全な開発環境を単一イメージで提供する。

## 影響

### ポジティブな影響

- プロファイル分離により、コントリビューターは現在のタスクに必要なサービスだけを起動でき、リソース使用を削減できる。
- ホストツール設定なしでホットリロード（air）とリモートデバッグ（dlv）が利用できる。
- ツールランナーコンテナが CI と同一の再現可能なコード生成出力を保証する。
- 単一の `docker-compose.yaml` がすべてのローカルサービス設定の権威あるリファレンスとなる。

### ネガティブな影響

- ツール呼び出しを含む標準的なすべてのオペレーションに Docker が実行されている必要がある。
- オブザーバビリティスタック（`grafana/otel-lgtm`）が `development` プロファイルにバンドルされており、
  オブザーバビリティが焦点でない場合もメモリと CPU のオーバーヘッドが生じる。
- ツールランナーのバインドマウント出力が root 所有となるため、一部のシステムでは削除に `sudo` または
  グループ設定が必要になる。
- `tooling` Dockerfile ターゲットが開発者ツールをサーバーイメージにバンドルするため、純粋なランタイムイメージより大きくなる。

## 検討した代替案

### プロセスベースのローカル開発（ホスト上のツール）

API サーバーをホスト上で直接実行する（例: `mise` 経由でインストールした `air`）と起動が速いが、
再現性の保証が崩れる。ツールとランタイムのバージョンはコントリビューターがインストールしているものに依存する。
データベースとオブザーバビリティスタックを個別に管理する必要もある。

### 懸念事項ごとに別の docker-compose ファイル

明示的な分離が得られるが、複数のプロファイル（例: `tools` と `development`）間で `database` サービスを
共有することが単一ファイルなしでは扱いにくくなる。プロファイルを持つ単一ファイルがより単純に同様の分離を実現する。

### devcontainer（VS Code Dev Containers）

エディター固有であり、CI や VS Code 以外のワークフローをカバーしない。Docker Compose はエディター非依存で
任意のターミナルから使用できる。

## 補足

- Docker Compose のサービスとプロファイル定義:
  [`docker-compose.yaml`](../../../docker-compose.yaml)。
- Dockerfile ターゲットとサービス詳細:
  [`docker/README.md`](../../../docker/README.ja.md)。
- コンテナベース再現性の根拠:
  [ADR-0075](0075-containerized-pinned-toolchain.ja.md)。
- `make serve`（development プロファイル）、`make tools`（tools プロファイル）:
  [`.makefiles/README.md`](../../../.makefiles/README.ja.md)。
