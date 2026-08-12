// サンプルAPI削除の対象を宣言する manifest（データ定義）。マーカー除去ロジックは sample-api.ts を参照。
// 拡張時は該当ドメインの paths にパスを追記し、共有ファイルの混在行を
// sample-api マーカーで囲めば、同じコマンドの削除対象に含まれる。
// core 基盤の idempotency_keys（migration 000001）と outbox（migration 000002）はこのマニフェストの削除対象に含まれない。

/** 1 ドメイン分の削除宣言。`paths` はリポジトリルートからの相対パス。 */
export type SampleDomain = {
  description: string;
  paths: readonly string[];
};

export const SAMPLE_DOMAINS: Readonly<Record<string, SampleDomain>> = {
  user: {
    description: "サンプル User API（フルスタック）",
    paths: [
      "internal/domain/user",
      "internal/usecase/user",
      "internal/infrastructure/rdb/repository/user",
      "internal/controller/handler/v1/users",
      "internal/controller/job/usercount",
      "internal/controller/job/userpurge",
      // 退会イベントの consuming 端（worker 実例）と、その broker adapter を組み立てる DI。
      // adapter の生成が di 側にあるのは、controller 層が infrastructure を import できないため。
      "internal/controller/worker/withdrawalarchive",
      "internal/di/module/withdrawalarchive.go",
      "internal/di/module/withdrawalarchive_test.go",
      "internal/integration/v1_users_test.go",
      "internal/integration/v1_users_detail_test.go",
      "internal/integration/v1_users_me_test.go",
      "internal/integration/v1_users_search_test.go",
      "internal/integration/v1_users_feed_test.go",

      // サンプル専用の生成物は再生成で復活しないため明示削除する
      "internal/infrastructure/rdb/sqlc/gen/user_repository.gen.sql.go",
      "database/gen/user_repository.gen.sql",

      "openapi/paths/v1/users.yaml",
      "openapi/paths/v1/users",
      "openapi/components/parameters/user",
      "openapi/components/parameters/search",
      "openapi/components/requests/users",
      "openapi/components/responses/users",
      "openapi/components/schemas/UserBaseInputRequest.yaml",
      "openapi/components/schemas/UserResponse.yaml",

      "database/dml/repository/user",
      "database/migrations/000004_create_users.up.sql",
      "database/migrations/000004_create_users.down.sql",
      "database/migrations/000015_add_users_table_search_text_column.up.sql",
      "database/migrations/000015_add_users_table_search_text_column.down.sql",
      "database/seed/000001_users.sql",
      "database/seed/000002_users_additional.sql",

      "docs/spec/user",
      "docs/spec/user-search",
    ],
  },

  userRole: {
    description:
      "サンプル 認可基盤（user_roles テーブル + user_roles ベース Authorizer）。user サンプルに同梱削除される",
    paths: [
      // Authorizer 実装（本番相当環境に配線。削除時は provideAuthorizer の dev/stg/prd case ごと除去され、以後それらの環境は fail-closed 既定に戻る）
      "internal/infrastructure/authz/userrole",

      "database/migrations/000005_create_roles.up.sql",
      "database/migrations/000005_create_roles.down.sql",
      "database/migrations/000006_create_user_roles.up.sql",
      "database/migrations/000006_create_user_roles.down.sql",
      "database/seed/000003_user_roles.sql",
      "database/seed/000004_user_roles_additional.sql",

      // domain(role.go 等) / infra repository(user_role_repository.go) / dml(select_roles_by_user_id.sql)
      // は user ドメインのディレクトリ配下にあり、user エントリの削除で同時に消えるため個別指定は不要。
      // roles / user_roles を含む共有生成物（schema.gen.sql / models.gen.go / user_repository.gen.*）は
      // 削除後の再生成（db-init → gen-query）で最新化される。
    ],
  },

  userIdentity: {
    description:
      "サンプル 認証アイデンティティ連携（user_identities テーブル + IdentityResolver の DB 実装）。user サンプルに同梱削除され、削除後は DI が passthrough resolver（core）へ差し替わる",
    paths: [
      "internal/infrastructure/auth/useridentity",
      "database/dml/repository/user_identity",
      "database/migrations/000007_create_user_identities.up.sql",
      "database/migrations/000007_create_user_identities.down.sql",
      "database/seed/000005_user_identities.sql",
      "database/seed/000006_user_identities_additional.sql",

      // サンプル専用の生成物は再生成で復活しないため明示削除する
      "internal/infrastructure/rdb/sqlc/gen/user_identity_repository.gen.sql.go",
      "database/gen/user_identity_repository.gen.sql",
    ],
  },

  prefecture: {
    description: "サンプル 都道府県マスタ（user サンプルが住所で参照する依存ドメイン。日本固有データ。GET /v1/prefectures の一覧 API を含む）",
    paths: [
      "internal/domain/prefecture",
      "internal/usecase/prefecture",
      "internal/infrastructure/rdb/repository/prefecture",
      "internal/controller/handler/v1/prefectures",
      "internal/integration/v1_prefectures_test.go",
      "database/dml/repository/prefecture",

      // サンプル専用の生成物は再生成で復活しないため明示削除する
      "internal/infrastructure/rdb/sqlc/gen/prefecture_repository.gen.sql.go",
      "database/gen/prefecture_repository.gen.sql",

      "openapi/paths/v1/prefectures.yaml",
      "openapi/components/responses/prefecture",
      "openapi/components/schemas/PrefectureResponse.yaml",

      "database/migrations/000003_create_prefectures.up.sql",
      "database/migrations/000003_create_prefectures.down.sql",

      "docs/spec/prefecture/usecase.md",
    ],
  },

  product: {
    description: "サンプル 商品ドメイン（GET /v1/products/statuses 商品ステータスマスタ一覧 / GET /v1/products/categories 商品カテゴリマスタ一覧 / GET /v1/products 公開商品一覧〈cursor + フィルタ + keyword + sort〉 / GET /v1/products/{productId} 公開商品詳細〈未存在・非公開は 404 秘匿〉 / PATCH /v1/products/{productId}/stock 在庫の増減〈admin・行ロックで直列化〉 / GET /v1/products/low-stock 在庫僅少一覧〈admin・閾値以下 top-N〉）",
    paths: [
      // ドメイン語彙の唯一の占有者。product と order の双方から使われるが撤去は一括なので、
      // 最初の利用者である product 側に置けば足りる。非サンプルの import 元は存在しない。
      // `internal/domain/lexicon` 自体は入場基準を述べる README ごと残り、占有者ゼロになる。
      "internal/domain/lexicon/money",
      "internal/controller/job/productimagegc",
      "internal/domain/product/status",
      "internal/usecase/product/status",
      "internal/infrastructure/rdb/repository/productstatus",
      "internal/controller/handler/v1/products/statuses",
      "internal/integration/v1_products_statuses_test.go",
      "database/dml/repository/product_status",

      // サンプル専用の生成物は再生成で復活しないため明示削除する
      "internal/infrastructure/rdb/sqlc/gen/product_status_repository.gen.sql.go",
      "database/gen/product_status_repository.gen.sql",

      "openapi/paths/v1/products/statuses.yaml",
      "openapi/components/responses/products/status",
      "openapi/components/schemas/ProductStatusResponse.yaml",
      "openapi/components/schemas/ProductStatusRef.yaml",

      "docs/spec/product-status/domain.md",
      "docs/spec/product-status/usecase.md",

      "internal/domain/product/category",
      "internal/usecase/product/category",
      "internal/infrastructure/rdb/repository/product_category",
      "internal/controller/handler/v1/products/categories",
      "internal/integration/v1_products_categories_test.go",
      "database/dml/repository/product_category",

      "internal/infrastructure/rdb/sqlc/gen/product_category_repository.gen.sql.go",
      "database/gen/product_category_repository.gen.sql",

      "openapi/paths/v1/products/categories.yaml",
      "openapi/components/responses/products/category",
      "openapi/components/schemas/ProductCategoryResponse.yaml",
      "openapi/components/schemas/ProductCategoryRef.yaml",

      "docs/spec/product-category/domain.md",
      "docs/spec/product-category/usecase.md",

      "internal/domain/product",
      "internal/usecase/product",
      "storage/seed/products",

      "internal/infrastructure/rdb/repository/product",
      "internal/usecase/product/ranking",
      "internal/infrastructure/rdb/query_service/product",
      "internal/controller/handler/v1/products",
      "internal/integration/v1_products_test.go",
      "internal/integration/v1_products_detail_test.go",
      "internal/integration/v1_products_stock_test.go",
      "internal/integration/v1_products_ranking_test.go",
      "internal/integration/v1_products_low_stock_test.go",
      "database/dml/repository/product",
      "database/dml/query_service/product",

      "internal/infrastructure/rdb/sqlc/gen/product_repository.gen.sql.go",
      "database/gen/product_repository.gen.sql",
      "internal/infrastructure/rdb/sqlc/gen/product_query_service.gen.sql.go",
      "database/gen/product_query_service.gen.sql",

      "openapi/paths/v1/products.yaml",
      "openapi/paths/v1/products/productId.yaml",
      "openapi/paths/v1/products/productId/stock.yaml",
      "openapi/paths/v1/products/images.yaml",
      "openapi/paths/v1/products/ranking.yaml",
      "openapi/paths/v1/products/low-stock.yaml",
      "openapi/components/parameters/product",
      "openapi/components/requests/products",
      "openapi/components/responses/products",
      "openapi/components/schemas/products",
      "openapi/components/schemas/ProductResponse.yaml",
      "openapi/components/schemas/ProductImageResponse.yaml",

      "docs/spec/product",
      "docs/spec/product-ranking",

      "database/migrations/000008_create_product_statuses.up.sql",
      "database/migrations/000008_create_product_statuses.down.sql",
      "database/migrations/000009_create_product_categories.up.sql",
      "database/migrations/000009_create_product_categories.down.sql",
      "database/migrations/000010_create_products.up.sql",
      "database/migrations/000010_create_products.down.sql",
      "database/migrations/000011_create_product_images.up.sql",
      "database/migrations/000011_create_product_images.down.sql",
      "database/seed/000007_products_electronic_equipment_01.sql",
      "database/seed/000008_products_electronic_equipment_02.sql",
      "database/seed/000009_products_electronic_equipment_03.sql",
      "database/seed/000010_products_books_01.sql",
      "database/seed/000011_products_books_02.sql",
      "database/seed/000012_products_clothing_01.sql",
      "database/seed/000013_products_clothing_02.sql",
      "database/seed/000014_products_clothing_03.sql",
      "database/seed/000015_products_clothing_04.sql",
      "database/seed/000016_products_food_01.sql",
      "database/seed/000017_products_food_02.sql",
      "database/seed/000018_products_food_03.sql",
      "database/seed/000019_products_food_04.sql",
      "database/seed/000020_products_food_05.sql",
      "database/seed/000021_products_food_06.sql",
      "database/seed/000022_products_food_07.sql",
      "database/seed/000023_products_food_08.sql",
      "database/seed/000024_products_food_09.sql",
      "database/seed/000025_products_furniture.sql",
    ],
  },

  order: {
    description: "サンプル 購入 API（POST /v1/purchases・CommandService 正例 / GET /v1/purchases/shippable・単一集約 Domain Service 正例。フルスタック）",
    paths: [
      // DB スキーマ
      "database/migrations/000012_create_purchase_statuses.up.sql",
      "database/migrations/000012_create_purchase_statuses.down.sql",
      "database/migrations/000013_create_purchases.up.sql",
      "database/migrations/000013_create_purchases.down.sql",
      "database/migrations/000014_create_purchase_details.up.sql",
      "database/migrations/000014_create_purchase_details.down.sql",
      "database/seed/000026_purchases_01.sql",
      "database/seed/000027_purchases_02.sql",
      "database/seed/000028_purchases_03.sql",
      "database/seed/000029_purchases_04.sql",
      "database/seed/000030_purchases_05.sql",
      "database/seed/000031_purchases_06.sql",
      "database/seed/000032_purchase_details_01.sql",
      "database/seed/000033_purchase_details_02.sql",
      "database/seed/000034_purchase_details_03.sql",
      "database/seed/000035_purchase_details_04.sql",
      "database/seed/000036_purchase_details_05.sql",
      "database/seed/000037_purchase_details_06.sql",
      "database/seed/000038_purchase_details_07.sql",
      "database/seed/000039_purchase_details_08.sql",
      "database/seed/000040_purchase_details_09.sql",
      // Go 各層
      "internal/domain/purchase",
      // 在籍と購入の進行状態にまたがる規則を持つドメインサービス。purchase 集約を参照するため
      // purchase と生死を共にする（残すと import 先を失う）。
      "internal/domain/service/membership",
      // 発送待ち購入のまとめ判定を持つドメインサービス。purchase 集約のみを参照する単一集約サービスで、
      // やはり purchase と生死を共にする。
      "internal/domain/service/dispatch",
      "internal/usecase/purchase",
      // checkout は purchase と exchangerate を束ねる合成 Usecase なので、両者と生死を共にする
      "internal/usecase/checkout",
      "internal/infrastructure/rdb/command_service/purchase",
      "internal/infrastructure/rdb/repository/purchase",
      "internal/infrastructure/rdb/query_service/purchase",
      "internal/controller/handler/v1/purchases",
      "internal/integration/v1_purchases_test.go",
      "internal/integration/v1_purchases_get_test.go",
      "internal/integration/v1_purchases_detail_test.go",
      "internal/integration/v1_purchases_cancel_test.go",
      "internal/integration/v1_purchases_pay_test.go",
      "internal/integration/v1_purchases_ship_test.go",
      "internal/integration/v1_purchases_deliver_test.go",
      "internal/integration/v1_purchases_shippable_test.go",
      "internal/integration/v1_users_me_purchases_summary_test.go",
      // DML
      "database/dml/command_service/purchase",
      "database/dml/repository/purchase",
      "database/dml/query_service/purchase",
      // 生成物（sqlc。openapi.yaml / DI 各ファイルの登録はマーカーで除去される）
      "internal/infrastructure/rdb/sqlc/gen/purchase_command_service.gen.sql.go",
      "internal/infrastructure/rdb/sqlc/gen/purchase_repository.gen.sql.go",
      "internal/infrastructure/rdb/sqlc/gen/purchase_query_service.gen.sql.go",
      "database/gen/purchase_command_service.gen.sql",
      "database/gen/purchase_repository.gen.sql",
      "database/gen/purchase_query_service.gen.sql",
      // OpenAPI
      "openapi/paths/v1/purchases.yaml",
      "openapi/paths/v1/purchases",
      "openapi/components/requests/purchases",
      "openapi/components/responses/purchases",
      "openapi/components/parameters/purchase",
      "openapi/components/schemas/PurchaseResponse.yaml",
      "openapi/components/schemas/PurchaseCancelResponse.yaml",
      "openapi/components/schemas/PurchasePayResponse.yaml",
      "openapi/components/schemas/PurchaseShipResponse.yaml",
      "openapi/components/schemas/PurchaseDeliverResponse.yaml",
      "openapi/components/schemas/PurchaseSummaryResponse.yaml",
      "openapi/components/schemas/PurchaseDetailResponse.yaml",
      "openapi/components/schemas/PurchaseGetDetailResponse.yaml",
      "openapi/components/schemas/PurchaseDetailItemResponse.yaml",
      "openapi/components/schemas/PurchaseDetailInput.yaml",
      "openapi/components/schemas/PurchaseStatusRef.yaml",
      "openapi/components/schemas/PurchaseStatusBreakdownResponse.yaml",
      // spec
      "docs/spec/purchase",
    ],
  },

  cart: {
    description:
      "サンプル カート API（/v1/carts/me・ゲストカートの所有者確定 / 明細ごとの再評価 / 期限切れ掃除ジョブ）。現時点は spec のみで、実装が入るたびに paths を追記する",
    paths: [
      "database/migrations/000016_create_carts.up.sql",
      "database/migrations/000016_create_carts.down.sql",
      "database/migrations/000017_create_cart_items.up.sql",
      "database/migrations/000017_create_cart_items.down.sql",

      // spec
      "docs/spec/cart",
    ],
  },

  exchangeRate: {
    description: "サンプル 為替レート換算 API（GET /v1/exchange-rates。外部 gateway + TTL キャッシュ decorator + reference_amount + degrade）",
    paths: [
      "internal/usecase/exchangerate",
      "internal/usecase/boundary/exchangerate",
      "internal/infrastructure/webapi/exchangerate",
      "internal/controller/handler/v1/exchangerate",
      "internal/integration/v1_exchange_rates_test.go",

      "openapi/paths/v1/exchange-rates.yaml",
      "openapi/components/responses/exchange-rates",
      "openapi/components/schemas/exchange-rates",

      // internal/usecase/tools/money は paging/search と同様の汎用 usecase ツールであり
      // サンプル削除対象に含めない（後続 purchases も再利用する恒久ヘルパ）。

      // spec
      "docs/spec/exchange-rate",
    ],
  },

  address: {
    description: "サンプル 郵便番号住所補完 API（GET /v1/addresses。外部 gateway 経由の zipcloud lookup + prefecture_id 変換 + degrade〈外部障害時 200 + 空候補〉）",
    paths: [
      "internal/usecase/address",
      "internal/usecase/boundary/address",
      "internal/infrastructure/webapi/address",
      "internal/controller/handler/v1/addresses",
      "internal/integration/v1_addresses_test.go",

      "openapi/paths/v1/addresses.yaml",
      "openapi/components/parameters/address",
      "openapi/components/responses/addresses",
      "openapi/components/schemas/addresses",

      "docs/spec/address/usecase.md",
    ],
  },

  dashboard: {
    description:
      "サンプル admin ダッシュボード集計 API（GET /v1/dashboard/summary・backend 合成/1画面1API。購入と商品を横断する集計 read）",
    paths: [
      // Go 各層
      "internal/usecase/dashboard",
      "internal/infrastructure/rdb/query_service/dashboard",
      "internal/controller/handler/v1/dashboard",
      "internal/integration/v1_dashboard_summary_test.go",
      // DML
      "database/dml/query_service/dashboard",
      // 生成物（sqlc。openapi.yaml / DI 各ファイルの登録はマーカーで除去される）
      "internal/infrastructure/rdb/sqlc/gen/dashboard_query_service.gen.sql.go",
      "database/gen/dashboard_query_service.gen.sql",
      // OpenAPI
      "openapi/paths/v1/dashboard",
      "openapi/components/parameters/dashboard",
      "openapi/components/responses/dashboard",
      "openapi/components/schemas/dashboard",

      "docs/spec/dashboard",
    ],
  },

  outboxBroker: {
    description:
      "outbox を SQS 互換 broker へ向ける配線。engine / seam / SQS adapter は送受信とも core、" +
      "ローカル broker も object storage の Garage と同じくローカルインフラとして残し、" +
      "core から adapter を参照する配線だけを削除対象にする" +
      "（削除後の結合をサンプル追加前と同一に保つ。ADR-0050 (broker-sdk-isolation-measured-as-coupling) が broker SDK の隔離を" +
      "リンクではなく結合で測る、と述べているのがこれ）。" +
      "object storage と揃うのは adapter / ローカルインフラ / config を core に置く点までで、" +
      "判別子の 1 分岐として配線する構造は queue 側だけのもの（object storage に選択肢は無い）。",
    paths: [
      "internal/infrastructure/publisher/queue_config.go",
      "internal/infrastructure/publisher/queue_config_test.go",
    ],
  },

  sampleTooling: {
    description: "サンプル削除ツール自身（削除完了後は不要）",
    // ディレクトリごと挙げる。ファイルを 1 本ずつ列挙すると、判定モジュールを足したときに
    // 漏れて、消えたはずの削除ツールの一部だけが利用者のリポジトリへ居座る。
    paths: ["scripts/setup/remove-sample-api"],
  },
};

// サンプル行が残す行と混在するため、行単位でマーカー除去する共有ファイル。
/**
 * 走査から外すディレクトリ名。依存の取得物と VCS の内部。
 *
 * @remarks
 * remove-boilerplate-identity 側にも同じ宣言があります。**共通化していません**。2 つの撤去は
 * 起爆の契機が違い（あちらはセットアップ、こちらはサンプル削除）、どちらが先に走るかも順序が
 * 決まっていないため、共有モジュールへ寄せると先に走ったほうがもう一方の足元を持って行きうる
 * 依存が増えます。数行なので、重複を許して各ツールが自分の分を抱えます。
 */
export const EXCLUDED_DIRECTORIES: ReadonlySet<string> = new Set([
  ".git",
  "node_modules",
  "vendor",
]);

/** 走査から外す相対パス接頭辞。いずれも生成物で、除去しても再生成で戻る。 */
export const EXCLUDED_PATH_PREFIXES: readonly string[] = [
  "docs/portal/guides/",
  "docs/coverage/",
  "docs/db-schema/",
  "docs/godoc/",
  "graphify-out/",
  "tmp/",
];

/**
 * マーカー文字列を「データ・散文」として持つファイル。走査の対象から外す。
 *
 * @remarks
 * 除去はリポジトリを走査するので、対象ファイルの一覧は要りません（一覧はその外側にマーカーを
 * 書けてしまい、しかも取りこぼしが無言だったため廃しました）。代わりに要るのがこの逆向きの
 * 宣言です。`sample-api` は、`boilerplate-only` と違って**マーカーの形そのものを教材・
 * フィクスチャ・規約説明が本文に持っている**ため、素朴に走査すると、マーカーではないものを
 * マーカーとして刈り取ってしまいます。
 *
 * 一覧の取りこぼしが無言だったのに対し、こちらの取りこぼしは声を出します。宣言し忘れた
 * ファイルは内容が壊れ、`sample-removal-check.yaml` の `make test` / `md-markdownlint-ci` /
 * `go build` が落ちます。
 */
export const MARKER_LITERAL_FILES: readonly string[] = [
  // マーカー除去そのもののテスト。入力として `# sample-api:begin` を持つ。
  "scripts/setup/lib/markers.test.ts",
  // 前提検査もマーカー除去を行うため、両名前空間を入力として持つ。あちらは撤去の契機が違う
  // （ボイラープレート撤去で丸ごと消える）ので、こちらからは通常のファイルとして見える。
  "scripts/premise-lint/rules.test.ts",
  // Go の文字列リテラルとして `// sample-api:line` を組み立て、走査器の挙動を検査している。
  "internal/architest/bindhandler_di_parity_test.go",
  // 教材。マーカーの書き方をコード例として示している。
  "docs/tutorial/build-user-feature.md",
  "docs/ja/tutorial/build-user-feature.ja.md",
  // マーカーの書き方を説明している散文。1 行に begin と end が同居するため、
  // 素通しすると「閉じられていない begin」として除去が中断する。
  "docs/get-started/setup-repository.md",
  "docs/ja/get-started/setup-repository.ja.md",
  // マーカー規約を説明している散文。
  ".claude/agents/drift-detector-glossary.md",
  ".codex/agents/drift-detector-glossary.toml",
];

// 削除後に再生成・整形・検証するための make ターゲット（順番に実行）。
export const BUILD_STEPS: readonly string[] = ["gen-api", "gen-query", "fix", "lint"];
