// サンプルAPI削除の対象を宣言する manifest（データ定義）。マーカー除去ロジックは sample-api.mjs を参照。
// 拡張時は該当ドメインの paths にパスを追記し、共有ファイルの混在行を
// sample-api マーカーで囲めば、同じコマンドの削除対象に含まれる。
// core 基盤の idempotency_keys（migration 000001）と outbox（migration 000002）はこのマニフェストの削除対象に含まれない。

export const SAMPLE_DOMAINS = {
  user: {
    description: "サンプル User API（フルスタック）",
    paths: [
      "internal/domain/user",
      "internal/usecase/user",
      "internal/infrastructure/rdb/repository/user",
      "internal/infrastructure/rdb/query_service/user",
      "internal/controller/handler/v1/users",
      "internal/controller/job/usercount",
      "internal/integration/v1_users_test.go",
      "internal/integration/v1_users_detail_test.go",
      "internal/integration/v1_users_me_test.go",
      "internal/integration/v1_users_search_test.go",
      "internal/integration/v1_users_feed_test.go",

      // サンプル専用の生成物は再生成で復活しないため明示削除する
      "internal/infrastructure/rdb/sqlc/gen/user_repository.gen.sql.go",
      "internal/infrastructure/rdb/sqlc/gen/user_query_service.gen.sql.go",
      "database/gen/user_repository.gen.sql",
      "database/gen/user_query_service.gen.sql",

      "openapi/paths/v1/users.yaml",
      "openapi/paths/v1/users",
      "openapi/components/parameters/user",
      "openapi/components/parameters/search",
      "openapi/components/requests/users",
      "openapi/components/responses/users",
      "openapi/components/schemas/UserBaseInputRequest.yaml",
      "openapi/components/schemas/UserResponse.yaml",

      "database/dml/repository/user",
      "database/dml/query_service/user",
      "database/migrations/000004_create_users.up.sql",
      "database/migrations/000004_create_users.down.sql",
      "database/migrations/000014_add_users_table_search_text_column.up.sql",
      "database/migrations/000014_add_users_table_search_text_column.down.sql",
      "database/seed/000001_users.sql",

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
      "database/seed/000008_user_roles.sql",

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
      "database/seed/000009_user_identities.sql",

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
    description: "サンプル 商品ドメイン（product_statuses は GET /v1/product-statuses の一覧 API を含む Go 層まで実装済。product_categories / products は DB スタブのみ）",
    paths: [
      "internal/domain/product/status",
      "internal/usecase/product/status",
      "internal/infrastructure/rdb/repository/productstatus",
      "internal/controller/handler/v1/product-statuses",
      "internal/integration/v1_product_statuses_test.go",
      "database/dml/repository/product_status",

      // サンプル専用の生成物は再生成で復活しないため明示削除する
      "internal/infrastructure/rdb/sqlc/gen/product_status_repository.gen.sql.go",
      "database/gen/product_status_repository.gen.sql",

      "openapi/paths/v1/product-statuses.yaml",
      "openapi/components/responses/product-status",
      "openapi/components/schemas/ProductStatusResponse.yaml",

      "docs/spec/product-status/domain.md",
      "docs/spec/product-status/usecase.md",

      "database/migrations/000008_create_product_statuses.up.sql",
      "database/migrations/000008_create_product_statuses.down.sql",
      "database/migrations/000009_create_product_categories.up.sql",
      "database/migrations/000009_create_product_categories.down.sql",
      "database/migrations/000010_create_products.up.sql",
      "database/migrations/000010_create_products.down.sql",
      "database/seed/000002_products_electronic_equipment.sql",
      "database/seed/000003_products_books.sql",
      "database/seed/000004_products_clothing.sql",
      "database/seed/000005_products_food_first.sql",
      "database/seed/000006_products_food_latter.sql",
      "database/seed/000007_products_furniture.sql",
    ],
  },

  order: {
    description: "サンプル 注文ドメイン（現状: DB スタブのみ。Go 層実装時に追記）",
    paths: [
      "database/migrations/000011_create_purchase_statuses.up.sql",
      "database/migrations/000011_create_purchase_statuses.down.sql",
      "database/migrations/000012_create_purchases.up.sql",
      "database/migrations/000012_create_purchases.down.sql",
      "database/migrations/000013_create_purchase_details.up.sql",
      "database/migrations/000013_create_purchase_details.down.sql",
    ],
  },

  sampleTooling: {
    description: "サンプル削除ツール自身（削除完了後は不要）",
    paths: [
      "scripts/setup/remove-sample-api.mjs",
      "scripts/setup/lib/sample-api.mjs",
      "scripts/setup/lib/sample-manifest.mjs",
    ],
  },
}

// サンプル行が残す行と混在するため、行単位でマーカー除去する共有ファイル。
export const MARKER_FILES = [
  "openapi/openapi.yaml",
  "internal/di/module/core/auth.go",
  "internal/di/module/controller.go",
  "internal/di/module/usecase.go",
  "internal/di/module/persistence.go",
  "internal/di/module/authz.go",
  "internal/di/module/authz_test.go",
  "internal/di/module/job.go",
  ".makefiles/github/operation/setup-repository.mk",
  ".makefiles/README.md",
  ".makefiles/README.ja.md",
  "scripts/README.md",
  "scripts/README.ja.md",
]

// 削除後に再生成・整形・検証するための make ターゲット（順番に実行）。
export const BUILD_STEPS = ["gen-api", "gen-query", "fix", "lint"]
