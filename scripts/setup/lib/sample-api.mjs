// サンプルAPI削除の対象を宣言する manifest と、マーカー除去ロジック。
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
      "database/migrations/000013_users_table_search_text_column.up.sql",
      "database/migrations/000013_users_table_search_text_column.down.sql",
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

  prefecture: {
    description: "サンプル 都道府県マスタ（user サンプルが住所で参照する依存ドメイン。日本固有データ）",
    paths: [
      "internal/domain/prefecture",
      "internal/infrastructure/rdb/repository/prefecture",
      "database/dml/repository/prefecture",

      // サンプル専用の生成物は再生成で復活しないため明示削除する
      "internal/infrastructure/rdb/sqlc/gen/prefecture_repository.gen.sql.go",
      "database/gen/prefecture_repository.gen.sql",

      "database/migrations/000003_create_prefectures.up.sql",
      "database/migrations/000003_create_prefectures.down.sql",
    ],
  },

  product: {
    description: "サンプル 商品ドメイン（現状: DB スタブのみ。Go 層実装時に追記）",
    paths: [
      "database/migrations/000007_create_product_statuses.up.sql",
      "database/migrations/000007_create_product_statuses.down.sql",
      "database/migrations/000008_create_product_categories.up.sql",
      "database/migrations/000008_create_product_categories.down.sql",
      "database/migrations/000009_create_products.up.sql",
      "database/migrations/000009_create_products.down.sql",
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
      "database/migrations/000010_create_purchase_statuses.up.sql",
      "database/migrations/000010_create_purchase_statuses.down.sql",
      "database/migrations/000011_create_purchases.up.sql",
      "database/migrations/000011_create_purchases.down.sql",
      "database/migrations/000012_create_purchase_details.up.sql",
      "database/migrations/000012_create_purchase_details.down.sql",
    ],
  },

  sampleTooling: {
    description: "サンプル削除ツール自身（削除完了後は不要）",
    paths: ["scripts/setup/remove-sample-api.mjs", "scripts/setup/lib/sample-api.mjs"],
  },
}

// サンプル行が残す行と混在するため、行単位でマーカー除去する共有ファイル。
export const MARKER_FILES = [
  "openapi/openapi.yaml",
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

// マーカーはコメント（// / # / <!-- のいずれか）に書かれる前提。コメント記号を必須にして、
// 文字列リテラルやドキュメント本文中の同一トークンを誤って拾わないようにする。
// markdown（<!-- ... -->）コメント行も対象に含める。
const BLOCK_BEGIN = /(?:\/\/|#|<!--)\s*sample-api:begin\b/
const BLOCK_END = /(?:\/\/|#|<!--)\s*sample-api:end\b/
const LINE_MARKER = /(?:\/\/|#|<!--)\s*sample-api:line\b/

// replace マーカー: `replace-begin`〜`replace-with` の有効行（サンプル在時に生きるコード）を除去し、
// `replace-with`〜`replace-end` の差し替え行（`// =` / `# =` でコメント化された退避コード）をアンコメントして残す。
// 削除後にだけ有効化したい代替コード（例: 削除後のみ有効な既定値やテストケース）を、単純な行/ブロック除去では
// 表現できない「置換」として扱うための仕組み。退避コメントは `//` 直後にスペースを置く（gocritic 準拠）。
const REPLACE_BEGIN = /(?:\/\/|#|<!--)\s*sample-api:replace-begin\b/
const REPLACE_WITH = /(?:\/\/|#|<!--)\s*sample-api:replace-with\b/
const REPLACE_END = /(?:\/\/|#|<!--)\s*sample-api:replace-end\b/
// 差し替え行の退避コメント。先頭の空白（インデント）は保持し、`//`/`#` と `=` マーカー・直後の空白1つだけ剥がす。
const REPLACE_CONTENT = /^(\s*)(?:\/\/|#)\s*=\s?(.*)$/

// `sample-api:begin`〜`sample-api:end` で囲まれた行と、行末に `sample-api:line` を持つ行を除去する。
// さらに `sample-api:replace-begin`/`replace-with`/`replace-end` による置換にも対応する。
// ネストにも対応するため depth カウンターで管理し、対応の取れないマーカーは throw する。
export function stripSampleMarkers(content) {
  const lines = content.split("\n")
  const out = []
  let depth = 0
  let removed = 0
  // 0: 置換外 / 1: 有効側（除去中） / 2: 差し替え側（アンコメント中）
  let replaceState = 0

  for (const line of lines) {
    if (REPLACE_BEGIN.test(line)) {
      if (replaceState !== 0) {
        throw new Error("sample-api:replace ブロックは入れ子にできません。")
      }
      replaceState = 1
      removed++
      continue
    }
    if (REPLACE_WITH.test(line)) {
      if (replaceState !== 1) {
        throw new Error("sample-api:replace-with に対応する sample-api:replace-begin がありません。")
      }
      replaceState = 2
      removed++
      continue
    }
    if (REPLACE_END.test(line)) {
      if (replaceState === 0) {
        throw new Error("sample-api:replace-end に対応する sample-api:replace-begin がありません。")
      }
      replaceState = 0
      removed++
      continue
    }
    if (replaceState === 1) {
      // 有効側（サンプル在時のコード）は除去する。
      removed++
      continue
    }
    if (replaceState === 2) {
      // 差し替え側は退避コメントをアンコメントして残す。
      const matched = REPLACE_CONTENT.exec(line)
      if (matched === null) {
        throw new Error(`sample-api:replace-with 〜 replace-end の行は //= または #= で始めてください: ${line}`)
      }
      out.push(matched[1] + matched[2])
      continue
    }

    if (BLOCK_BEGIN.test(line)) {
      depth++
      removed++
      continue
    }
    if (BLOCK_END.test(line)) {
      if (depth === 0) {
        throw new Error("sample-api:end に対応する sample-api:begin が見つかりません。")
      }
      depth--
      removed++
      continue
    }
    if (depth > 0 || LINE_MARKER.test(line)) {
      removed++
      continue
    }
    out.push(line)
  }

  if (depth > 0) {
    throw new Error("sample-api:begin に対応する sample-api:end が見つかりません。")
  }
  if (replaceState !== 0) {
    throw new Error("sample-api:replace-begin に対応する sample-api:replace-end が見つかりません。")
  }

  return { content: out.join("\n"), removed }
}
