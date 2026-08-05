# User — Usecase Spec

> 既存実装（`internal/usecase/user`）を spec 化したベースに、詳細系エンドポイント（GetUsersDetail / Put / Patch / Delete）向けの GetUser / UpdateUser / UpdateUserPartially / DeleteUser を追記したもの。
> 更新・論理削除は「load → ドメインメソッドで変更 → Update で永続化」に統一。認証は外部の OIDC/JWT に委譲し、ユーザーの認証情報は本ユースケースでは扱わない。
> 論理削除済みユーザーは detail 系（GET / PUT / PATCH / DELETE）の対象外とし、`FindByID` / `Update` の SQL で `deleted_at IS NULL` をフィルタする。これにより削除済みへの取得・更新・再削除はすべて `NotFound`（404）に統一される（GET で削除済みを返したり、更新でエラー種別がぶれることを防ぐ）。

## Overview

ユーザーユースケースは、ユーザー一覧取得・作成・件数取得を提供するアプリケーションサービス。ドメインの `user.Repository` と `prefecture.Repository` をオーケストレーションし、ドメインエンティティを外側に晒さず DTO（出力 `UserView` / 更新入力 `UpdateProfileParams`）へ変換して返す。

都道府県は ID 参照のみを保持する設計のため、一覧・作成ともに `prefecture.Repository` から都道府県名を解決して DTO に詰める。作成時はトランザクション境界内で都道府県解決・エンティティ生成・永続化を行う。

## Interface

```yaml
package: internal/usecase/user
name: Usecase
methods:
  - name: ListUsers
    signature: ListUsers(ctx context.Context, active *bool, page *paging.Page) ([]UserView, error)
  - name: CreateUser
    signature: CreateUser(ctx context.Context, dto *CreateParamsDTO) (UserView, error)
  - name: CountUsers
    signature: CountUsers(ctx context.Context, active *bool) (int64, error)
  # 詳細系エンドポイント向け（追記分。いずれも認可判定のため認証主体 authn を受け取る）
  - name: GetUser
    signature: GetUser(ctx context.Context, authn *auth.Authn, id uuid.UUID) (UserView, error)
  - name: UpdateUser
    signature: UpdateUser(ctx context.Context, authn *auth.Authn, id uuid.UUID, dto *UpdateProfileParams) (UserView, error)
  - name: UpdateUserPartially
    signature: UpdateUserPartially(ctx context.Context, authn *auth.Authn, id uuid.UUID, dto *PatchParamsDTO) (UserView, error)
  - name: DeleteUser
    signature: DeleteUser(ctx context.Context, authn *auth.Authn, id uuid.UUID) error
```

退会後の物理削除はジョブ専用の入口で、HTTP ハンドラが使う `Usecase` とは利用者も依存も異なるため、独立したインターフェースに分ける（`outbox.GCUsecase` と同じ形）。

```yaml
package: internal/usecase/user
name: PurgeUsecase
methods:
  - name: PurgeDeleted
    signature: PurgeDeleted(ctx context.Context, retention time.Duration, batchSize int32, dryRun bool) (PurgeResult, error)
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
    - name: UpdateProfileParams
      type: UpdateProfileParams   # embedded
# 詳細系エンドポイント向け（追記分）
# UpdateUser（PUT・プロフィール全更新）は入力に UpdateProfileParams、出力に UserView を使う
# （DeletedAt は出力専用で更新入力には持たせない）。
- name: PatchParamsDTO
  description: |
    PATCH（部分更新）の入力。指定フィールドのみ更新し、nil（未指定）は据え置き。
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
- name: PurgeResult
  description: 退会後の物理削除の実行結果。
  fields:
    - name: Purged
      type: int64   # 物理削除したユーザー件数。dryRun では削除対象となった件数
    - name: SkippedWithPurchases
      type: int64   # 購入を保持しているため削除しなかったユーザー件数
```

## Dependencies

```yaml
- tracer            # observability.TracerFactory -> LayerTracer（メソッドごとに span）
- tx_manager        # boundary/tx.Manager
- clock             # boundary/clock.Clock
- authorizer        # boundary/authz.Authorizer（詳細系の認可判定。admin または対象ユーザー本人）
- user_repository   # domain/user.Repository
- prefecture_repository  # domain/prefecture.Repository
- purchase_repository    # domain/purchase.Repository（退会時の進行中購入の確認）
- outbox_emit       # usecase/outbox.EmitUsecase（退会イベントの発行）
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
  - トランザクション内で
      - pftRepo.FindByName で都道府県を名前解決
      - user.New でエンティティ生成（不変条件検証）
      - userRepo.Create で永続化
  - 生成したエンティティと都道府県名から UserView を構築して返す
calls:
  - clock.Now
  - tx_manager.Do            # トランザクション境界。内部で以下を実行
  - prefecture_repository.FindByName
  - user.New
  - user_repository.Create
errors:
  - FindByName / user.New / Create のエラーを伝播
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
      - user.UpdateProfile で全プロフィールフィールド + updatedAt を置換
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

### UpdateUserPartially

```yaml
tx_required: true
steps:
  - clock.Now で現在時刻を取得
  - トランザクション内で
      - user_repository.FindByID で対象を取得（存在しない / 論理削除済みなら NotFound 伝播）
      - PrefectureName が指定されていれば prefecture_repository.FindByName で都道府県解決（入力エラーは伝播）、未指定なら prefecture_repository.FindByID で現在の都道府県を解決（レスポンス名用。未解決は参照整合性破れ）
      - 各フィールドは provided なら新値、nil なら現在値（getter）をマージしてフルセットを構築（未指定/null とも nil=据え置き。フィールドのクリアは PATCH では提供せず PUT を使う）
      - user.UpdateProfile でマージ後のフルセット + updatedAt を置換
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
  - authorizer.Authorize で退会の認可を確認（対象ユーザーを所有者とするリソース。admin または本人のみ許可。authn が nil なら ErrUnauthenticated）
  - clock.Now で現在時刻を取得
  - トランザクション内で（論理削除・イベント発行・拒否判定を単一 tx にまとめ、退会だけが成立してイベントが失われることを防ぐ）
      - user_repository.LockByID で対象を排他ロックして取得（存在しない / 論理削除済みなら NotFound 伝播。SQL が deleted_at IS NULL でフィルタするため、削除済みへの再 DELETE は NotFound になる）
      - purchase_repository.ExistsInProgressByUserID で進行中の購入を確認し、残っていれば Conflict で退会を拒否（論理削除もイベントも残さない）
      - user.MarkAsDeleted で deletedAt を設定（ドメイン不変条件として既削除なら ErrAlreadyDeleted を返すが、LockByID フィルタにより通常経路では到達しない防御的チェック）
      - user_repository.Update で永続化（論理削除）
      - event.BuildWithdrawn で自己完結スナップショットを構築し、outbox_emit.Emit で user.withdrawn.v1 を発行（退会に伴う関連データの後始末は受信側の結果整合に委ねる）
calls:
  - authorizer.Authorize
  - clock.Now
  - tx_manager.Do
  - user_repository.LockByID
  - purchase_repository.ExistsInProgressByUserID
  - user.MarkAsDeleted
  - user_repository.Update
  - outbox_emit.Emit
errors:
  - authn が nil なら ErrUnauthenticated / Authorize 拒否は ErrForbidden(PermissionDenied) を伝播
  - 進行中の購入が残っている場合は ErrConflict
  - LockByID(NotFound) / ExistsInProgressByUserID / MarkAsDeleted(ErrAlreadyDeleted) / Update / Emit を伝播
```

ロックの取得順が不変条件である（[ADR-0107]）。`LockByID` は進行中購入の判定より**前**に置く。判定より後だと
「退会が判定を通過 → 購入作成が成立 → 退会が確定」の順序を止められず、退会済みユーザーに進行中の購入が
ぶら下がる。購入作成側は同じ行を共有ロックで押さえるため、この排他ロックとだけ衝突して直列化される。

### PurgeDeleted（PurgeUsecase）

```yaml
tx_required: true   # 1 バッチ = 1 トランザクション
steps:
  - retention / batchSize が 0 以下なら既定値（30 日 / 1000 件）に置き換える
  - clock.Now から retention を引いて打ち切り時刻 cutoff を決める
  - 候補が尽きるまでバッチを反復（各バッチをトランザクション内で実行）
      - user_repository.FindDeletedBefore で cutoff より古い候補を ID 昇順の keyset で最大 batchSize 件取得
      - 候補が 0 件ならそのバッチは何もしない
      - purchase_repository.FindUserIDsWithPurchases で購入を持つ候補を特定し、スキップ件数に計上
      - dryRun でなければ user_repository.PurgeByIDs で残りを物理削除し、削除件数を加算（dryRun では対象件数のみ加算）
      - 取得件数が batchSize に満たなければ終了。満ちていれば境界を「取得した候補の末尾 ID」へ進めて次バッチへ
  - 累計の PurgeResult を返す
calls:
  - clock.Now
  - tx_manager.Do
  - user_repository.FindDeletedBefore
  - purchase_repository.FindUserIDsWithPurchases
  - user_repository.PurgeByIDs
errors:
  - 各 Repository のエラーを伝播。失敗したバッチはロールバックされるが、それ以前にコミットされた
    バッチの物理削除は取り消せないため、エラー時もそこまでの累計を PurgeResult に含めて返す
notes:
  - 境界は削除可否によらず必ず候補の末尾まで進める。購入保持でスキップされた候補は削除されず残るため、
    境界を進めないと同じ候補を取り直し続け、先頭バッチが全件スキップ対象のときに無限ループする。
```

[ADR-0107]: ../../adr/0107-withdrawal-purchase-row-lock-serialization.md
