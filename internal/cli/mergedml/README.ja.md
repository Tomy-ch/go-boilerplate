# merge-dml

[English](README.md) | 日本語

`database/dml/<type>/<category>/` 配下の複数の DML SQL ファイルを、sqlc コード生成用に単一ファイルへ結合します。出力先は `database/gen/<category>_<type>.gen.sql` です。

## コマンド

```text
merge-dml --type <type> [flags]
```

## フラグ

|フラグ|デフォルト|説明|
|---|---|---|
|`--type`|*(必須)*|対象の DML タイプ。`repository` / `query_service` / `system_cqrs` / `command_service` のいずれか|
|`--work-dir`|`/app`|作業ディレクトリ（プロジェクトルート）|

## 使い方

```bash
# リポジトリの DML ファイルをすべて結合
./server merge-dml --type repository

# 作業ディレクトリを指定してクエリサービスの DML ファイルを結合
./server merge-dml --type query_service --work-dir /app
```

## 注意点

- カテゴリ単位で並列に結合します。並列数は `runtime.NumCPU()` を `[2, 4]` の範囲にクランプした値です（下限 `2` は I/O 待ちの多い処理が直列化するのを防ぎ、上限 `4` は Docker/CI 内で CPU を占有し過ぎないようにするため）。
- 各カテゴリ内の SQL ファイルはパス順にソートされてから結合されるため、出力は安定します。結合時には各ソースファイルの内容の前に `-- === source: <path> ===` の見出しコメントが挿入され、由来を追跡できます。
- ソースカテゴリに SQL ファイルがなくなった場合、古い生成ファイルは自動的に削除されます。
- 出力パスは `database/gen/` 配下であることが検証され、意図しない場所への書き込みを防止します。
