# Local Development Environment

English: [local-environment.md](../../maintenance/local-environment.md)

ローカル開発を構成する **docker compose の 2 層構成（共有インフラ / checkout 毎の app）・
ホットリロード（air）・コード生成 runner・`make serve` の worktree スロットリング** を 1 枚で
俯瞰するための地図。各要素の詳細は既存の正本ドキュメントへリンクするに留め、ここでは
**全体像と役割分担**だけを示す（重複再掲を避け、ドリフトを防ぐ）。

- make ターゲットの一覧 → [`.makefiles/README.md`](../../../.makefiles/README.ja.md)
- 生成物 / root 所有 / DB 落ち等の詰まり所 → `repo-ops` スキル
- worktree × DB スロットプールの詳細 → [db-worktree-pool.ja.md](db-worktree-pool.ja.md)
- o11y の送出配線 → [observability.ja.md](../design/observability.ja.md)、認証（疑似 OIDC）→ [auth.ja.md](../design/auth.ja.md)

## 全体像

```mermaid
graph TB
  dev["開発者 / make"]

  subgraph infra["infra 層 - プロジェクト gobp-shared（全 checkout で 1 インスタンス）"]
    db[("database<br/>PostgreSQL 18<br/>:5432 固定")]
    obs["observability<br/>otel-lgtm<br/>Grafana :3000 / OTLP :4317,:4318"]
    gar["garage (+ garage_init)<br/>S3 互換ストレージ<br/>:3900 / :3903"]
    docs["docs_viewer :7001"]
    sql["sql_editor :7000"]
    er["er_diagram_generator :5433"]
    subgraph runners["tool-runner（profile: generate / user: root）"]
      go["go_tool_runner"]
      node["node_tool_runner"]
      py["python_tool_runner"]
    end
  end

  subgraph app["app 層 - プロジェクト APP_PROJECT（checkout 毎に 1 つ）"]
    api["api_server<br/>air + dlv<br/>:8080+N / dlv :2345+N / pprof :6060+N"]
    auth["mock_auth_server<br/>疑似 OIDC / JWKS<br/>:4000+N"]
  end

  dev -->|make serve| api
  dev -->|make gen-api / gen-query / lint| runners
  api -->|host.docker.internal:5432| db
  api -->|OTLP 送出| obs
  api -->|S3 API| gar
  api -->|JWKS で JWT 検証| auth
```

## compose の 2 層構成（infra / app）

compose のサービスは 2 層に分かれており、主 checkout と任意個の worktree が同時に `make serve`
できる。以下の変数は [`.makefiles/docker/compose.mk`](../../../.makefiles/docker/compose.mk) で定義する。

| 層 | compose プロジェクト | サービス | ライフサイクル |
| --- | --- | --- | --- |
| **infra** | `INFRA_PROJECT` = `gobp-shared`（固定名） | `database` / `observability` / `garage`（+ `garage_init`）— 固定ポートでしか動けないもの全て | `make infra-up` で起動。`make serve` / `job` / `worker` / `outbox-relay` が冒頭で冪等に呼ぶ。`make infra-down` は**全 checkout に影響する**停止 |
| **app** | `APP_PROJECT` = DB スロット保持時は `gobp-wt-N`、未取得なら `gobp-app-<ディレクトリ名>` | `api_server` / `mock_auth_server` | `make serve` で起動。`make serve-stop` は**この checkout の app だけ**を停止 |

- app 層は `docker compose -p $(APP_PROJECT) -f docker-compose.yaml -f docker-compose.attach.yaml --profile development` で起動する。
  [`docker-compose.attach.yaml`](../../../docker-compose.attach.yaml) はスロット保持時だけでなく**常に**重ねられ、
  `DB_HOST` / `OBS_OTLP_ENDPOINT` / `OBJECT_STORAGE_ENDPOINT` / `AUTH_ISSUER` を実行時 env で上書きして
  共有インフラを `host.docker.internal` 経由で参照させる（`internal/config` の loader は実行時 env を
  `env/.env` より優先する）。
- app 層のホスト公開ポートは全てスロット番号 `N` で相対化される: API `8080+N` / mock 認証 `4000+N` /
  dlv `2345+N` / pprof `6060+N`（スロット未取得なら `8080` / `4000` / `2345` / `6060`）。
  コンテナ内部のポートは動かない。
- observability は **全 checkout で共有**。稼働中の全 app の traces / metrics / logs が
  `http://localhost:3000` の 1 つの Grafana に集まる。
- DB ツーリング（`make db-migrate-*` / `db-seed` / `db-drop-tables` / `gen-query` 等、
  `docker compose run --rm *_tool_runner` と `docker compose exec database` の呼び出し）はプロジェクト名を
  明示しない。`compose.mk` が `COMPOSE_PROJECT_NAME` の既定を `gobp-shared` に置くため、これらは infra 層の
  ネットワークで動く。補助サービス（`docs_viewer` / `sql_editor` / `er_diagram_generator`）も同じプロジェクトに属する。

## コンテナ群

| サービス | 層 | 由来 | ホストポート | 役割 |
| --- | --- | --- | --- | --- |
| `api_server` | app | build `docker/server/Dockerfile` | `${API_HOST_PORT:-8080}:8080` / dlv `${DLV_HOST_PORT:-2345}:2345` / pprof `${PPROF_HOST_PORT:-6060}:6060`（内部ポートは固定） | アプリ本体。dev target は **air** で起動しホットリロード＋delve デバッグ |
| `mock_auth_server` | app | build `docker/mock-auth-server/Dockerfile` | `${MOCK_AUTH_HOST_PORT:-4000}:4000`（内部 4000） | 疑似 OIDC 認証サーバー（JWT テストプロバイダ）。RS 側の JWKS 検証相手 |
| `database` | infra | `postgres:18.3-bookworm` | `5432` 固定 | 全 checkout 共有の**単一**インスタンス（並列化は DB 名で行う。下記スロットリング参照） |
| `observability` | infra | `grafana/otel-lgtm` | `3000`（Grafana UI）/ `4317`（OTLP gRPC）/ `4318`（OTLP HTTP）/ `3200`（Tempo API） | 全 checkout の traces / metrics / logs の受け皿。profile: `development` |
| `garage` | infra | build `docker/garage/Dockerfile` | `3900`（S3 API）/ `3903`（Admin API） | ローカル開発用の S3 互換オブジェクトストレージ（テストは in-process の gofakes3 を使う） |
| `garage_init` | infra | build `docker/garage/Dockerfile` | なし（one-shot） | garage のレイアウト / バケット / アクセスキーの冪等プロビジョニング |
| `docs_viewer` | infra | build `docker/document/Dockerfile` | `7001:80` | 開発用ドキュメントビューア |
| `sql_editor` | infra | `sosedoff/pgweb` | `7000:8081` | ブラウザ DB クライアント |
| `er_diagram_generator` | infra | `schemaspy/schemaspy` | `5433:3000` | ER 図生成 |
| `go_tool_runner` / `node_tool_runner` / `python_tool_runner` | infra | build `docker/tools/Dockerfile`（各 target） | なし（run/exec 実行） | コード生成・lint 等のツールボックス。**`user: root`**・profile: `generate`・リポジトリを `.:/app` にバインド |

> `docs_viewer` / `sql_editor` は API スロット帯（`8080+N`）との衝突回避で **7000 番台へ退避**済み。

## ホットリロード（air + delve）

`api_server` の dev target は `CMD ["air", "-c", ".air.toml"]`。[`.air.toml`](../../../.air.toml) が
`internal` / `cmd` / `pkg` 配下の `.go` 変更を監視し、`go build`（`-gcflags='all=-N -l'` でデバッグ情報を保持）
→ `tmp/main` を生成 → **delve**（`dlv --listen=:2345 --headless … exec --continue`）で `serve` を起動する。
ソース保存で自動再ビルド・再起動され、公開された dlv ポート（`2345+N`）に IDE のリモートデバッガをアタッチできる。

## コード生成 runner（tool-runner）

`make gen-api` / `gen-query` / `lint` などは、ホストの Go/Node/Python に依存せず
**tool-runner コンテナ内で実行**される（`docker/tools/Dockerfile` の go / node / python target）。
リポジトリを `.:/app` にバインドし、コンテナ内 **root** で走るため、生成物がホスト側で root 所有になり
`git` が触れなくなる等の典型的な詰まりがある。**具体的な復旧コマンドは `repo-ops` スキルを参照**
（ここでは再掲しない）。ターゲット一覧は [`.makefiles/README.md`](../../../.makefiles/README.ja.md)。

## server + API スロットリング（worktree 並列）

複数の git worktree（および主 checkout）が **単一の共有インフラ** を衝突なく並列利用するための仕組み。
**分離軸は「別ポートの DB」ではなく「共有 DB 内のデータベース名」**で、app 層のホストポートだけを
スロット番号 `N` でずらす。

```mermaid
graph LR
  subgraph shared["infra 層（gobp-shared）"]
    pg[("PostgreSQL :5432")]
    obs["observability :3000 / :4318"]
  end
  main["主 checkout<br/>project gobp-app-&lt;dir&gt;<br/>api :8080 / auth :4000<br/>DB: local / test"]
  wt1["worktree #1（スロット 1）<br/>project gobp-wt-1<br/>api :8081 / auth :4001<br/>DB: wt1_local / wt1_test"]
  wt2["worktree #2（スロット 2）<br/>project gobp-wt-2<br/>api :8082 / auth :4002<br/>DB: wt2_local / wt2_test"]

  main --> pg
  wt1 --> pg
  wt2 --> pg
  main --> obs
  wt1 --> obs
  wt2 --> obs
```

- **DB ポートは `5432` 固定**（`5432+N` ではない）。スロット `N` は DB 名 `wt<N>_local` / `wt<N>_test` で分離。
- `API_HOST_PORT = 8080+N`、`MOCK_AUTH_HOST_PORT = 4000+N`、`DLV_HOST_PORT = 2345+N`、`PPROF_HOST_PORT = 6060+N`。
- スロットを取らない checkout は既定の DB 名（`local` / `test`）と既定ポートのまま動くため、
  スロット取得は**並列作業のための opt-in** に留まる。
- リース・ブートストラップ・`make slot-acquire` / `slot-free` / `slot-release` 等の**全詳細は正本の [db-worktree-pool.ja.md](db-worktree-pool.ja.md) を参照**。

## 関連ドキュメント

| 目的 | 参照先 |
| --- | --- |
| make ターゲット一覧 | [`.makefiles/README.md`](../../../.makefiles/README.ja.md) |
| worktree × DB スロットプール（正本） | [db-worktree-pool.ja.md](db-worktree-pool.ja.md) |
| 生成物 / root 所有 / DB 落ちの復旧手順 | `repo-ops` スキル |
| Observability の送出配線 | [observability.ja.md](../design/observability.ja.md) |
| 認証（疑似 OIDC / JWKS） | [auth.ja.md](../design/auth.ja.md) |
