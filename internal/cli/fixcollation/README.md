# fix-collation コマンド

概要: `fix-collation` コマンドは、実行環境の照合順序（collation）バージョン不整合が原因で発生する問題を解消するために、
対象データベースに対して必要なメンテナンス SQL を実行する CLI ユーティリティです。具体的には `REINDEX DATABASE` と
`ALTER DATABASE <db> REFRESH COLLATION VERSION` を実行します。

## 役割

- OS のロケールライブラリとデータベースの照合順序バージョンに差がある場合に発生する不整合（mismatch）を修正するための操作を自動化します。

## 実装の要点

- 実行コマンド: デフォルトで `psql` を使用します（ソース内の `psqlCommand` 変数）。
- 実行場所: `workDir` フィールドで指定されたディレクトリ（実装では `/app`）を作業ディレクトリとしてコマンドが実行されます。
- 実行内容: 以下の順で psql を呼び出します。
  1. `REINDEX DATABASE <database>;`（トランザクション内では実行不可なため単独で実行）
  2. `ALTER DATABASE <database> REFRESH COLLATION VERSION;`（単独で実行）
- コマンド呼び出しは `exec.CommandContext` を用いて行われ、標準出力/標準エラーは実行環境にそのまま出力されます。
- psql 実行時に `-v ON_ERROR_STOP=1` を付与しているため、SQL 実行エラーが発生すると即座に停止します。
- ロギング: 実行開始・失敗・成功のログをアプリケーションロガーへ出力します。

## 使い方

- CLI コマンドとして使用します（アプリケーションに組み込まれている想定）。
- フラグ:
  - `--database` : 対象データベース名（デフォルト: `local`）
- 例:
  - `./your-app fix-collation --database local`

## 前提 / 要件

- 対象は PostgreSQL の機能（`REINDEX DATABASE` / `ALTER DATABASE ... REFRESH COLLATION VERSION`）を使用するため、PostgreSQL を対象とした操作であること。
- `psql` が PATH にあり、実行ホストから対象データベースへ接続できること。接続情報はアプリケーション設定（`config.NewDatabaseConfig(cfg).DSN()`）から取得されます。
- 実行ユーザーには対象データベースに対する `REINDEX` や `ALTER DATABASE` を行う十分な権限が必要です。

## 注意点 / セーフティ

- `REINDEX DATABASE` および `ALTER DATABASE ...` はデータベースに影響を与える破壊的操作ではありませんが、DB ロックや一時的な性能低下を引き起こす可能性があります。本番環境での実行はメンテナンス時間帯に行ってください。
- コマンドは実際に DB 上で SQL を実行します。ターゲットデータベース名を誤ると意図しない DB に対して操作が行われるため注意してください。
- `workDir` が `/app` に固定されている実装になっているため、ローカル環境で直接実行する場合は適切な作業ディレクトリや `psql` のインストール状況を確認してください。

## トラブルシューティング

- psql の起動に失敗する場合: `psql` が PATH にあるか、バイナリに実行権限があるかを確認してください。
- 接続エラーが出る場合: アプリケーションの設定にある DSN（接続情報）が正しいか、ネットワークや認証情報を確認してください。
- 権限エラーが出る場合: 実行ユーザーに `REINDEX` / `ALTER DATABASE` を行う権限があるか、DB 管理者に確認してください。

## 拡張・カスタマイズ

- PostgreSQL 以外の DB 管理用に流用する場合は、`psqlCommand` を対象 DB の CLI に差し替え、実行する SQL を対象 DB 用に書き換える必要があります（本実装は PostgreSQL 固有の操作を行います）。
