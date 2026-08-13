package validator

import (
	"fmt"
	"sort"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/controller/httpstack/oapi/validator/gen"
)

// publicOperations は認証不要（security 未宣言または security: [] の明示）が
// 意図どおりである公開 operation の許可リスト。キーは "METHOD /path" 形式。
// 新しい operation はここに登録しない限り認証必須とみなされるため、
// security の宣言漏れはテストで検出される。認証不要が正しい場合のみ、
// 公開理由のコメントとともに追加すること。
var publicOperations = map[string]string{
	"GET /health":                         "ヘルスチェック（監視系は認証不要）",
	"GET /healthz":                        "liveness チェック（監視系は認証不要）",
	"GET /ready":                          "readiness チェック（監視系は認証不要）",
	"GET /version":                        "バージョン情報の公開エンドポイント",
	"GET /_internal/types/error-response": "ErrorResponse 型生成用の内部エンドポイント（コード生成専用）",
	"GET /v1/prefectures":                 "都道府県マスタの公開 API",
	"GET /v1/products/statuses":           "商品ステータスマスタの公開 API",
	"GET /v1/products/categories":         "商品カテゴリマスタの公開 API",
	"GET /v1/products":                    "商品一覧の公開 API",
	"GET /v1/products/count":              "商品検索の一致件数を返す公開 API",
	"GET /v1/products/{productId}":        "商品詳細の公開 API",
	"GET /v1/products/ranking":            "商品売上ランキングの公開 API",
	"GET /v1/exchange-rates":              "為替レート換算の公開 API",
	"GET /v1/addresses":                   "郵便番号からの住所補完の公開 API",
}

// effectiveSecurity は OpenAPI の意味論に従い operation に適用される
// security 要件を返す。operation 側の宣言があればそれが優先され、
// 無ければトップレベル（グローバル）の security が適用される。
func effectiveSecurity(spec *openapi3.T, op *openapi3.Operation) openapi3.SecurityRequirements {
	if op.Security != nil {
		return *op.Security
	}
	return spec.Security
}

// requiresAuth は security 要件が「認証必須」を意味するかを判定する。
// 要件が空なら公開 operation。要件リストは OR 結合のため、空の要件
// オブジェクト {} が1つでも含まれると匿名アクセスを許すので認証必須とはみなさない。
func requiresAuth(reqs openapi3.SecurityRequirements) bool {
	if len(reqs) == 0 {
		return false
	}
	for _, req := range reqs {
		if len(req) == 0 {
			return false
		}
	}
	return true
}

func TestSecurityDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("許可リスト外のすべてのoperationが認証必須のsecurityを宣言している", func(t *testing.T) {
			t.Parallel()

			spec, err := gen.GetSpec()
			require.NoError(t, err)
			require.NotNil(t, spec.Paths)

			seen := make(map[string]bool, spec.Paths.Len())
			for path, item := range spec.Paths.Map() {
				for method, op := range item.Operations() {
					key := fmt.Sprintf("%s %s", method, path)
					seen[key] = true
					if requiresAuth(effectiveSecurity(spec, op)) {
						assert.NotContains(t, publicOperations, key,
							"%s は認証必須のため許可リストのエントリが不要です。publicOperations から削除してください", key)
						continue
					}
					assert.Contains(
						t,
						publicOperations,
						key,
						"%s に認証必須の security 宣言がありません。認証が必要なら OpenAPI 定義に security: [BearerAuth] を追加し、意図的な公開 API なら publicOperations に理由コメント付きで追加してください",
						key,
					)
				}
			}

			staleEntries := make([]string, 0, len(publicOperations))
			for key := range publicOperations {
				if !seen[key] {
					staleEntries = append(staleEntries, key)
				}
			}
			sort.Strings(staleEntries)
			assert.Empty(t, staleEntries,
				"許可リストに spec に存在しない operation があります。publicOperations から削除してください")
		})
	})
}
