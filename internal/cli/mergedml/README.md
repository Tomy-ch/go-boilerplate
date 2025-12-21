# merge dml コマンド

概要: `merge-dml` コマンドは、プロジェクト内の DML ディレクトリ
(`database/dml/<type>/<category>/...`) 配下にある複数の `.sql` ファイルを
カテゴリ単位で結合し、SQLC 等のコード生成に渡すための単一 SQL ファイル
（`database/gen/<category>_<type>.gen.sql`）を生成する CLI ユーティリティです。

## 役割

- DML リポジトリ（`repository`）やクエリサービス（`query_service`）といったタイプごとに、カテゴリ配下の `.sql` ファイルを走査して連結し、1 ファイルにまとめます。
- 既存の生成物で今回不要になったファイルは削除（stale cleanup）します。

## 実装の要点

- コマンド実体: Cobra コマンド `merge-dml`。
- ワークディレクトリ: ジェネレーターはデフォルトで `workDir = "/app"` を想定。
- DML のルート: デフォルト `database/dml/`（この下に `repository` / `query_service` 等がある想定）。
- 生成先: `database/gen/` 配下に `<category>_<type>.gen.sql` というファイル名で出力。
- スキャン方法: `filepath.WalkDir` で対象ディレクトリ配下の `.sql` ファイルを収集し、ソートして安定化した順序で連結します（差分がブレにくくするため）。
- 出力: 各ファイルの先頭に `-- === source: <path> ===` のヘッダを追加して、どこから来たか分かるようにしています（SQL コメントなので無害）。
- 出力の安全性: `ensureUnderDir` で出力先パスが `database/gen/` 配下であることを検証します。
- 空カテゴリ（カテゴリは存在するが `.sql` が無い）については、該当する生成ファイルを削除します。
- stale cleanup: 生成が不要になった `<category>_<type>.gen.sql` を削除します。削除対象は、指定した `type` のサフィックス（`_<type>.gen.sql`）を持つファイルに限定します。

### 並列処理とチューニング

- SQLC のデータベース introspection / 生成処理の並列度に関する定数を持ちます。
  - `sqlcDBConcurrency`（デフォルト 4）: sqlc 生成ジョブの並列数。
  - `maxSQLCConcurrency`（デフォルト 4）: 占有してよい並列の上限（CPU の占有を制限）。
  - `minSQLCConcurrency`（デフォルト 2）: 並列の下限（I/O待ちを考慮）。
- `resolveConcurrencyConst()` で `runtime.NumCPU()` と定数を組み合わせて最終的な同時実行数を決定します。

## 使い方

- コマンド（アプリケーションに組み込まれた CLI として提供されます）
  - `merge-dml --type repository`
  - `merge-dml --type query_service --work-dir /app`
- フラグ:
  - `--type` : 対象のタイプ（必須） — 例: `repository` または `query_service`。
  - `--work-dir` : 作業ディレクトリのパス（省略時は `/app`）。

### 実行例

- ルート（アプリケーションコンテナ内など）

```bash
# カテゴリごとに database/gen/<category>_repository.gen.sql が生成される
./your-app merge-dml --type repository
```

## 前提 / 要件

- 実行環境は CI やコンテナなど、`workDir`（デフォルト `/app`） がプロジェクトルートを指すことを期待しています。
- `database/dml/<type>/` 配下にカテゴリごとのサブディレクトリが存在すること。
- 実行ユーザーに対するファイルシステムの読み書き権限（`database/gen/` への書き込み・削除権限）。

## 注意点 / セーフティ

- ファイル削除挙動: stale cleanup は `database/gen/` 配下でかつ指定した `type` のサフィックスを持つファイルのみを対象とします（誤削除を避けるためのフィルタリング）。
- 出力先検証: `ensureUnderDir` で出力パスが `database/gen/` 配下であることを検証し、外部パスへは書き出しません。
- マージ順: ファイルはパスでソートしてから連結します。これにより生成差分が安定します。
- SQL ファイルの結合時に、ファイル末尾に改行が無いケースでも壊れないように改行を補完します。

## トラブルシューティング

- `no categories found under dml directory` が出る場合:
  - `database/dml/<type>/` にカテゴリディレクトリが無いことを示します。
  - この場合コマンドは `database/gen/` 配下の当該タイプの生成物を削除（cleanup）し、終了します。
- ファイルアクセス権のエラー:
  - 実行ユーザーに `database/dml/` の読み取り、`database/gen/` の書き込み・削除権限があるか確認してください。
- 出力ファイルに予期しないパスが含まれる場合:
  - `ensureUnderDir` の検証に失敗しているかもしれません。設定された `workDir` と `genRootDir` の値を確認してください。

## 拡張・カスタマイズ

- `workDir`, `dmlRootDir`, `genRootDir`, `sqlcCfg` は `generator` のフィールドとして設定されています。テストや特殊な実行環境向けにこれらを上書きできます。
- さらに高度なフィルタリングやテンプレート（ヘッダ、フッタ）を付与したい場合は、`buildCategorySQLFile` の連結処理に前処理/後処理のフックを追加してください。
