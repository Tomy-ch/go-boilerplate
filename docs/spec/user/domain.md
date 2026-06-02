# User — Domain Spec

> 既存実装（`internal/domain/user`）を spec 化したもの。手書き実装から逆生成した現状仕様であり、未実装機能（更新 / 論理削除など）は含まない。

## Overview

ユーザー集約は、アカウントの基本情報（氏名・認証情報・連絡先）と住所情報（都道府県・市区町村・番地・建物・郵便番号）を保持するドメインの中核エンティティ。生成時に全フィールドの形式・長さ・必須を検証し、不変条件を満たさない `User` は構築できない。

都道府県は ID 参照（`prefectureID`）のみを保持し、表示名は別集約（`prefecture`）から解決する。パスワードは平文を保持せず、検証済みのハッシュ（`passwordHash`）のみを保持する。平文パスワードは値オブジェクト `RawPassword` で長さ検証されたうえで、外側のレイヤーでハッシュ化される。

## Entity

```yaml
package: internal/domain/user
struct: User
fields:
  - name: id
    type: uuid.UUID
    required: true        # IsNil の場合は ErrInvalidID
  - name: firstName
    type: string
    required: true
    min_length: 1
    max_length: 100
  - name: lastName
    type: string
    required: true
    min_length: 1
    max_length: 100
  - name: passwordHash
    type: string
    required: true
    min_length: 1
    max_length: 255
  - name: email
    type: string
    required: true
    min_length: 1
    max_length: 100
  - name: phone
    type: string
    required: true
    min_length: 1
    max_length: 20
  - name: prefectureID
    type: uuid.UUID
    required: true        # IsNil の場合は ErrInvalidPrefectureID
  - name: city
    type: string
    required: true
    min_length: 1
    max_length: 100
  - name: street
    type: string
    required: true
    min_length: 1
    max_length: 255
  - name: building
    type: "*string"
    required: false       # nil 許容。非 nil の場合のみ長さ検証
    min_length: 1
    max_length: 255
  - name: postalCode
    type: string
    required: true
    min_length: 1
    max_length: 8
  - name: createdAt
    type: time.Time
    required: true
  - name: updatedAt
    type: time.Time
    required: true
  - name: deletedAt
    type: "*time.Time"
    required: false       # nil 許容（未削除を表す）
```

## Cross-field Invariants

- `updatedAt >= createdAt`（更新日時は作成日時以降）。違反時 `ErrInvalidUpdatedAt`
- `deletedAt != nil` のとき `deletedAt >= createdAt`。違反時 `ErrInvalidDeletedAt`
- `deletedAt != nil` のとき `deletedAt >= updatedAt`。違反時 `ErrInvalidDeletedAt`

## Behavior Methods

```yaml
# 現状、状態遷移メソッドは未実装（getter のみ）。
# 以下は単純フィールド getter ではない派生メソッド。
- name: FullName
  signature: FullName() string
  description: firstName + " " + lastName を連結したフルネームを返す派生値。
```

## Value Objects

```yaml
- name: RawPassword
  underlying_type: string
  validation: 長さが MinRawPasswordLength(8) 以上 MaxRawPasswordLength(64) 以下。違反時 ErrInvalidRawPassword
  factory: NewRawPassword
  methods:
    - name: Value
      returns: string
```

## Repository Methods

```yaml
- name: FindByActive
  signature: FindByActive(ctx context.Context, active *bool, limit, offset int32) (Users, error)
  behavior: |
    アクティブ状態（active）に基づきユーザー一覧をページング付き（limit / offset）で取得する。
    active=nil は全件、true はアクティブのみ、false は削除済みのみを想定。
- name: Create
  signature: Create(ctx context.Context, user *User) error
  behavior: ユーザーを 1 件永続化する。
- name: CountByActive
  signature: CountByActive(ctx context.Context, active *bool) (int64, error)
  behavior: アクティブ状態（active）に基づきユーザーの総件数を返す。
```
