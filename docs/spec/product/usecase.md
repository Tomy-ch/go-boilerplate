# Product — Usecase Spec

> 公開商品一覧（`GET /v1/products`）は単一集約・products 自身の列へのフィルタ / keyword / sort・cursor ページングであり、
> QueryService ではなく domain `product.Repository` の `FindPublishedList` に委譲する（ADR-0027 / `docs/rules.md` の Repository 境界に準拠）。

## Overview

商品一覧ユースケースは、公開済み商品を公開日時順（cursor ページネーション）で取得する read-only な thin orchestrator。`product.Repository`（domain Repository）の `FindPublishedList` に委譲し、取得した `Product` エンティティ一覧を usecase DTO（`ProductView`）へ写像して返す。

cursor ページングは「直前ページ末尾行のソートキー `(publishedAt, id)` を不透明トークン化して次ページを取得する」keyset 方式。usecase は不透明カーソルの符号化・復号（`encodeProductCursor` / `decodeProductCursor`）を担い、domain へは境界を primitive（`AfterPublishedAt` / `AfterID`）で渡す。次ページ有無は `limit + 1` 件取得して超過分で判定し（`hasNext`）、超過時のみ末尾行から `NextCursor` を生成する。

sort は `-publishedAt`（既定=降順）/ `publishedAt`（昇順）の 2 値のみ。controller が enum を `Ascending` bool に写像して渡す。ステータス可視範囲の絞り込みは後続 PBI（#555）で対応する。ドメイン集約を outer 層へ露出させないため、`Product` は usecase 内で `ProductView` へ写像してから返す（DTO Boundary）。

商品作成（`POST /v1/products`）は admin 認可のうえ、`tx.Manager` の境界内で商品ステータス / カテゴリの名称を ID から解決し、`Product` を構築して登録する write ユースケース。マスタ不在はサーバ側整合性異常（500）、価格・在庫などの業務不変条件違反は 422 に落とす。

在庫の増減（`PATCH /v1/products/{productId}/stock`）は admin 認可のうえ、`tx.Manager` の境界内で対象商品を悲観ロックしてから在庫を増減する write ユースケース。購入による在庫減算と同じ行を対象とするため、取得〜更新が直列化され、並行する増減は失われず合成される（ADR-0100）。待てば解消しうる競合は一時障害（503）、取得後の変化を検出した恒久的な衝突は 409 として露出する。

在庫僅少一覧（`GET /v1/products/low-stock`）は admin 認可のうえ、在庫が在庫警告閾値以下まで減った商品を在庫の少ない順に上位 `limit` 件返す read-only な thin orchestrator。判定条件と並び順は `product.Repository` の `FindAllLowStock` が持ち、usecase は取得件数の正規化（既定値 20 / 範囲 1〜100 へのクランプ）と DTO への写像のみを担う。cursor ページングは持たない。

商品部分更新（`PATCH /v1/products/{productId}`）は admin 認可のうえ、`tx.Manager` の境界内で read-modify-write を行う write ユースケース。読み込み時点のバージョンを条件に更新することで、並行編集による上書き（lost update）を防ぐ。送られたフィールドのみを反映し、未送信は現在値を据え置き、null 明示はクリアする 3 状態の解決は usecase が担う（domain へは解決後の確定値のみを渡す）。参照の再解決は `statusId` / `categoryId` が指定された場合に限る。

## Interface

```yaml
package: internal/usecase/product
name: Usecase
methods:
  - name: ListProducts
    signature: ListProducts(ctx context.Context, params ListProductsParams) (*ProductListView, error)
  - name: GetProduct
    signature: GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error)
  - name: CreateProduct
    signature: CreateProduct(ctx context.Context, authn *auth.Authn, params CreateProductParams) (ProductView, error)
  - name: UpdateProduct
    signature: UpdateProduct(ctx context.Context, authn *auth.Authn, id uuid.UUID, params UpdateProductParams) (ProductView, error)
  - name: UpdateProductStock
    signature: UpdateProductStock(ctx context.Context, authn *auth.Authn, id uuid.UUID, params UpdateProductStockParams) (ProductView, error)
  - name: ListLowStockProducts
    signature: ListLowStockProducts(ctx context.Context, authn *auth.Authn, params ListLowStockProductsParams) (ProductLowStockListView, error)
```

未参照画像の回収はジョブ専用の入口で、HTTP ハンドラが使う `Usecase` とは利用者も依存も異なるため、
独立したインターフェースに分ける（`outbox.GCUsecase` / `user.PurgeUsecase` と同じ形）。

```yaml
package: internal/usecase/product
name: ImageGCUsecase
methods:
  - name: SweepOrphans
    signature: SweepOrphans(ctx context.Context, grace time.Duration, batchSize int32, dryRun bool) (ImageGCResult, error)
```

## DTOs

```yaml
- name: ListProductsParams
  description: 公開商品一覧取得の入力。cursor（取得件数 + 境界）とフィルタ・並び順を保持する。
  fields:
    - name: Cursor
      type: "*paging.Cursor"
    - name: CategoryID
      type: "*uuid.UUID"
    - name: StatusID
      type: "*uuid.UUID"
    - name: Keyword
      type: "*string"
    - name: Ascending
      type: bool
- name: ProductView
  description: 商品 1 件分の usecase 出力 DTO。domain エンティティ Product から写像する。Price は価格スケールの十進量（pkg/decimal.Decimal）で、controller が decimal 文字列へ整形する。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Name
      type: string
    - name: Description
      type: "*string"
    - name: Price
      type: decimal.Decimal
    - name: Quantity
      type: int
    - name: StockWarningThreshold
      type: "*int"
    - name: StatusID
      type: uuid.UUID
    - name: CategoryID
      type: uuid.UUID
    - name: PublishedAt
      type: "*time.Time"     # 未公開は nil
    - name: ImagePath
      type: "*string"        # 画像未設定は nil
    - name: Version
      type: int              # 楽観ロックのバージョン。部分更新の要求へそのまま渡す
- name: CreateProductParams
  description: 商品作成の入力。price は十進文字列で受け取り usecase で decimal へ解釈する（負値は 422）。publishedAt / imagePath は nil 許容。
  fields:
    - name: Name
      type: string
    - name: Description
      type: "*string"
    - name: Price
      type: string
    - name: Quantity
      type: int
    - name: StockWarningThreshold
      type: "*int"
    - name: CategoryID
      type: uuid.UUID
    - name: StatusID
      type: uuid.UUID
    - name: PublishedAt
      type: "*time.Time"
    - name: ImagePath
      type: "*string"
- name: UpdateProductParams
  description: |
    商品部分更新の入力。nil のポインタは未指定（現在値を据え置く）を表す。
    クリアも許容するフィールドは pkg/patch.Field で 3 状態（未指定 / null 明示 / 値指定）を表す。
  fields:
    - name: Version
      type: int                       # 更新対象を読み込んだ時点のバージョン。不一致は 409
    - name: Name
      type: "*string"
    - name: Price
      type: "*string"
    - name: Quantity
      type: "*int"
    - name: CategoryID
      type: "*uuid.UUID"
    - name: StatusID
      type: "*uuid.UUID"
    - name: Description
      type: "patch.Field[string]"     # null 明示でクリア
    - name: StockWarningThreshold
      type: "patch.Field[int]"        # null 明示でクリア
    - name: PublishedAt
      type: "patch.Field[time.Time]"  # null 明示でクリア（未公開へ戻す）
    - name: ImagePath
      type: "patch.Field[string]"     # null 明示でクリア
- name: UpdateProductStockParams
  description: 在庫の増減の入力。
  fields:
    - name: Delta
      type: int                       # 在庫の増減量。正で補充、負で差し引き。増減後が負になる要求は 422
- name: ProductListView
  description: 公開商品一覧（cursor ページネーション）の取得結果。
  fields:
    - name: Items
      type: "[]ProductView"
    - name: NextCursor
      type: "*string"        # 最終ページの場合は nil
- name: ListLowStockProductsParams
  description: 在庫僅少商品一覧取得の入力。cursor を持たない top-N のため取得件数のみを受け取る。
  fields:
    - name: Limit
      type: int              # 下限（1）未満は既定値 20、上限（100）超過は 100 へクランプ
- name: ProductLowStockListView
  description: 在庫僅少商品一覧の取得結果。cursor を持たないため NextCursor はない。
  fields:
    - name: Items
      type: "[]ProductView"  # 在庫の少ない順（同数は商品 ID 昇順）
```

## Dependencies

```yaml
- tracer              # observability.TracerFactory -> LayerTracer
- txm                 # boundary/tx.Manager（CreateProduct / UpdateProduct / UpdateProductStock のトランザクション境界）
- product_repository  # domain/product.Repository（FindPublishedList / FindAllLowStock / FindPublishedByID / FindByID / LockByID / Create / Update / UpdateStock）
- category_repository # domain/product/category.Repository（FindByID でカテゴリ名称を解決）
- status_repository   # domain/product/status.Repository（FindByID でステータス名称を解決）
- authorizer          # boundary/authz.Authorizer（CreateProduct / UpdateProduct / UpdateProductStock / ListLowStockProducts の admin 認可）
```

## Workflow

### ListProducts

```yaml
tx_required: false
steps:
  - Cursor が nil の場合は apperror.ErrInvalidArgument を返す
  - decodeProductCursor で不透明カーソルを keyset 境界（publishedAt, id）へ復号する（先頭ページは境界なし）
  - domain の ListParams を組み立てる（Limit=Cursor.Limit32()+1、Ascending、CategoryID/StatusID/Keyword、AfterPublishedAt/AfterID）
  - product_repository.FindPublishedList で公開商品を取得する
  - 取得件数が Cursor.Limit() を超える場合は次ページありと判定し、末尾を切り詰める
  - 各 Product を ProductView（ID / Name / Description / Price / Quantity / StockWarningThreshold / StatusID / CategoryID / PublishedAt）へ写像する
  - 次ページありの場合、切り詰め後の末尾行から encodeProductCursor で NextCursor を生成する
calls:
  - product_repository.FindPublishedList
errors:
  - Cursor が nil の場合は apperror.ErrInvalidArgument
  - カーソル復号失敗時は apperror.ErrInvalidArgument（decodeProductCursor 由来）
  - product_repository.FindPublishedList のエラーをそのまま伝播する
```

### GetProduct

```yaml
tx_required: false
steps:
  - product_repository.FindPublishedByID で公開中の単一商品を取得する（未存在・非公開はいずれも NotFound）
  - Product を ProductView（ID / Name / Description / Price / Quantity / StockWarningThreshold / StatusID / CategoryID / PublishedAt）へ写像する
calls:
  - product_repository.FindPublishedByID
errors:
  - product_repository.FindPublishedByID のエラーをそのまま伝播する（未存在・非公開は apperror.ErrNotFound → 404）
```

### CreateProduct

```yaml
tx_required: true
steps:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）を返す
  - authorizer.Authorize（ActionProductCreate / resource=product）で admin 認可を確認する（拒否は 403）
  - price（文字列）を decimal へ解釈し money.NewPrice で非負を検証する（負値は 422 / 非数値は 400。非数値は OpenAPI でも弾く）
  - uuid.New で商品 ID を採番する
  - txm.Do 内で以下を実行する:
      - status_repository.FindByID / category_repository.FindByID で名称を解決する（未存在は整合性異常として ErrInternal=500）
      - product.New で商品エンティティを構築する（負在庫・名称長超過は 422）
      - product_repository.Create で登録する（DB の FK 制約は多層防御の保険。正典の 500 は上のマスタ確認）
  - 生成した Product を ProductView（ImagePath / PublishedAt を含む）へ写像して返す
calls:
  - status_repository.FindByID
  - category_repository.FindByID
  - product_repository.Create
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）
  - 認可拒否は authz 由来の apperror.ErrPermissionDenied（403）
  - 負価格・負在庫・名称長超過は apperror.ErrValidation（422）
  - status_id / category_id 不在は apperror.ErrInternal（500・整合性異常）
```

### UpdateProduct

```yaml
tx_required: true
steps:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）を返す
  - authorizer.Authorize（ActionProductUpdate / resource=product）で admin 認可を確認する（拒否は 403）
  - price が指定された場合のみ decimal へ解釈し money.NewPrice で非負を検証する（負値は 422 / 非数値は 400）
  - txm.Do 内で以下を実行する:
      - product_repository.FindByID で現在の商品を読み込む（未存在は 404。未公開商品も対象）
      - EnsureVersion で要求バージョンと現在バージョンの一致を確認する（不一致は 409。この時点で書き込みへ進まない）
      - statusId / categoryId のいずれかが指定された場合、status / category の参照をペアで再解決する（未指定側も現在の ID でマスタと突合し参照整合を再確認する）。両方とも未指定の場合のみ現在値を据え置き、マスタ問い合わせを行わない
      - 未指定は現在値、null 明示は nil へ解決した確定値で product.Update を呼ぶ（不変条件違反は 422）
      - product_repository.Update で読み込み時点のバージョンを条件に更新し、採番後のバージョンを受け取る（0 行は 409）
  - Product を ProductView へ写像し、Version を採番後の値で上書きして返す
calls:
  - product_repository.FindByID
  - status_repository.FindByID
  - category_repository.FindByID
  - product_repository.Update
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）
  - 認可拒否は authz 由来の apperror.ErrPermissionDenied（403）
  - 未存在は apperror.ErrNotFound（404）
  - バージョン不一致は product.ErrVersionConflict（apperror.ErrConflict → 409）
  - 負価格・負在庫・名称長超過は apperror.ErrValidation（422）
  - status_id / category_id 不在は apperror.ErrInternal（500・整合性異常）
```

> 部分更新の 3 状態（未送信 / null 明示 / 値指定）は `pkg/patch.Field` で表し、usecase が現在値に対して解決する。
> domain は解決後の確定値のみを受け取り、3 状態を知らない。
>
> 409（バージョン不一致）は、`tx.Manager` が透過的にリトライする serialization_failure（ADR-0029）とは別物で、
> 同じ内容の再送では解消しない。クライアントは最新を取得し直してからやり直す必要がある。

### UpdateProductStock

```yaml
tx_required: true
steps:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）を返す
  - authorizer.Authorize（ActionProductStockUpdate / resource=product）で admin 認可を確認する（拒否は 403）
  - txm.Do 内で以下を実行する:
      - product_repository.LockByID で対象商品を悲観ロックして取得する（未存在は 404。未公開商品も対象）
      - product.AdjustStock(delta) で在庫を増減する（増減後が負になる場合は 422。この時点で書き込みへ進まない）
      - product_repository.UpdateStock で取得時点のバージョンを条件に在庫を更新し、採番後のバージョンを受け取る（0 行は 409）
  - Product を ProductView へ写像し、Version を採番後の値で上書きして返す
calls:
  - product_repository.LockByID
  - product_repository.UpdateStock
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）
  - 認可拒否は authz 由来の apperror.ErrPermissionDenied（403）
  - 未存在は apperror.ErrNotFound（404）
  - 取得後に他者が更新していた場合は product.ErrVersionConflict（apperror.ErrConflict → 409）
  - 増減後の在庫が保持できる範囲を外れる場合は product.ErrInvalidQuantity（apperror.ErrValidation → 422）
  - 直列化の待機が解消できない場合は apperror.ErrUnavailable（503）
```

> 行ロックで取得〜更新を直列化するため、並行する在庫の増減は失われず合成される。行ロック競合
> （serialization_failure / deadlock）は `tx.Manager` が透過的にリトライし、枯渇した場合とロック待ちが
> タイムアウトした場合のみ一時障害（503）として露出する。
>
> 409（バージョン不一致）は、直列化を経てもなお取得後の変化を検出した場合の fail-closed な応答であり、
> 一時障害と異なり同じ内容の再送では解消しない。クライアントは最新を取得し直してからやり直す。
>
> 在庫の更新もバージョンを進めるため、更新前のバージョンを持つ部分更新（`UpdateProduct`）は 409 で弾かれる。
> これにより、悲観ロック（同時更新の直列化）と楽観ロック（読み込み後の変化の検出）が同じ行で噛み合う。

### ListLowStockProducts

```yaml
tx_required: false
steps:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）を返す
  - authorizer.Authorize（ActionProductListLowStock / resource=product・所有者なし）で admin 認可を確認する（拒否は 403）
  - paging.NewLimit（lowStockLimitPolicy）で取得件数を正規化する（0 以下は既定値 20、上限超過は 100 へクランプ）
  - product_repository.FindAllLowStock で在庫僅少商品を在庫の少ない順に最大 limit 件取得する
  - 各 Product を ProductView へ写像する（在庫僅少の再判定は行わない。判定は Repository の契約）
calls:
  - product_repository.FindAllLowStock
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）
  - 認可拒否は authz 由来の apperror.ErrPermissionDenied（403）
  - product_repository.FindAllLowStock のエラーをそのまま伝播する
```

> `limit` の既定値は OpenAPI の `default: 20` ではなく usecase の `paging.NewLimit` が与える。
> oapi-codegen は任意クエリパラメータへ既定値を適用せず、未指定は nil のままハンドラへ届くため。
> 範囲外（1 未満 / 100 超）の要求は OpenAPI リクエストバリデータが 400 で弾くため、クランプは
> バリデータを通らない経路に対する二重防御として働く。

### SweepOrphans（ImageGCUsecase）

```yaml
tx_required: false   # DB は読み取りのみ。削除先はオブジェクトストレージで、2 相コミットできない
steps:
  - grace / batchSize が 0 以下なら既定値（24 時間 / 1000 件）に置き換える
  - clock.Now から grace を引いて打ち切り時刻 cutoff を決める
  - 列挙が尽きるまでページを反復
      - object_storage.List で prefix="products/" のオブジェクトを最大 batchSize 件取得する
      - 接頭辞が products/ で、かつ ModifiedAt が cutoff より前のキーだけを候補に絞る（検査件数へ計上）
      - 候補が 0 件ならそのページは照合も削除も行わない
      - product_repository.FilterExistingImagePaths で参照済みのパスを特定し、候補から除外する
      - dryRun でなければ object_storage.Delete で残りを削除し、削除件数を加算（dryRun では対象件数のみ加算）
      - NextCursor が空なら終了。非空なら次ページへ
  - 累計の ImageGCResult を返す
calls:
  - clock.Now
  - object_storage.List
  - product_repository.FilterExistingImagePaths
  - object_storage.Delete
errors:
  - 各依存のエラーを伝播。削除済みのオブジェクトは復元できないため、エラー時もそこまでの累計を
    ImageGCResult に含めて返す
notes:
  - 参照照合が失敗したページでは、オブジェクトを 1 件も削除せずに中断する。
    「照合エラー = 未参照」と倒れるとバケット内の全画像を消すため、唯一の致命的な失敗モードにあたる。
  - 猶予期間が方式の核心。アップロード直後のオブジェクトは「商品作成フォーム記入中でまだ参照されていない」
    正常な状態と区別がつかないため、年齢述語なしでは正常なアップロードを削除してしまう。
  - 列挙が prefix を無視する実装に当たっても商品画像以外を消さないよう、候補を絞る際に接頭辞を再検査する。
```
