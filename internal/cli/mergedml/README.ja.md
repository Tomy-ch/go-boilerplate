# merge-dml

[English](README.md) | 日本語

`database/dml/<type>/<category>/` 配下の複数の DML SQL ファイルを、sqlc コード生成用に単一ファイルへ結合します。出力先は `database/gen/<category>_<type>.gen.sql` です。

## コマンド

```
merge-dml --type <type> [flags]
```

## フラグ

| フラグ | デフォルト | 説明 |
|--------|------------|------|
| `--type` | *(必須)* | 対象の DML タイプ（例: `repository`, `query_service`） |
| `--work-dir` | `/app` | 作業ディレクトリ（プロジェクトルート） |

## 使い方

```bash
# リポジトリの DML ファイルをすべて結合
./server merge-dml --type repository

# 作業ディレクトリを指定してクエリサービスの DML ファイルを結合
./server merge-dml --type query_service --work-dir /app
```

## 注意点

- 各カテゴリ内の SQL ファイルはパス順にソートされてから結合されるため、出力は安定します。
- ソースカテゴリに SQL ファイルがなくなった場合、古い生成ファイルは自動的に削除されます。
- 出力パスは `database/gen/` 配下であることが検証され、意図しない場所への書き込みを防止します。
