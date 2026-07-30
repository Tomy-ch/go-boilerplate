package errorhandler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
)

// allowHeaderSeparator は、Allow ヘッダーのメソッド区切りです（RFC 9110 §10.2.1 の #method）。
const allowHeaderSeparator = ", "

// allowProbeMethods は、リクエストのパステンプレートを逆引きするために試行するメソッドです。
// 405 のリクエスト自身のメソッドは定義上どのルートにも解決しないため、
// 同じパスに定義された別メソッドで解決させてテンプレートを得ます。
//
// OpenAPI が operation を定義できるメソッドを網羅します。1 つでも欠けると、
// 欠けたメソッドだけを定義したパスでテンプレートを引けず Allow が欠落します。
var allowProbeMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodHead,
	http.MethodOptions,
	http.MethodTrace,
	http.MethodConnect,
}

// AllowPolicy は、405 レスポンスへ返す Allow ヘッダーの値を解決します。
type AllowPolicy interface {
	// Allow は、req のパスが OpenAPI で許可するメソッドを Allow ヘッダーの値として返します。
	// パスが spec のどのルートにも解決しない場合は空文字を返します。
	Allow(req *http.Request) string
}

// openAPIAllowPolicy は [AllowPolicy] の実装です。
// 起動時にパステンプレートごとの Allow ヘッダー値を組み立てて保持し、リクエスト単位でテンプレートを逆引きします。
type openAPIAllowPolicy struct {
	router routers.Router
	allow  map[string]string
}

// NewOpenAPIAllowPolicy は、spec からパスごとの許可メソッド一覧を構築します。
// 解決は Host に依存しないため、router は [newHostAgnosticRouter] で構築します。
func NewOpenAPIAllowPolicy(spec *openapi3.T) (AllowPolicy, error) {
	router, err := newHostAgnosticRouter(spec)
	if err != nil {
		return nil, err
	}

	return &openAPIAllowPolicy{
		router: router,
		allow:  buildAllowMap(spec),
	}, nil
}

// Allow は [AllowPolicy] を実装します。
// 405 のリクエストはそのメソッドでは解決しないため、他メソッドで探索してパステンプレートを特定します。
func (p *openAPIAllowPolicy) Allow(req *http.Request) string {
	for _, method := range allowProbeMethods {
		probe := req.Clone(req.Context())
		probe.Method = method

		route, _, err := p.router.FindRoute(probe)
		if err != nil || route == nil {
			continue
		}
		return p.allow[route.Path]
	}
	return ""
}

// buildAllowMap は、パステンプレートごとの Allow ヘッダー値を組み立てます。
// Echo のルータが返す値と揃えるため、spec に定義された全メソッドへ OPTIONS を加えます
// (OPTIONS は Echo が自動で応答するため、spec の定義有無によらず常に許可されます)。
func buildAllowMap(spec *openapi3.T) map[string]string {
	allow := make(map[string]string)
	if spec.Paths == nil {
		return allow
	}

	for path, pathItem := range spec.Paths.Map() {
		methods := make([]string, 0, len(pathItem.Operations())+1)
		for method := range pathItem.Operations() {
			if method != http.MethodOptions {
				methods = append(methods, method)
			}
		}
		sort.Strings(methods)

		allow[path] = strings.Join(append([]string{http.MethodOptions}, methods...), allowHeaderSeparator)
	}
	return allow
}
