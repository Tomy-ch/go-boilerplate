# gen-sqlc

`database/dml/<type>/<category>/`のSQLを元に、`sqlc`でGoコードを生成するCLIです。

- 生成先: `internal/infrastructure/rdb/<category>/gen/`（package `gen` 固定）
- 実行場所: Dockerツールコンテナ（`WORKDIR=/app`）
- 用途: dev/CI専用（本番では実行しません）

## なぜテンプレYAMLを一時生成するのか

- **カテゴリごとに`queries`/`out`パスが異なる**ため、単一`sqlc.yaml`では表現しづらいためです。
- `sqlc`は現状、設定を標準入力で受け取れないため、「**テンプレを置換→一時YAML**」が最もシンプルです。
- 実行後に削除するのは、**設定ファイルの増殖を避ける**ためです。
  - 必要なら`.gitignore`除外で残す運用も可能です。

## 並列実行ポリシー

- 生成はI/O待ちが大きいので、並列数(既定4)で十分なスループットを確保します。
- Docker/CI環境で**全コア占有を避ける**ため、`runtime.NumCPU()`を上限にさらに上限(=4)を設定しています。
- 下限の2はI/Oの隙間を埋めるためです。

> 定数:
>
> - `sqlcDBConcurrency = 4`
> - `maxSQLCConcurrency = 4`
> - `minSQLCConcurrency = 2`

## パス解決とWORKDIR

- `WORKDIR=/app`を前提に、`exec.Cmd.Dir=/app`で実行します。
- テンプレ内の`queries:`/`out:`は`/app`起点（絶対 or ルート相対）としています。
- これにより**相対パスのブレ（実行場所依存）を排除**し、どこからでも同じように実行できます。

## よくあるエラーと対処

- `star expansion failed for query`
  → `SELECT *`/`u.*`の展開に失敗。**列を明示**する or **schema.sql** を参照してください。
  - オニオンアーキテクチャで安全にドメインを作成するための必要最小限のSQLを生成するための制約として対応は行っていません。
- `path error: stat ... no such file or directory`
  → YAMLの`queries:`パス不一致。**/app 基準**の絶対パスにするか、`Cmd.Dir=/app` を確認。
- DB接続エラー
  → `DATABASE_URL` を見直し。`pg_isready` / `psql "\conninfo"` で疎通確認。

## 使い方

```bash
# RepositoryとQueryService全カテゴリを生成
make gen-sqlc

# Repository(Domain用)のみを生成
make gen-sqlc-repo

# QueryServiceのみを生成
make gen-sqlc-qs

# Repository(Domain用)で特定カテゴリを生成
make gen-sqlc-repo-<category>

# QueryServiceで特定カテゴリを生成
make gen-sqlc-qs-<category>
```
