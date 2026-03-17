# pgerror パッケージ

[English](README.md) | 日本語

概要: **PostgreSQL 固有のエラーをアプリケーション共通エラーへ正規化し、接続エラーを判定するための Infrastructure レイヤーコンポーネント。DB 固有のエラー仕様を上位レイヤーから隠蔽するための変換レイヤーです。**

## アーキテクチャ上の位置

```txt
Repository
   ↓
pgerror
   ↓
PostgreSQL driver (pgx / pgconn)
```

pgerror は **DB driver が返すエラーをアプリケーション共通エラーへ変換するレイヤー**です。

このレイヤーを挟むことで、

- Usecase
- Domain
- Handler

などの上位レイヤーが **PostgreSQL 固有のエラー仕様を意識する必要がなくなります。**

## 役割

このパッケージは PostgreSQL 固有のエラー処理をアプリケーション共通形式へ正規化します。

主な責務:

- PostgreSQL SQLSTATE を `apperror` へ変換
- DB 接続不可エラーの判定
- `sql.ErrNoRows` を `NotFound` へ変換
- Infrastructure → Usecase 間のエラー仕様を統一

これにより **DB 実装依存のエラー判定をアプリケーションコードから分離**できます。

## エラー正規化

`NormalizeError` は PostgreSQL エラーをアプリケーション共通エラーへ変換します。

```go
func NormalizeError(err error) error
```

処理の流れ:

```txt
DB error
   ↓
NormalizeError
   ↓
AppError (apperror)
```

この関数は **Infrastructure 層から Usecase 層へ返すエラーの唯一の正規化ポイント**として使用することが推奨されます。

## SQLSTATE マッピング

以下の PostgreSQL SQLSTATE がアプリケーションエラーへ変換されます。

|SQLSTATE|意味|AppError|
|--------|------|----------|
|23505|unique violation|Conflict|
|23503|foreign key violation|InvalidArgument|
|23502|not null violation|InvalidArgument|
|23514|check violation|InvalidArgument|
|22001|string too long|InvalidArgument|
|22P02|invalid text representation|InvalidArgument|
|42501|insufficient privilege|PermissionDenied|

これらに該当しない PostgreSQL エラーは

```txt
Internal
```

として扱われます。

## 特別処理

### sql.ErrNoRows

`sql.ErrNoRows` は PostgreSQL SQLSTATE ではないため特別扱いされます。

```txt
sql.ErrNoRows
   ↓
NotFound
```

これにより Repository 層は

```go
return NormalizeError(err)
```

のみで `NotFound` エラーを扱えます。

## 接続不可エラー判定

`IsUnavailable` はデータベース接続不可エラーを判定します。

```go
func IsUnavailable(err error) bool
```

以下のエラーが接続不可として扱われます。

```txt
context.DeadlineExceeded
net.Error (timeout)
driver.ErrBadConn
PostgreSQL SQLSTATE 08XXX
```

この判定は

- リトライ
- サーキットブレーカー
- フェイルオーバー

などの復旧処理に利用できます。

## エラーラッピング

pgerror は元の DB エラーを保持したまま

```txt
apperror
+ 
original error message
```

の形で `xerrors.Wrap` によりラップします。

これにより

- アプリケーションエラー種別
- 元の DB エラー

の両方を保持できます。

## 必要度

### 本番運用

必須

理由:

- DB 制約違反を正しい HTTP ステータスへ変換
- DB 接続断の検知
- セキュリティ上の raw error 隠蔽

を行うため、アプリケーションとして必須のレイヤーです。

### 開発 / テスト

必須

理由:

- DB エラー仕様をテストから隠蔽
- CI でのエラーハンドリング安定化
- sqlc / repository テストの可読性向上

## 注意点

### NormalizeError を一箇所で使用する

`NormalizeError` は **Infrastructure → Usecase の境界で一度だけ適用する**ことが推奨されます。

複数箇所で変換するとエラー構造が崩れる可能性があります。

### PostgreSQL 固有仕様

SQLSTATE `08XXX` は PostgreSQL 固有の接続エラーです。

他 DB (MySQL / TiDB など) に移行する場合は

```txt
pgerror
```

の実装を差し替える必要があります。

### 上位レイヤーで DB 判定をしない

Usecase / Domain 層で

```txt
SQLSTATE
pgconn
pgx
```

などの DB 依存ロジックを書かないようにしてください。
