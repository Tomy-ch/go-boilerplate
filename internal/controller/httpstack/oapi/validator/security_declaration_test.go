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

// bearerSchemeName は、Bearer トークンによる認証を宣言する securityScheme の名前。
const bearerSchemeName = "BearerAuth"

const (
	// authRequired は、資格情報を必ず要求する。
	authRequired securityKind = iota
	// fullyPublic は、資格情報を受け取る経路を持たない。
	fullyPublic
	// optionalAuth は、資格情報が無くても通すが、提示された資格情報が無効なら拒否する。
	optionalAuth
)

// publicOperations は完全公開（security 未宣言または security: [] の明示）が
// 意図どおりである operation の許可リスト。キーは "METHOD /path" 形式。
// ここに載る operation は資格情報を受け取らないため、無効な資格情報という状態自体が存在しない。
// 資格情報の有無で挙動が変わる operation は optionalAuthOperations が持つ。
//
// 新しい operation はどちらのリストにも載せない限り認証必須とみなされるため、
// security の宣言漏れはテストで検出される。完全公開が正しい場合のみ、
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

// optionalAuthOperations は任意認証（security に BearerAuth と空要件の両方を宣言）が
// 意図どおりである operation の許可リスト。キーは "METHOD /path" 形式。
//
// ここに載る operation の意図は「資格情報が無ければ匿名として 200、資格情報が提示されて
// それが無効なら 401」であり、完全公開とは失敗時の意味論が異なる
// （ADR-0019 (optional-authentication-fail-closed)）。
// 同じ一覧に並べると security の宣言を読んでも姿勢が読み取れなくなるため、リストを分けている。
var optionalAuthOperations = map[string]string{
	"GET /v1/carts/me": "カートはゲストと認証済みユーザーの双方が主体になれる（未認証でも 200、無効な資格情報は 401）",
	"PUT /v1/carts/me/items/{productId}": "ゲストも明細を投入でき、その場でカートとセッショントークンが作られる" +
		"（未認証でも 200、無効な資格情報は 401）",
	"DELETE /v1/carts/me/items/{productId}": "ゲストも自分の明細を取り除ける" +
		"（未認証でも 204、無効な資格情報は 401）",
}

// securityKind は、security 要件が表す認証の姿勢。
type securityKind int

// effectiveSecurity は OpenAPI の意味論に従い operation に適用される
// security 要件を返す。operation 側の宣言があればそれが優先され、
// 無ければトップレベル（グローバル）の security が適用される。
func effectiveSecurity(spec *openapi3.T, op *openapi3.Operation) openapi3.SecurityRequirements {
	if op.Security != nil {
		return *op.Security
	}
	return spec.Security
}

// classifySecurity は security 要件を 3 つの姿勢へ分類する。
// 要件リストは OR 結合で、空の要件オブジェクト {} は必ず満たされるため匿名アクセスを許す。
// 空の要件しか無い宣言は資格情報を受け取る余地が無く、完全公開と区別できない。
func classifySecurity(reqs openapi3.SecurityRequirements) securityKind {
	if len(reqs) == 0 {
		return fullyPublic
	}
	if emptyRequirementIndex(reqs) < 0 {
		return authRequired
	}
	if len(reqs) == 1 {
		return fullyPublic
	}
	return optionalAuth
}

// declaresBearer は、要件リストのいずれかが BearerAuth を名指ししているかを返す。
func declaresBearer(reqs openapi3.SecurityRequirements) bool {
	for _, req := range reqs {
		if _, ok := req[bearerSchemeName]; ok {
			return true
		}
	}
	return false
}

// emptyRequirementIndex は、匿名アクセスを許す空の要件が要件リストの何番目にあるかを返す。
// 空の要件が無い場合は -1。
func emptyRequirementIndex(reqs openapi3.SecurityRequirements) int {
	for i, req := range reqs {
		if len(req) == 0 {
			return i
		}
	}
	return -1
}

// staleEntries は、spec に存在しない operation を指したままの許可リストのキーを返す。
func staleEntries(allowList map[string]string, seen map[string]bool) []string {
	stale := make([]string, 0, len(allowList))
	for key := range allowList {
		if !seen[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale)
	return stale
}

// assertAllowListMembership は、operation の姿勢に対応する許可リストにだけ載っていることを表明する。
func assertAllowListMembership(t *testing.T, key string, kind securityKind) {
	t.Helper()

	switch kind {
	case authRequired:
		assert.NotContains(t, publicOperations, key,
			"%s は認証必須のため許可リストのエントリが不要です。publicOperations から削除してください", key)
		assert.NotContains(t, optionalAuthOperations, key,
			"%s は認証必須のため許可リストのエントリが不要です。optionalAuthOperations から削除してください", key)
	case fullyPublic:
		assert.Contains(t, publicOperations, key,
			"%s に認証必須の security 宣言がありません。認証が必要なら OpenAPI 定義に security: [BearerAuth] を追加し、"+
				"意図的な公開 API なら publicOperations に理由コメント付きで追加してください", key)
		assert.NotContains(t, optionalAuthOperations, key,
			"%s は資格情報を受け取らないため任意認証ではありません。publicOperations へ移してください", key)
	case optionalAuth:
		assert.Contains(t, optionalAuthOperations, key,
			"%s は任意認証（BearerAuth と空要件の両方を宣言）です。意図どおりなら optionalAuthOperations へ "+
				"理由コメント付きで追加してください（ADR-0019 (optional-authentication-fail-closed)）", key)
		assert.NotContains(t, publicOperations, key,
			"%s は無効な資格情報を 401 で拒否するため完全公開ではありません。optionalAuthOperations へ移してください", key)
	}
}

// assertAllowListsCoverSpec は、spec の全 operation が姿勢に対応する許可リストにだけ載っていること、
// および許可リストに実体の無いエントリが残っていないことを表明する。
func assertAllowListsCoverSpec(t *testing.T) {
	t.Helper()

	spec, err := gen.GetSpec()
	require.NoError(t, err)
	require.NotNil(t, spec.Paths)

	seen := make(map[string]bool, spec.Paths.Len())
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			key := fmt.Sprintf("%s %s", method, path)
			seen[key] = true
			assertAllowListMembership(t, key, classifySecurity(effectiveSecurity(spec, op)))
		}
	}

	assert.Empty(t, staleEntries(publicOperations, seen),
		"publicOperations に spec に存在しない operation があります。削除してください")
	assert.Empty(t, staleEntries(optionalAuthOperations, seen),
		"optionalAuthOperations に spec に存在しない operation があります。削除してください")
}

// assertOptionalAuthDeclaresBearer は、任意認証として登録した operation が BearerAuth を宣言していることを表明する。
//
// 空の要件だけでは資格情報を受け取る経路が無く、無効な資格情報を拒否しようがない。拒否は提示された
// 資格情報の検証失敗として起きるため、BearerAuth の宣言が前提になる
// （ADR-0019 (optional-authentication-fail-closed)）。
func assertOptionalAuthDeclaresBearer(t *testing.T) {
	t.Helper()

	spec, err := gen.GetSpec()
	require.NoError(t, err)
	require.NotNil(t, spec.Paths)

	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			key := fmt.Sprintf("%s %s", method, path)
			if _, ok := optionalAuthOperations[key]; !ok {
				continue
			}
			assert.True(t, declaresBearer(effectiveSecurity(spec, op)),
				"%s は任意認証として登録されていますが BearerAuth を宣言していません", key)
		}
	}
}

func TestSecurityDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証を要求しないoperationは姿勢に対応する許可リストにだけ載っている", func(t *testing.T) {
			t.Parallel()

			assertAllowListsCoverSpec(t)
		})

		t.Run("任意認証として登録したoperationはBearerAuthを宣言している", func(t *testing.T) {
			t.Parallel()

			assertOptionalAuthDeclaresBearer(t)
		})

		t.Run("2つの許可リストは同じoperationを重複して持たない", func(t *testing.T) {
			t.Parallel()

			for key := range optionalAuthOperations {
				assert.NotContains(t, publicOperations, key,
					"%s が両方の許可リストに登録されています。姿勢はどちらか一方です", key)
			}
		})
	})
}

func TestSecurityRequirementOrder(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("任意認証のoperationは空のsecurity要件を末尾に置いている", func(t *testing.T) {
			t.Parallel()

			// 要件は先頭から順に試され、空の要件は必ず満たされてそこで評価が止まる。空を先に置くと
			// 認証関数が一度も呼ばれず、提示された資格情報の検証失敗が拒否へ結びつかない
			// （ADR-0019 (optional-authentication-fail-closed)）。
			spec, err := gen.GetSpec()
			require.NoError(t, err)
			require.NotNil(t, spec.Paths)

			for path, item := range spec.Paths.Map() {
				for method, op := range item.Operations() {
					reqs := effectiveSecurity(spec, op)
					idx := emptyRequirementIndex(reqs)
					if idx < 0 || len(reqs) == 1 {
						continue
					}
					assert.Equal(t, len(reqs)-1, idx,
						"%s %s: 空の security 要件は末尾に置いてください。先に置くと認証関数が呼ばれず、"+
							"無効な資格情報が匿名として通ります", method, path)
				}
			}
		})
	})
}
