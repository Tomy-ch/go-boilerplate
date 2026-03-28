# ドメイン層（`internal/domain`）ガイド

## オニオンアーキテクチャでの役割

- **ビジネスの中心（核）**。エンティティ、値オブジェクト、ドメインサービス、ドメインイベントなどの **本質的なルール** を表現する。
- 外界（HTTP / DB / UI）への関心は一切持たず、**純粋なモデルと言語** で振る舞いを定義する。
- 変更に最も強い層。**ここが壊れない限りプロダクトは保守できる** という前提で守る。

## このプロジェクトでの役割

- `internal/domain/<bounded-context>/<aggregate>/` 配下に **Entity / ValueObject / DomainService / Repository(IF)** を配置する。

例）`internal/domain/user/`

```text
user_domain.go       ← Aggregate Root (User)
value.go             ← 値オブジェクト
service.go           ← Domain Service
user_repository.go   ← Repository Interface
error.go             ← ドメインエラー
constant.go          ← 検証定数
```

- **副作用を持たない関数（純関数）** でルールを記述するのが原則。  
  I/O・時刻取得・乱数生成などは **引数で注入された値** に依存させる。

- 状態変更は **エンティティのメソッド** で行い、外部リソースへのアクセスはしない。

- 型は **effectively immutable** を基本とする。

  - private field + getter
  - defensive copy（`ptr.Copy`）
  - setterは禁止
  - 状態変更は **振る舞いメソッド**

- 依存関係は **コンストラクタで注入** する。

- 外部ライブラリは直接持ち込まず **pkg wrapper経由** で使用する。

例：

- UUID → `pkg/uuid`
- Decimal → `pkg/decimal`
- Error → `pkg/xerrors`

## ドメインの境界

Domain 層は **ビジネスルールと状態遷移を表現する層**である。

Domain の責務：

- 不変条件（Invariant）
- 状態遷移
- 値の整合性
- ビジネスルール

Domain の責務ではないもの：

- 検索仕様
- DB最適化
- SQL構造
- 外部API呼び出し
- 集計処理

これらは次の層で扱う。

- Usecase
- QueryService
- ReadModel

Repository は **永続化の抽象のみ提供する**。

単純な Query は実務上許容する。

許容例：

- `FindByXXX`
- `FindAll`
- `CountByXXX`

## 実装上の注意点

### 命名/構造

- 構造体名は **ドメイン名**
- スライス型は必要に応じて定義

```go
type Users []*User
```

- Repository インターフェース名は `Repository`
- パッケージ名はドメイン名
- コンストラクタは `New`

### コンストラクタ経由以外でセットしない

- 不変条件は `New(...)` で保証
- setterは禁止
- 状態変更は **振る舞いメソッド**

### 取得は getter 経由

- フィールド公開禁止

```go
ID()
FirstName()
Email()
```

- pointer型は **defensive copy**

```go
ptr.Copy(...)
```

### 構造体にタグを打たない

Domainは外界を知らない。

禁止：

```text
json
db
validate
```

これらは DTO / Infra に置く。

### 時刻・ID の扱い

- `time.Now()` は Domain で使わない
- UUID 生成も Domain で行わない

生成は：

- Controller
- Usecase

Domainは **型付き値のみ受け取る**

```go
uuid.UUID
time.Time
```

### バリデーション

#### 形式チェック

原則 **値オブジェクト**

例：

```go
NewEmail(...)
```

軽量ドメインでは基本型も許容。

#### 境界値チェック

境界値は `constant.go`

```go
minLength
maxEmailLength
```

#### エラー

エラーは **具体エラー**

```go
ErrInvalidEmail
ErrInvalidPostalCode
```

抽象エラーは直接返さない。

```go
if !stringkit.InRange(email, minLength, maxEmailLength) {
    return nil, xerrors.Wrap(ErrInvalidEmail, ...)
}
```

### 不変条件（Domain Invariant）

エンティティは **Invariantを常に満たす**。

例：

- `updatedAt >= createdAt`
- `deletedAt >= createdAt`
- `deletedAt >= updatedAt`

Invariant保証箇所：

- `New(...)`
- 状態変更メソッド

Usecase / Repository は  
**Invariant保証責務を持たない**。

## Aggregate Design

この boilerplate では **Aggregate を設計単位**とする。

```text
internal/domain/<bounded-context>/<aggregate>/
```

### Aggregate Root

Aggregate には **1つの Root** が存在する。

責務：

- 整合性保証
- 外部操作入口
- 永続化対象

```go
type User struct {
    id uuid.UUID
}
```

Repository は **Root に対して定義**

```go
type Repository interface {
    CreateUser(ctx context.Context, user *User) error
}
```

### Aggregate の整合性

変更は **Root経由のみ**

```text
Usecase → Aggregate Root → Entity
```

### Aggregate Boundary

Aggregate は **小さく保つ**

基本原則：

```text
1 Aggregate = 1 Transaction Boundary
```

避ける：

- 巨大Aggregate
- DB構造直写
- 強結合モデル

### Aggregate 間参照

参照は **IDのみ**

```go
type Order struct {
    userID uuid.UUID
}
```

禁止：

```text
Order {
    user *User
}
```

### 複数Aggregateルール

複数Aggregateに跨るルールは

- Domain Service
- Usecase

例：

```text
User退会 → Subscription停止
```

### Query と Aggregate

Aggregate は **Write Model**

以下は扱わない：

- 集計
- レポート
- 複雑検索
- GROUP BY

これらは

- QueryService
- ReadModel

へ配置。

## インフラ層の依存性逆転

Repository は **永続化抽象**

```go
type Repository interface {
    FindAll(ctx context.Context, limit, offset int32) (Users, error)
    CreateUser(ctx context.Context, user *User) error
    CountByActive(ctx context.Context, active*bool) (int64, error)
}
```

実装：

```text
internal/infrastructure/persistence/postgres/
```

`sqlc` でドメインへマッピング。

### Repository に許容するメソッド

- `FindAll`
- `FindByXXX`
- `CountByXXX`

想定：

```text
SELECT / WHERE / JOIN
```

### Repository に持たせないもの

- GROUP BY
- SUM / AVG
- WITH句
- 境界越JOIN

配置先：

- Usecase
- QueryService
- ReadModel

## 呼び出せる層

呼び出し元：

- Usecase

Domainは **他層を呼ばない**

他Aggregateルール：

- Domain Service

例外：

```text
参照専用Aggregate
```

## テスト戦略

Domain テストは **純粋単体テスト**

依存禁止：

- DB
- HTTP
- 環境変数
- time.Now()

### コンストラクタ検証

`New(...)` が **Invariantを保証**

例：

- IDゼロ値
- 境界値
- 時刻整合性

```go
require.ErrorIs(t, err, ErrInvalidEmail)
```

### Getter 契約テスト

```text
ID()
FirstName()
Email()
CreatedAt()
UpdatedAt()
```

### Immutable 保証テスト

pointer型：

```text
*string
*time.Time
```

検証：

1. constructorポインタ変更
2. getter返り値変更

Entity内部は変化しない。

### ドメイン振る舞いテスト

例：

```text
FullName()
```

結果：

```text
firstName + " " + lastName
```

### エラー分類テスト

```go
require.ErrorIs(t, err, ErrInvalidEmail)
```

### テスト設計ポリシー

#### Deterministic

```go
baseTime := time.Date(2025,1,1,0,0,0,0,time.UTC)
```

#### 並列実行

```go
t.Parallel()
```

#### Fail Fast

```go
require.NoError(t, err)
```

### Test Fixture

Fixtureを推奨。

理由：

- 重複削減
- Invariant保証
- テスト簡潔化

```go
func newTestUser(t *testing.T)*User {
    baseTime := time.Date(2025,1,1,0,0,0,0,time.UTC)

    id := uuid.NewTestFromSalt(t,"user")
    prefectureID := uuid.NewTestFromSalt(t,"prefecture")

    user, err := New(
        id,
        "John",
        "Doe",
        "hashed_password",
        "john@example.com",
        "1234567890",
        prefectureID,
        "Shibuya",
        "1-2-3",
        nil,
        "1500001",
        baseTime,
        baseTime.Add(time.Hour),
        nil,
    )

    require.NoError(t, err)
    return user
}
```

### 不変条件保持テスト

状態遷移テスト：

```text
Before → Behavior → After
```

例：

```go
func TestUser_UpdateEmail(t *testing.T) {
    user := newTestUser(t)

    updatedAt := user.UpdatedAt().Add(time.Hour)

    err := user.UpdateEmail("new@example.com", updatedAt)

    require.NoError(t, err)
    require.Equal(t, "new@example.com", user.Email())
}
```

不正ケース：

```go
require.ErrorIs(t, err, ErrInvalidUpdatedAt)
```

## やっていいこと / いけないこと

### Do

- constructorで完全性保証
- 振る舞いメソッドで状態遷移
- VOで整合性担保
- Repository抽象化
- テーブル駆動テスト

### Don’t

禁止：

- http.*
- echo.*
- sqlc型
- jsonタグ
- setter
- DB主導設計
- Domainでtime.Now()

```go
// constant.go
package user

const (
    minLength           = 1
    maxFirstNameLength  = 100
    maxLastNameLength   = 100
    maxPasswordLength   = 255
    maxEmailLength      = 100
    maxPhoneLength      = 20
    maxCityLength       = 100
    maxStreetLength     = 255
    maxBuildingLength   = 255
    maxPostalCodeLength = 8
)
```

```go
// error.go
package user

import (
    "boilerplate-go/internal/apperror"
    "boilerplate-go/pkg/xerrors"
)

var (
    errInvalid             = xerrors.Wrap(apperror.ErrValidation, "invalid user")
    ErrInvalidID           = xerrors.Wrap(errInvalid, "id failed")
    ErrInvalidFirstName    = xerrors.Wrap(errInvalid, "first name failed")
    ErrInvalidLastName     = xerrors.Wrap(errInvalid, "last name failed")
    ErrInvalidPassword     = xerrors.Wrap(errInvalid, "password failed")
    ErrInvalidEmail        = xerrors.Wrap(errInvalid, "email failed")
    ErrInvalidPhone        = xerrors.Wrap(errInvalid, "phone failed")
    ErrInvalidPrefectureID = xerrors.Wrap(errInvalid, "prefecture id failed")
    ErrInvalidCity         = xerrors.Wrap(errInvalid, "city failed")
    ErrInvalidStreet       = xerrors.Wrap(errInvalid, "street failed")
    ErrInvalidBuilding     = xerrors.Wrap(errInvalid, "building failed")
    ErrInvalidPostalCode   = xerrors.Wrap(errInvalid, "postal code failed")
    ErrInvalidUpdatedAt    = xerrors.Wrap(errInvalid, "updated at failed")
    ErrInvalidDeletedAt    = xerrors.Wrap(errInvalid, "deleted at failed")
)
```

```go
// user_domain.go
package user

import (
    "time"

    "boilerplate-go/pkg/ptr"
    "boilerplate-go/pkg/stringkit"
    "boilerplate-go/pkg/uuid"
    "boilerplate-go/pkg/xerrors"
)

type Users []*User

type User struct {
    id           uuid.UUID
    firstName    string
    lastName     string
    passwordHash string
    email        string
    phone        string
    prefectureID uuid.UUID
    city         string
    street       string
    building     *string
    postalCode   string
    createdAt    time.Time
    updatedAt    time.Time
    deletedAt    *time.Time
}

func New(
    id uuid.UUID,
    firstName string,
    lastName string,
    passwordHash string,
    email string,
    phone string,
    prefectureID uuid.UUID,
    city string,
    street string,
    building *string,
    postalCode string,
    createdAt time.Time,
    updatedAt time.Time,
    deletedAt *time.Time,
) (*User, error) {
    if id.IsNil() {
        return nil, xerrors.Wrap(ErrInvalidID, "id is required")
    }

    if !stringkit.InRange(firstName, minLength, maxFirstNameLength) {
        return nil, xerrors.Wrap(ErrInvalidFirstName, stringkit.ErrorMsgInRange(minLength, maxFirstNameLength, firstName))
    }

    if !stringkit.InRange(lastName, minLength, maxLastNameLength) {
        return nil, xerrors.Wrap(ErrInvalidLastName, stringkit.ErrorMsgInRange(minLength, maxLastNameLength, lastName))
    }

    if !stringkit.InRange(passwordHash, minLength, maxPasswordLength) {
        return nil, xerrors.Wrap(ErrInvalidPassword, stringkit.ErrorMsgInRange(minLength, maxPasswordLength, passwordHash))
    }

    if !stringkit.InRange(email, minLength, maxEmailLength) {
        return nil, xerrors.Wrap(ErrInvalidEmail, stringkit.ErrorMsgInRange(minLength, maxEmailLength, email))
    }

    if !stringkit.InRange(phone, minLength, maxPhoneLength) {
        return nil, xerrors.Wrap(ErrInvalidPhone, stringkit.ErrorMsgInRange(minLength, maxPhoneLength, phone))
    }

    if prefectureID.IsNil() {
        return nil, xerrors.Wrap(ErrInvalidPrefectureID, "prefectureID is required")
    }

    if !stringkit.InRange(city, minLength, maxCityLength) {
        return nil, xerrors.Wrap(ErrInvalidCity, stringkit.ErrorMsgInRange(minLength, maxCityLength, city))
    }

    if !stringkit.InRange(street, minLength, maxStreetLength) {
        return nil, xerrors.Wrap(ErrInvalidStreet, stringkit.ErrorMsgInRange(minLength, maxStreetLength, street))
    }

    if building != nil && !stringkit.InRange(*building, minLength, maxBuildingLength) {
        return nil, xerrors.Wrap(ErrInvalidBuilding, stringkit.ErrorMsgInRange(minLength, maxBuildingLength, *building))
    }

    if !stringkit.InRange(postalCode, minLength, maxPostalCodeLength) {
        return nil, xerrors.Wrap(ErrInvalidPostalCode, stringkit.ErrorMsgInRange(minLength, maxPostalCodeLength, postalCode))
    }

    if updatedAt.Before(createdAt) {
        return nil, xerrors.Wrap(ErrInvalidUpdatedAt, "updatedAt must be after or equal to createdAt")
    }

    if deletedAt != nil && deletedAt.Before(createdAt) {
        return nil, xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to createdAt")
    }

    if deletedAt != nil && deletedAt.Before(updatedAt) {
        return nil, xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to updatedAt")
    }

    return &User{
        id:           id,
        firstName:    firstName,
        lastName:     lastName,
        passwordHash: passwordHash,
        email:        email,
        phone:        phone,
        prefectureID: prefectureID,
        city:         city,
        street:       street,
        building:     ptr.Copy(building),
        postalCode:   postalCode,
        createdAt:    createdAt,
        updatedAt:    updatedAt,
        deletedAt:    ptr.Copy(deletedAt),
    }, nil
}

func (u *User) ID() uuid.UUID        { return u.id }
func (u *User) FirstName() string    { return u.firstName }
func (u *User) LastName() string     { return u.lastName }
func (u *User) PasswordHash() string { return u.passwordHash }
func (u *User) Email() string        { return u.email }
func (u *User) Phone() string        { return u.phone }
func (u *User) PrefectureID() uuid.UUID { return u.prefectureID }
func (u *User) City() string         { return u.city }
func (u *User) Street() string       { return u.street }
func (u *User) Building() *string    { return ptr.Copy(u.building) }
func (u *User) PostalCode() string   { return u.postalCode }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }
func (u *User) DeletedAt() *time.Time { return ptr.Copy(u.deletedAt) }

func (u *User) FullName() string {
    return u.firstName + " " + u.lastName
}
```

```go
// user_repository.go
//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package user

import "context"

type Repository interface {
    FindAll(ctx context.Context, limit, offset int32) (Users, error)
    FindByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (Users, error)
    CreateUser(ctx context.Context, user *User) error
    CountByActive(ctx context.Context, active *bool) (int64, error)
}
```
