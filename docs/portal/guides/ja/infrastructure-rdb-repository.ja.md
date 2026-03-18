# Repository 実装ガイド

[English](README.md) | 日本語

## 役割

Repository は **Domain の永続化抽象（Repository Interface）を Infrastructure で実装する層**です。

この層の責務は次の 3 つに限定されます。

1. sqlc を利用したクエリ実行
2. DB Row → Domain エンティティ変換
3. DB エラーの正規化

Repository は **ビジネスロジックを持ちません**。

```txt
Usecase
   ↓
Repository (Infra)
   ↓
sqlc
   ↓
Database
```

Repository は Domain が定義する Repository Interface を **実装するだけ**の層です。

## アーキテクチャ上の位置

Repository 実装は次の場所に配置します。

```txt
internal/infrastructure/rdb/repository/<aggregate>/
```

例

```txt
repository/
 ├ user/
 │   └ user_repository.go
 └ prefecture/
     └ prefecture_repository.go
```

Repository Interface は **Domain 層に配置**されます。

```txt
internal/domain/<aggregate>/repository.go
```

Infra はこのインターフェースを **実装するのみ**です。

## Repository メソッドの責務

Repository メソッドは次の処理だけを行います。

```txt
Query
 ↓
sqlc
 ↓
Row
 ↓
Domain Entity
 ↓
return
```

Repository は次を行いません。

- ビジネスルール
- 集計処理
- DTO生成
- Usecaseロジック

これらは **Usecase / Domain の責務**です。

## sqlc の利用

Repository は **sqlc が生成したクエリコード**を利用します。

```go
rows, err := db.ListUsers(ctx, &gen.ListUsersParams{...})
```

sqlc により

- 型安全な SQL 実行
- コンパイル時検証

が可能になります。

sqlc の生成コードは次の場所にあります。

```txt
internal/infrastructure/rdb/sqlc/gen
```

## Row → Domain 変換

sqlc が返す Row 構造体は **Infrastructure 専用型**です。

そのため Repository は必ず **Domain エンティティへ変換**します。

```go
user, err := user.New(
    uuid.FromPrimitive(row.Users.ID),
    row.Users.FirstName,
    row.Users.LastName,
    ...
)
```

重要ルール

- `sqlc` Row を上位層に返さない
- Domain constructor を利用する

## Domain constructor error

Domain エンティティ生成に失敗した場合、
そのエラーは **そのまま返却します。**

```go
user, err := user.New(...)
if err != nil {
    return nil, err
}
```

理由

- Domain invariant violation
- DBデータ不整合
- migrationミス

これらは **Domainレイヤーの責務として扱う**ためです。

## UUID 変換

DB は primitive UUID を使用します。

Domain では `pkg/uuid.UUID` を使用します。

そのため変換が必要になります。

```go
uuid.FromPrimitive(row.ID)
uuid.ToPrimitiveUniqueList(ids)
```

UUID 操作は

```txt
pkg/uuid
```

のラッパーを利用します。

## nullable 変換

DB の nullable 値は `sql.Null*` 型で表現されます。

Repository では

```txt
internal/infrastructure/rdb/conv
```

のユーティリティを使用して変換します。

例

```go
conv.StringPtrFromNull(row.Users.Building)
conv.NullStringFromPtr(user.Building())
```

これにより

```txt
sql.NullString ⇔ *string
sql.NullTime   ⇔ *time.Time
```

の変換を一元化できます。

## sqlc helper

LIKE検索などの補助処理は

```txt
internal/infrastructure/rdb/sqlc
```

の helper を使用します。

例

```go
escaped := sqlc.EscapeForLike(keyword, sqlc.DefaultLikeEscapeChar)
pattern := sqlc.WrapContainsLikePattern(escaped)
```

削除状態

```go
DeletedState: sqlc.BoolPtrToDeletedState(active)
```

目的

- LIKEインジェクション防止
- 検索パターン統一
- 状態変換の一元化

## LoggingDBProvider

Repository は直接 `driver.DatabaseDriver` を使わず

```go
db := gen.New(r.db.NewLoggingDB(ctx))
```

で sqlc Querier を生成します。

`loggingdb.DBProvider` は次を提供します。

- SQL ログ出力
- DB / Tx の透過切り替え
- Contextベース接続取得

Repository は **DB接続状態を意識しない設計**になります。

## エラー正規化

PostgreSQL エラーは

```txt
internal/infrastructure/rdb/postgres/pgerror
```

で `apperror` に変換します。

```go
return pgerror.NormalizeError(err)
```

主な変換

```txt
sql.ErrNoRows      → ErrNotFound
unique violation   → ErrConflict
connection error   → ErrUnavailable
others             → ErrInternal
```

## トランザクション

トランザクション境界は **Usecase 層**が管理します。

Repository は Tx を開始しません。

クエリ実行は

```go
gen.New(r.db.NewLoggingDB(ctx))
```

を利用して行います。

これにより

```txt
Tx / DB
```

を透過的に切り替えます。

## Observability（Tracing）

Infrastructure 層では

```txt
observability.LayerTracer
```

を利用します。

```go
ctx, endSpan := r.tracer.Start(ctx)
defer endSpan()
```

Repository は

- span開始
- span終了

のみを知ります。

OpenTelemetry SDK には直接依存しません。

## Repository構造体

Repository 実装は次の依存を持ちます。

- driver.DatabaseDriver は、ロギング機能を持たない純粋な DB 接続ドライバです。
- loggingdb.DBProvider は、ロギング機能を持つ DB 接続プロバイダです。

```go
type repository struct {
    db     loggingdb.DBProvider
    tracer observability.LayerTracer
}
```

constructor

```go
func New(
    db loggingdb.DBProvider,
    tf observability.TracerFactory,
) user.Repository {
    return &repository{
        db:       db,
        tracer:   tf.Infra(),
    }
}
```

## テスト戦略

Repository のテストは **Infrastructure Integration Test** として実装します。

Repository は SQL 実行の正しさも責務に含むため、
**mock を使用せず実際の DB を利用して検証します。**

テスト対象の構造は次の通りです。

```txt
Repository
   ↓
sqlc
   ↓
driver
   ↓
PostgreSQL
```

このレイヤー全体を **実 DB でテスト**します。

### テストの目的

Repository テストでは次を検証します。

- SQL クエリが正しく実行される
- Row → Domain 変換が正しい
- Domain constructor エラーが正しく伝搬される
- PostgreSQL エラーが `pgerror.NormalizeError` により正規化される
- Pagination（limit / offset）が正しく機能する

Repository テストは **Domain ロジックの検証を目的としません**。

### テスト実行環境

Repository テストは `testkit` を使用して DB を初期化します。

```go
 db, provider := testkit.NewTestDBWithLoggingProvider(t)
```

この関数は次を提供します。

- テスト用 DB 接続
- loggingdb.DBProvider

### トランザクションテスト

各テストは **トランザクション内で実行されます**。

```go
 txm := testkit.NewTestTransactionManager(t)

 txm.WithinTx(func(ctx context.Context) {
     // test logic
 })
```

内部動作

```txt
BEGIN
 ↓
test
 ↓
ROLLBACK
```

これにより

- DB 状態が汚れない
- テストの独立性が保たれる

### 並列実行

Repository テストでは `t.Parallel()` を使用してテスト自体は並列実行できます。

ただし、`testkit.NewTestTransactionManager(t)` が提供するトランザクションマネージャは
内部でトランザクション実行を **直列化**します。

そのため実行モデルは次のようになります。

```txt
テスト実行        → 並列
トランザクション  → 直列
```

各テストは `WithinTx` 内で

```txt
BEGIN
 ↓
test
 ↓
ROLLBACK
```

の形で実行されます。

トランザクションを直列化することで、複数テストが同時に走っても
DB状態の競合やテスト間の干渉が発生しないようにしています。

### Domain エラーの検証

Repository は Domain constructor を利用して
Row → Domain 変換を行います。

そのため DB 内に **不正データが存在する場合**、
Domain エラーが発生します。

テストではこれも検証します。

```go
require.ErrorIs(t, err, user.ErrInvalidLastName)
```

これは次のケースを検証します。

```txt
DBデータ不整合
migrationミス
Domain invariant violation
```

### テストのエラー正規化

DB エラーは `pgerror.NormalizeError` により
`apperror` に変換されます。

例

```txt
sql.ErrNoRows      → ErrNotFound
unique violation   → ErrConflict
connection error   → ErrUnavailable
others             → ErrInternal
```

Repository テストでは

```txt
ErrConflict
ErrNotFound
```

などの **正規化結果**を検証します。

## Repository Anti-Patterns

Repository 層では **よくある誤った実装パターン**があります。  
これらはアーキテクチャ境界を壊す原因になるため **実装してはいけません。**

### 1. ビジネスロジックを書く

Repository は **永続化層**です。  
ビジネスルールを書いてはいけません。

NG例

```go
func (r *repository) Create(ctx context.Context, user *user.User) error {
    if user.IsPremium() {
        // ❌ ビジネスルール
    }
}
```

正しい責務

```txt
Repository
 ├ Query 実行
 ├ Row → Domain 変換
 └ エラー正規化
```

ビジネスルールは **Domain / Usecase 層の責務**です。

### 2. DTO を生成する

Repository は **DTO を生成しません。**

NG例

```go
return UserDTO{
    Name: row.Users.Name,
}
```

Repository は **Domain エンティティのみ返却**します。

```go
return user.New(...)
```

DTO 変換は **Usecase / Presenter 層の責務**です。

### 3. sqlc Row をそのまま返す

sqlc の Row 型は **Infrastructure 専用型**です。

NG例

```go
return rows
```

必ず Domain へ変換します。

```go
u, err := user.New(...)
```

理由

```txt
sqlc 型を上位層に漏らさない
```

### 4. QueryService を書く

Repository は **集約単位の永続化抽象**です。

そのため

```txt
FindByKeyword
SearchUser
AggregateSearch
```

などの **検索専用 API を実装してはいけません。**

検索は

```txt
QueryService
```

として別レイヤーに分離します。

### 5. トランザクションを開始する

Repository は **トランザクション境界を管理しません。**

NG例

```go
tx, _ := db.Begin()
```

トランザクション管理は

```txt
Usecase
```

の責務です。

Repository は

```go
gen.New(r.db.NewLoggingDB(ctx))
```

を使用して

```txt
Tx / DB
```

を透過的に利用します。

### 6. Controller 型を参照する

Repository は **HTTP 層に依存しません。**

NG例

```go
func (r *repository) Create(ctx echo.Context)
```

Repository は **純粋な Go インターフェース**で実装します。

```go
func (r *repository) Create(ctx context.Context, user *user.User)
```

### 7. Domain interface を Infra に定義する

Repository Interface は **Domain 層に定義します。**

NG例

```txt
internal/infrastructure/repository/user_repository_interface.go
```

正しい配置

```txt
internal/domain/user/repository.go
```

Infra は **Domain Interface の実装のみ**を行います。

## Do / Don't

### Do

- sqlc 生成コードを使用
- Row → Domain 変換
- nullable 変換は conv を利用
- pgerror.NormalizeError を利用
- LIKE helper を利用

### Don't

- Domain interface を Infra に定義
- sqlc 型を上位に返す
- ビジネスロジックを書く
- Controller 型を参照
- QueryService を書く

## 実装例

```go
package user

// repositoryで名称固定
type repository struct {
    db     loggingdb.DBProvider
    tracer observability.LayerTracer
}

// Newで名称固定
func New(
    db loggingdb.DBProvider,
    tf observability.TracerFactory,
) user.Repository {
    return &repository{
        db:       db,
        tracer:   tf.Infra(),
    }
}

func (r *repository) FindAll(ctx context.Context, limit, offset int32) (user.Users, error) {
    // Spanの開始・終了呼び出して設定
    ctx, endSpan := r.tracer.Start(ctx)
    defer endSpan()

    // driver.ResolveDriverWithLogを使うことでログを自動で出力
    // 不要な場合は、driver.ResolveDriver(ctx, r.db)を使う
    db := gen.New(r.db.NewLoggingDB(ctx))

    // genで生成されたDMLの呼び出し
    rows, err := db.ListUsers(ctx, &gen.ListUsersParams{
        OffsetParam: offset,
        LimitParam:  limit,
    })

    if err != nil {
        // エラー正規化して返す。
        // エラー内容はpgerrorパッケージ（internal/infrastructure/rdb/postgres/pgerror）で判定される
        return nil, pgerror.NormalizeError(err)
    }


    // Domainエンティティへの詰め替え
    users := make(user.Users, len(rows))
    for i, row := range rows {
        u, err := user.New(
            uuid.FromPrimitive(row.Users.ID),
            row.Users.FirstName,
            row.Users.LastName,
            row.Users.PasswordHash,
            row.Users.Email,
            row.Users.Phone,
            uuid.FromPrimitive(row.Users.PrefectureID),
            row.Users.City,
            row.Users.Street,
            conv.StringPtrFromNull(row.Users.Building),
            row.Users.PostalCode,
            row.Users.CreatedAt,
            row.Users.UpdatedAt,
            conv.TimePtrFromNull(row.Users.DeletedAt),
        )
        if err != nil {
            return nil, err
        }
        users[i] = u
    }
    return users, nil
}
```
