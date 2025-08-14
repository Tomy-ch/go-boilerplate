# SQLC ベストプラクティス集

このドキュメントでは、`sqlc` のコード生成を最適化するための **独自ディレクティブ** や **擬似関数** の使い方をまとめます。  
これらを活用することで、生成される Go コードの可読性・保守性を大幅に向上できます。

## 1. 基本構文：`-- name:` と実行種別

各 SQL ファイルには、**クエリ名**と**実行種別**をコメントで指定します。

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = sqlc.arg(id);

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (sqlc.arg(name), sqlc.arg(email));
```

| 実行種別 | 説明 |
|----------|------|
| `:one`       | 単一レコードを返す |
| `:many`      | 複数レコードを返す |
| `:exec`      | 結果を返さない（INSERT, UPDATE, DELETE） |
| `:execrows`  | `RowsAffected` を返す |
| `:batch`     | 複数クエリをバッチ実行 |

## 2. `sqlc.arg()` — 引数名を明示する

パラメータ名を明示的に指定することで、生成される構造体のフィールド名を制御できます。

```sql
WHERE age > sqlc.arg(min_age)
```

生成される Go コード例：

```go
type GetUsersParams struct {
    MinAge int
}
```

一部のパラメータでは、nullableを許容するために`sql.carg()`を利用しても、`sql.NullXxxx`が生成されます。

```sql
LIMIT sqlc.arg(limit_param)
OFFSET sqlc.arg(offset_param)
```

## 3. `sqlc.embed()` — 構造体埋め込み

JOIN 結果をネスト構造で返したい場合に使用します。

```sql
-- name: GetUserWithProfile :one
SELECT sqlc.embed(u), sqlc.embed(p)
FROM users u
JOIN profiles p ON p.user_id = u.id
WHERE u.id = sqlc.arg(id);
```

生成される Go コード例：

```go
type GetUserWithProfileRow struct {
    User    User
    Profile Profile
}
```

## 4. `sqlc.narg()` — NULL 許容パラメータ

NULL を許容するパラメータを指定します。

```sql
WHERE deleted_at IS sqlc.narg(deleted_at)
```

生成される Go コード例：

```go
type GetUsersParams struct {
    DeletedAt sql.NullTime
}
```

## 5. 型キャストで型推論を補強

PostgreSQL の型を明示することで、Go 側の型も強制できます。

```sql
WHERE id = sqlc.arg(user_id)::uuid
```

生成される Go コード例（`pgx` 使用時など）：

```go
type GetUserParams struct {
    UserID uuid.UUID
}
```

## 6. 生成ファイル側で生成型をオーバーライドする

`database/sqlc/sqlc.template.yaml`で生成の型を上書きできます。

```yaml
version: "2"
sql:
  - engine: postgresql
    # ...
    gen:
      go:
        package: gen
        out: internal/infrastructure/rdb/...
        overrides:
          - db_type: "pg_catalog.int4"
            go_type: "int"
```

## 7. 配列パラメータ

配列を渡す場合は `ANY()` と型キャストを併用します。

```sql
WHERE id = ANY(sqlc.arg(user_ids)::uuid[])
```

生成される Go コード例：

```go
type GetUsersParams struct {
    UserIDs []uuid.UUID
}
```

## 8. 戻り値フィールド名を意識した SELECT

カラム名に応じて Go 側のフィールド名が決まります。

```sql
-- name: GetUserEmailAndName :one
SELECT email, name FROM users WHERE id = sqlc.arg(id);
```

生成される Go コード例：

```go
type GetUserEmailAndNameRow struct {
    Email string
    Name  string
}
```

## 9. 複雑クエリはサブクエリ / CTE で可読性確保

```sql
-- name: SearchUsers :many
SELECT * FROM (
    SELECT id, name FROM users
    WHERE name ILIKE '%' || sqlc.arg(keyword) || '%'
) sub
ORDER BY name;
```

## 推奨運用フロー

1. 必須: 各クエリに`-- name:`と種別を付与
2. 必須: パラメータは必ず`sqlc.arg()`で命名
3. 必須: NULL許容は`sqlc.narg()`を使う
4. 任意: JOINは`sqlc.embed()`を活用
5. 任意: 型は必要に応じてCASTで強制
6. 任意: 配列やIN句は`ANY()`と併用
7. 任意: 長いクエリはCTEやサブクエリで整理

## 参考リンク

- [sqlc 公式ドキュメント](https://docs.sqlc.dev/en/latest/)
- [PostgreSQL 型一覧](https://www.postgresql.org/docs/current/datatype.html)
- [Go Database/SQL パッケージ](https://pkg.go.dev/database/sql)
