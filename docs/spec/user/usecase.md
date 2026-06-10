# User — Usecase Spec

> 既存実装（`internal/usecase/user`）を spec 化したベースに、未実装の詳細系エンドポイント（GetUsersDetail / Put / Patch / Delete / PutUsersMePassword）向けの GetUser / UpdateUser / UpdateUserPartially / ChangePassword / DeleteUser を追記したもの。
> 追記分は scaffold の入力となる目標仕様。更新・論理削除は「load → ドメインメソッドで変更 → Update で永続化」に統一。プロフィール更新系（PUT / PATCH）は password を扱わず、パスワード変更は自己変更専用の ChangePassword（PUT /v1/users/me/password、認証トークンの本人に束縛し現パスワード照合のうえ行う）で実施する。
> 論理削除済みユーザーは detail 系（GET / PUT / PATCH / DELETE）の対象外とし、`FindByID` / `Update` の SQL で `deleted_at IS NULL` をフィルタする。これにより削除済みへの取得・更新・再削除はすべて `NotFound`（404）に統一される（GET で削除済みを返したり、更新でエラー種別がぶれることを防ぐ）。

## Overview

ユーザーユースケースは、ユーザー一覧取得・作成・件数取得を提供するアプリケーションサービス。ドメインの `user.Repository` と `prefecture.Repository` をオーケストレーションし、ドメインエンティティを外側に晒さず DTO（出力 `UserView` / 更新入力 `UpdateProfileParams`）へ変換して返す。

都道府県は ID 参照のみを保持する設計のため、一覧・作成ともに `prefecture.Repository` から都道府県名を解決して DTO に詰める。作成時はトランザクション境界内で都道府県解決・エンティティ生成・永続化を行う。パスワードは `RawPassword` で検証後、`security.Hasher` でハッシュ化してからエンティティに渡す。

## Interface

```yaml
package: internal/usecase/user
name: Usecase
methods:
  - name: ListUsers
    signature: ListUsers(ctx context.Context, active *bool, page *paging.Paging) ([]UserView, error)
  - name: CreateUser
    signature: CreateUser(ctx context.Context, dto *CreateParamsDTO) (UserView, error)
  - name: CountUsers
    signature: CountUsers(ctx context.Context, active *bool) (int64, error)
  # 詳細系エンドポイント向け（追記分）
  - name: GetUser
    signature: GetUser(ctx context.Context, id uuid.UUID) (UserView, error)
  - name: UpdateUser
    signature: UpdateUser(ctx context.Context, id uuid.UUID, dto *UpdateProfileParams) (UserView, error)
  - name: UpdateUserPartially
    signature: UpdateUserPartially(ctx context.Context, id uuid.UUID, dto *PatchParamsDTO) (UserView, error)
  - name: ChangePassword
    signature: ChangePassword(ctx context.Context, id uuid.UUID, currentPassword, newPassword string) error
  - name: DeleteUser
    signature: DeleteUser(ctx context.Context, id uuid.UUID) error
```

## DTOs

```yaml
- name: UserView
  description: ユーザー取得結果の出力 DTO。
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
- name: UpdateProfileParams
  description: ユーザープロフィール更新の入力（可変フィールド）。出力専用の DeletedAt は含まない。
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
      type: string
    - name: City
      type: string
    - name: Street
      type: string
    - name: Building
      type: "*string"
- name: CreateParamsDTO
  description: ユーザー作成に必要なパラメータ。UpdateProfileParams を埋め込む。
  fields:
    - name: UserID
      type: uuid.UUID
    - name: RawPassword
      type: string
    - name: UpdateProfileParams
      type: UpdateProfileParams   # embedded
# 詳細系エンドポイント向け（追記分）
# UpdateUser（PUT・プロフィール全更新）は入力に UpdateProfileParams、出力に UserView を使う
# （password は含めず、変更は ChangePassword で行う。DeletedAt は出力専用で更新入力には持たせない）。
- name: PatchParamsDTO
  description: |
    PATCH（部分更新）の入力。指定フィールドのみ更新し、nil（未指定）は据え置き。password は含めない。
    PATCH は「部分マージ」セマンティクスを採用し、**フィールドのクリア（null 設定）は提供しない**。
    送信が省略でも null でも同じ nil として扱い「据え置き」となる（生成リクエスト型が `*string` で
    未指定と null を区別できないため、JSON Merge Patch 的な区別は持たない）。
    フィールドを空/クリアしたい場合は **PUT（全更新、building は null 指定可）** を使う。
  fields:
    - name: FirstName
      type: "*string"
    - name: LastName
      type: "*string"
    - name: Email
      type: "*string"
    - name: Phone
      type: "*string"
    - name: PostalCode
      type: "*string"
    - name: PrefectureName
      type: "*string"
    - name: City
      type: "*string"
    - name: Street
      type: "*string"
    - name: Building
      type: "*string"   # nullable だが、未指定/null とも nil=据え置き扱い。クリアは PUT を使う
```

## Dependencies

```yaml
- tracer            # observability.TracerFactory -> LayerTracer（メソッドごとに span）
- tx_manager        # boundary/tx.Manager
- clock             # boundary/clock.Clock
- encrypter         # boundary/security.Hasher
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
  - 各ユーザーを UserView へ変換し、都道府県名を埋める
calls:
  - user_repository.FindByActive
  - prefecture_repository.FindByIDs
errors:
  - userRepo / pftRepo のエラーを伝播。ユーザーが参照する都道府県を FindByIDs で解決できない場合は参照整合性破れとして ErrInternal(500)
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
  - 生成したエンティティと都道府県名から UserView を構築して返す
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

### GetUser

```yaml
tx_required: false
steps:
  - user_repository.FindByID で単一ユーザーを取得（存在しない / 論理削除済みなら NotFound 伝播）
  - prefecture_repository.FindByID で都道府県名を解決
  - UserView へ変換して返す（PrefectureName を埋める）
calls:
  - user_repository.FindByID
  - prefecture_repository.FindByID
errors:
  - user FindByID の NotFound は伝播（404）。prefecture FindByID の NotFound は参照整合性破れとして ErrInternal(500)、その他は伝播
```

### UpdateUser

```yaml
tx_required: true
steps:
  - clock.Now で現在時刻を取得
  - トランザクション内で
      - user_repository.FindByID で対象を取得（存在しない / 論理削除済みなら NotFound 伝播）
      - prefecture_repository.FindByName で都道府県を名前解決
      - user.UpdateProfile で全プロフィールフィールド + updatedAt を置換（password は対象外）
      - user_repository.Update で永続化
  - UserView へ変換して返す
calls:
  - clock.Now
  - tx_manager.Do
  - user_repository.FindByID
  - prefecture_repository.FindByName
  - user.UpdateProfile
  - user_repository.Update
errors:
  - FindByID(NotFound) / FindByName / UpdateProfile / Update を伝播
```

### ChangePassword

```yaml
tx_required: true
steps:
  - clock.Now で現在時刻を取得
  - user.NewRawPassword で現パスワード・新パスワードを検証（長さ制約。現パスワードも検証し bcrypt の 72 バイト切り詰めを防ぐ）
  - トランザクション内で
      - user_repository.FindByID で対象を取得（存在しない / 論理削除済みなら NotFound 伝播）
      - encrypter.Compare で現パスワードと保存済みハッシュを照合（不一致なら ErrCurrentPasswordMismatch=422。authn=401/権限=403 ではなく、整形済みリクエストの意味的検証失敗として扱う）
      - encrypter.Hash で新パスワードのハッシュを生成
      - user.ChangePassword でパスワードハッシュ + updatedAt を置換
      - user_repository.Update で永続化
calls:
  - clock.Now
  - user.NewRawPassword
  - tx_manager.Do
  - user_repository.FindByID
  - encrypter.Compare
  - encrypter.Hash
  - user.ChangePassword
  - user_repository.Update
errors:
  - NewRawPassword(現/新) / FindByID(NotFound) / Compare / ErrCurrentPasswordMismatch(422) / Hash / ChangePassword / Update を伝播
```

### UpdateUserPartially

```yaml
tx_required: true
steps:
  - clock.Now で現在時刻を取得
  - トランザクション内で
      - user_repository.FindByID で対象を取得（存在しない / 論理削除済みなら NotFound 伝播）
      - PrefectureName が指定されていれば prefecture_repository.FindByName で都道府県解決（入力エラーは伝播）、未指定なら prefecture_repository.FindByID で現在の都道府県を解決（レスポンス名用。未解決は参照整合性破れ）
      - 各フィールドは provided なら新値、nil なら現在値（getter）をマージしてフルセットを構築（未指定/null とも nil=据え置き。フィールドのクリアは PATCH では提供せず PUT を使う）
      - user.UpdateProfile でマージ後のフルセット + updatedAt を置換（password は更新しない）
      - user_repository.Update で永続化
  - UserView へ変換して返す
calls:
  - clock.Now
  - tx_manager.Do
  - user_repository.FindByID
  - prefecture_repository.FindByName
  - user.UpdateProfile
  - user_repository.Update
errors:
  - user FindByID(NotFound) / FindByName / UpdateProfile / Update を伝播。指定なし時に現在の都道府県(FindByID)が NotFound なら ErrInternal(500)
```

### DeleteUser

```yaml
tx_required: true
steps:
  - clock.Now で現在時刻を取得
  - トランザクション内で
      - user_repository.FindByID で対象を取得（存在しない / 論理削除済みなら NotFound 伝播。FindByID が deleted_at IS NULL でフィルタするため、削除済みへの再 DELETE は NotFound になる）
      - user.MarkAsDeleted で deletedAt を設定（ドメイン不変条件として既削除なら ErrAlreadyDeleted を返すが、FindByID フィルタにより通常経路では到達しない防御的チェック）
      - user_repository.Update で永続化（論理削除）
calls:
  - clock.Now
  - tx_manager.Do
  - user_repository.FindByID
  - user.MarkAsDeleted
  - user_repository.Update
errors:
  - FindByID(NotFound) / MarkAsDeleted(ErrAlreadyDeleted) / Update を伝播
```
