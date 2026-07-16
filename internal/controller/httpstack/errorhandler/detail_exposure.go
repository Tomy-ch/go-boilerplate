package errorhandler

import (
	"net/http"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// detailsPropertyName は、error レスポンススキーマが details 公開を opt-in していることを示すプロパティ名です。
// このプロパティを持つ operation だけがクライアントへ details を返せます。
const detailsPropertyName = "details"

// DetailPolicy は、エラーレスポンスに details を含めてよいリクエストかを判定します。
type DetailPolicy interface {
	// Allows は、req が解決する operation が OpenAPI で details 公開を opt-in しているかを返します。
	// ルート不一致・operation 未解決・OperationID 空・未 opt-in はいずれも false(fail-closed)。
	Allows(req *http.Request) bool
}

// openAPIDetailPolicy は [DetailPolicy] の実装です。
// 起動時に spec を解析して opt-in 済み operationId のセットを保持し、リクエスト単位で判定します。
type openAPIDetailPolicy struct {
	router  routers.Router
	allowed map[string]bool
}

// NewOpenAPIDetailPolicy は、spec からエンドポイント opt-in ポリシーを構築します。
// details を返せるのは、error レスポンス(4xx/5xx)の JSON スキーマに details プロパティを持つ operation のみです。
// 判定は Host に依存しないため、router は servers を除去した複製から構築します
// (servers を残すと gorillamux が Host マッチを行い、proxy 等の Host 不一致で全て fail-closed に倒れる)。
func NewOpenAPIDetailPolicy(spec *openapi3.T) (DetailPolicy, error) {
	hostAgnostic := *spec
	hostAgnostic.Servers = nil

	router, err := gorillamux.NewRouter(&hostAgnostic)
	if err != nil {
		return nil, err
	}

	return &openAPIDetailPolicy{
		router:  router,
		allowed: buildDetailExposureMap(spec),
	}, nil
}

// Allows は [DetailPolicy] を実装します。ルート解決に失敗した場合は false(fail-closed)を返します。
func (p *openAPIDetailPolicy) Allows(req *http.Request) bool {
	route, _, err := p.router.FindRoute(req)
	if err != nil || route == nil || route.Operation == nil {
		return false
	}
	return p.allowed[route.Operation.OperationID]
}

// buildDetailExposureMap は、error レスポンスの JSON スキーマに details プロパティを持つ operation を集めます。
func buildDetailExposureMap(spec *openapi3.T) map[string]bool {
	allowed := make(map[string]bool)
	if spec.Paths == nil {
		return allowed
	}
	for _, pathItem := range spec.Paths.Map() {
		for _, op := range pathItem.Operations() {
			if op.OperationID == "" || op.Responses == nil {
				continue
			}
			if operationExposesDetails(op) {
				allowed[op.OperationID] = true
			}
		}
	}
	return allowed
}

// operationExposesDetails は、operation の error レスポンスのいずれかが details プロパティを持つかを返します。
func operationExposesDetails(op *openapi3.Operation) bool {
	for status, respRef := range op.Responses.Map() {
		if !isErrorStatusCode(status) {
			continue
		}
		if responseHasDetailsProperty(respRef) {
			return true
		}
	}
	return false
}

// responseHasDetailsProperty は、レスポンスの application/json スキーマが details プロパティを持つかを返します。
func responseHasDetailsProperty(respRef *openapi3.ResponseRef) bool {
	if respRef == nil || respRef.Value == nil {
		return false
	}
	mediaType := respRef.Value.Content.Get("application/json")
	if mediaType == nil || mediaType.Schema == nil {
		return false
	}
	return schemaHasDetailsProperty(mediaType.Schema.Value)
}

// schemaHasDetailsProperty は、スキーマ(または allOf 合成の各要素)が details プロパティを持つかを返します。
// ErrorResponseWithDetails は現状トップレベルに details を持つが、allOf 合成へ変えても壊れないよう allOf も走査します。
func schemaHasDetailsProperty(schema *openapi3.Schema) bool {
	if schema == nil {
		return false
	}
	if _, ok := schema.Properties[detailsPropertyName]; ok {
		return true
	}
	for _, sub := range schema.AllOf {
		if sub != nil && schemaHasDetailsProperty(sub.Value) {
			return true
		}
	}
	return false
}

// isErrorStatusCode は、レスポンスキー(status 文字列)が 4xx/5xx のエラーステータスかを返します。
// "default" や "4XX" 等のワイルドカードは対象外とします(本 API は明示的な数値ステータスのみ使用)。
func isErrorStatusCode(status string) bool {
	code, err := strconv.Atoi(status)
	if err != nil {
		return false
	}
	return isErrorStatus(code)
}
