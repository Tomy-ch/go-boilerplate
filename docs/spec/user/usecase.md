# User — Usecase Spec

> 既存実装（`internal/usecase/user`）を spec 化したもの。手書き実装から逆生成した現状仕様であり、未実装機能（取得詳細 / 更新 / 部分更新 / 削除）は含まない。

## Overview

ユーザーユースケースは、ユーザー一覧取得・作成・件数取得を提供するアプリケーションサービス。ドメインの `user.Repository` と `prefecture.Repository` をオーケストレーションし、ドメインエンティティを外側に晒さず DTO（`MutableFields`）へ変換して返す。

都道府県は ID 参照のみを保持する設計のため、一覧・作成ともに `prefecture.Repository` から都道府県名を解決して DTO に詰める。作成時はトランザクション境界内で都道府県解決・エンティティ生成・永続化を行う。パスワードは `RawPassword` で検証後、`security.Encrypter` でハッシュ化してからエンティティに渡す。

## Interface

```yaml
package: internal/usecase/user
name: Usecase
methods:
  - name: ListUsers
    signature: ListUsers(ctx context.Context, active *bool, page *paging.Paging) ([]MutableFields, error)
  - name: CreateUser
    signature: CreateUser(ctx context.Context, dto *CreateParamsDTO) (MutableFields, error)
  - name: CountUsers
    signature: CountUsers(ctx context.Context, active *bool) (int64, error)
```

## DTOs

```yaml
- name: MutableFields
  description: ユーザー取得結果の DTO。
  fields:
    - name: FirstName
      type: string
    - name: LastName
      type: string
    - name: Email
      type: string
    - name: Phone
      type: string
    - name: PostalCode
      type: string
    - name: PrefectureName
      type: string          # prefecture.Repository から解決した都道府県名
    - name: City
      type: string
    - name: Street
      type: string
    - name: Building
      type: "*string"
    - name: DeletedAt
      type: "*time.Time"
- name: CreateParamsDTO
  description: ユーザー作成に必要なパラメータ。MutableFields を埋め込む。
  fields:
    - name: UserID
      type: uuid.UUID
    - name: RawPassword
      type: string
    - name: MutableFields
      type: MutableFields   # embedded
```

## Dependencies

```yaml
- tracer            # observability.TracerFactory -> LayerTracer（メソッドごとに span）
- tx_manager        # boundary/tx.Manager
- clock             # boundary/clock.Clock
- encrypter         # boundary/security.Encrypter
- user_repository   # domain/user.Repository
- prefecture_repository  # domain/prefecture.Repository
```

## Workflow

### ListUsers

```yaml
tx_required: false
steps:
  - userRepo.FindByActive で active / ページング条件に基づきユーザー一覧を取得
  - 取得した各ユーザーの prefectureID を集約し、pftRepo.FindByIDs で都道府県をまとめて解決（N+1 回避、子 span で計測）
  - prefectureID -> Prefecture のマップを構築
  - 各ユーザーを MutableFields へ変換し、都道府県名を埋める
calls:
  - user_repository.FindByActive
  - prefecture_repository.FindByIDs
errors:
  - userRepo / pftRepo のエラーをそのまま伝播
```

### CreateUser

```yaml
tx_required: true
steps:
  - clock.Now で現在時刻を取得
  - user.NewRawPassword で平文パスワードを検証
  - encrypter.Hash でパスワードハッシュを生成
  - トランザクション内で
      - pftRepo.FindByName で都道府県を名前解決
      - user.New でエンティティ生成（不変条件検証）
      - userRepo.Create で永続化
  - 生成したエンティティと都道府県名から MutableFields を構築して返す
calls:
  - clock.Now
  - user.NewRawPassword
  - encrypter.Hash
  - tx_manager.Do            # トランザクション境界。内部で以下を実行
  - prefecture_repository.FindByName
  - user.New
  - user_repository.Create
errors:
  - NewRawPassword / Hash / FindByName / user.New / Create のエラーを伝播
```

### CountUsers

```yaml
tx_required: false
steps:
  - userRepo.CountByActive で active 条件に基づく総件数を取得して返す
calls:
  - user_repository.CountByActive
errors:
  - userRepo のエラーをそのまま伝播
```
