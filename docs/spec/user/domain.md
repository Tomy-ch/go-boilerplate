# User — Domain Spec

> 既存実装（`internal/domain/user`）を spec 化したベースに、未実装の詳細系エンドポイント（GetUsersDetail / Put / Patch / Delete）向けの更新・論理削除を追記したもの。
> 追記分は scaffold の入力となる目標仕様（FindByID / Update + UpdateProfile / MarkAsDeleted）。更新・論理削除は「load → ドメインメソッドで変更 → Update で永続化」に統一する方針。

## Overview

ユーザー集約は、アカウントの基本情報（氏名・認証情報・連絡先）と住所情報（都道府県・市区町村・番地・建物・郵便番号）を保持するドメインの中核エンティティ。生成時に全フィールドの形式・長さ・必須を検証し、不変条件を満たさない `User` は構築できない。

都道府県は ID 参照（`prefectureID`）のみを保持し、表示名は別集約（`prefecture`）から解決する。認証は外部の OIDC/JWT に委譲し、`User` はパスワード等の認証情報を保持しない。

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
  - name: email
    type: Email               # 値オブジェクト（長さ + local@domain 形式を factory で検証）
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
    type: PostalCode           # 値オブジェクト（NNN-NNNN 形式を factory で検証。OpenAPI の pattern と一致）
    required: true
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
- `UpdateProfile` での更新時は `updatedAt >= 現在の updatedAt`（単調非減少）。違反時 `ErrInvalidUpdatedAt`

## Behavior Methods

```yaml
# 派生メソッド（単純フィールド getter ではない）
- name: IsActive
  signature: IsActive() bool
  description: |
    ユーザーが在籍しているか（退会していないか）を返す述語。
    「在籍している」が指す条件はこの述語が定義であり、永続化層の絞り込み条件が定義になることはない。

# 状態遷移メソッド（更新・論理削除エンドポイント向け。追記分）
- name: UpdateProfile
  signature: UpdateProfile(profile Profile, updatedAt time.Time) error
  description: |
    プロフィール（氏名・連絡先・住所・都道府県ID）と updatedAt を一括で置き換える。
    各フィールドは New と同じ不変条件で検証する（長さ範囲 / prefectureID 非 nil /
    building は非 nil 時のみ検証）。updatedAt は createdAt 以降かつ現在の updatedAt 以降
    （単調非減少）でなければならず、違反時は ErrInvalidUpdatedAt を返す。
    既に論理削除済みの場合は ErrAlreadyDeleted を返し、更新しない。
    PUT は全フィールド指定、PATCH は load した現在値に provided フィールドをマージした
    フルセットを渡して呼ぶ（usecase 側でマージ）。
- name: MarkAsDeleted
  signature: MarkAsDeleted(deletedAt time.Time) error
  description: |
    論理削除。deletedAt を設定する（deletedAt >= createdAt かつ >= updatedAt を検証）。
    論理削除は更新操作でもあるため、updatedAt も deletedAt の値へ更新する。
    既に削除済み（deletedAt != nil）の場合は ErrAlreadyDeleted を返す。
```

## Value Objects

```yaml
- name: Email
  underlying_type: string
  validation: 長さが 1 以上 100 以下、かつ local@domain 形式（ドメインにドットを含む）。違反時 ErrInvalidEmail
  factory: NewEmail
  methods:
    - name: Value
      returns: string
- name: PostalCode
  underlying_type: string
  validation: NNN-NNNN 形式（半角数字 3 桁 + ハイフン + 4 桁。OpenAPI リクエストの pattern と一致）。違反時 ErrInvalidPostalCode
  factory: NewPostalCode
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
# 詳細系エンドポイント向け（追記分）
- name: FindByID
  signature: FindByID(ctx context.Context, id uuid.UUID) (*User, error)
  behavior: |
    ID から単一ユーザーを取得する。存在しない場合は apperror.ErrNotFound に
    正規化されたエラーを返す（infra の pgerror.NormalizeError 経由）。
- name: Update
  signature: Update(ctx context.Context, user *User) error
  behavior: |
    ID をキーに mutable フィールド（氏名・連絡先・住所・prefectureID）と
    updatedAt / deletedAt を更新する。PUT / PATCH / DELETE（論理削除）すべてが
    load → ドメインメソッドで変更 → 本メソッドで永続化、という共通経路で利用する。
# 退会後の物理削除ジョブ向け（追記分）
- name: FindDeletedBefore
  signature: FindDeletedBefore(ctx context.Context, cutoff time.Time, afterID *uuid.UUID, limit int32) ([]uuid.UUID, error)
  behavior: |
    cutoff より前に論理削除されたユーザーの ID を、ID の昇順で最大 limit 件返す。
    afterID=nil は先頭から、それ以外は afterID より後ろを返す keyset ページネーション。
    削除できない候補（購入を持つユーザー）を挟んでも前進できるよう、境界を offset ではなく
    ID で受け取る。エンティティを再構築せず ID だけを返すのは、後続が物理削除のみを行うため。
- name: PurgeByIDs
  signature: PurgeByIDs(ctx context.Context, ids []uuid.UUID) (int64, error)
  behavior: |
    指定した ID のユーザーを従属データごと物理削除し、削除したユーザーの件数を返す。
    ids が空の場合は何も削除せず 0 を返す。トランザクション境界は usecase が持つ
    （1 バッチ = 1 トランザクション）。
    論理削除されていないユーザーは従属データを含めて削除対象から外れるため、返る件数が ids の
    件数を下回ることがある。保持期間の判定は usecase の責務だが、「生きているユーザーを不可逆に
    消さない」ことは呼び手を問わない不変条件なので、永続化側の最終防壁として持つ。
```
