# インフラ層（`internal/infrastructure`）ガイド

## オニオンアーキテクチャでの役割

- **外部技術（DB・外部API・メッセージング等）へのアクセス実装**を担う層。
- **Domain が定義した Repository インターフェース**を満たす具体実装を提供する（依存関係の逆転：Domainは抽象のみ、Infraが具体）。
- I/O・接続・ドライバ・リトライ・ロギングなど**技術的な詳細**をここに閉じ込め、上位層に漏らさない。

## この boilerplate での役割

- RDB 実装は `internal/infrastructure/rdb/...` に配置。PostgreSQL を想定。
- `sqlc` の生成物を利用し、**Repository 実装**で
  - クエリ呼び出し
  - `sql.Null*` などの**nullable変換**
  - **Domainエンティティへの詰め替え**
  を行う。
- ドライバ解決とSQLのロガー（`zap`）はここで利用・注入。

## sqlc のラッパーを利用

- **生成クエリ**を安全に呼び出すため、`sqlc`のsqlcの生成ファイルを使用。
- 取得結果は **infra専用のRow型** → **Domainエンティティ**へマッピング。
- **nullable変換**は `internal/infrastructure/rdb/conv` のユーティリティを使用し、
  `sql.NullString ⇔ *string` / `sql.NullTime ⇔ *time.Time` 変換を一元化。

## 実装する上での注意点

### 命名/構造

- インターフェイスの実装構造体は`repository`とする
- インスタンスの生成関数名は `New` で統一し、[di/infrastructure.go](../di/repository.go) で登録する。

### 依存方向

- Repository の **インターフェースは Domain 層**（`internal/domain/<agg>/repository.go`）に置く。
- Infra はそのインターフェースを**実装するだけ**。

### 変換の責務

- `sqlc`生成Row/Modelを**上位へ返さない**。必ずDomainへ詰め替えて返す。
- `sql.ErrNoRows` 等は **`apperror.ErrNotFound` に正規化**して返す。
- ユニーク制約違反は **`apperror.ErrConflict` に正規化**して返す。
- その他のエラーは **`apperror.ErrInternal` に正規化**して返す。

### トランザクション

- Tx 境界はUsecase側で管理。Infraは`*sql.DB`/`*sql.Tx`の両方に対応できるようにする。

### ロギング

- 機微情報（パスワード等）のログ出力は禁止。クエリバインド値は極力マスク。

### エラー正規化

- ドライバ固有のエラーを `apperror` にマップ（NotFound/Conflict/Unavailable 等）。

## 呼び出せる層

- **Usecase→Infra（Repository実装）** のみ。
- ControllerやDomainから直接Infra実装を呼ばない。
- DI（`fx`）で **Repository実装**をProvideし、Usecaseへ注入する。

## やっていいこと / いけないこと(まとめ)

### Do

- `sqlc`生成コードでクエリ発行し、**Domain エンティティへ変換**して返す。
- `conv/nullable`で **nullable ⇔ ポインタ**変換を一元化。
- `ErrNoRows` → `ErrNotFound`、ユニーク制約 → `ErrConflict` 等に正規化。
- テストでRepositoryの変換とエラー正規化を検証。

### Don’t

- DomainのインターフェースをInfraで定義する。
- `sqlc`生成型や`sql.Null*`を上位層に返す。
- ビジネスロジックやドメインルールの判定を持ち込む。
- Controller/OpenAPI型を参照する。

## 最小スニペット（雛形）

```go
// 唯一性のある名称
package user

// repositoryで名称固定
type repository struct {
    db *sql.DB
    z  *zap.Logger
}

// レコードが見つからない場合のエラー
var ErrUserNotFound = xerrors.Wrap(apperror.ErrNotFound, "user not found")

// Newで名称固定
func New(db *sql.DB, z *zap.Logger) user.Repository {
    return &repository{
        db: db,
        z:  z,
    }
}


func (r *repository) GetAllUsers(ctx context.Context, limit, offset int) (user.Entities, error) {
    // rdbdriver.ResolveDriverWithLogを使うことでログを自動で出力
    // 不要な場合は、rdbdriver.ResolveDriver(ctx, r.db)を使う
    db := gen.New(rdbdriver.ResolveDriverWithLog(ctx, r.db, r.z))
    // genで生成されたDMLの呼び出し
    rows, err := db.GetUsersDomain(ctx, gen.GetUsersDomainParams{
        OffsetParam: conv.NewNullInt64(int64(offset)),
        LimitParam:  conv.NewNullInt64(int64(limit)),
    })
    if err != nil {
        // レコードが見つからない場合のエラー
        // 本来は0件でもnilを返す設計が正しいが、例として記載
        if xerrors.Is(err, sql.ErrNoRows) {
            return nil, xerrors.Wrap(ErrUserNotFound, err.Error())
        }
        return nil, err
    }

    // Domainエンティティへの詰め替え
    users := make(user.Entities, len(rows))
    for i, row := range rows {
        user, err := user.New(
            row.ID.String(),
            row.FirstName,
            row.LastName,
            row.PasswordHash,
            row.Email,
            row.Phone,
            row.PrefectureID.String(),
            row.PrefectureName,
            row.City,
            row.Street,
            conv.StringPtrFromNull(row.Building),
            row.PostalCode,
            conv.TimePtrFromNull(row.DeletedAt),
        )
        if err != nil {
            return nil, err
        }
        users[i] = *user
    }
    return users, nil
}

```
