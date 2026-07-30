package architest

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// internalPathPrefix は、コード生成専用として spec に置かれ、実装・公開されない operation のパス接頭辞です。
	internalPathPrefix = "/_internal/"
	// specSourceFile は、OpenAPI 定義の手書き正本です。
	specSourceFile = "openapi/openapi.yaml"
)

var (
	// echoImportRe は、Echo v5 を import するファイルを判定します。ルート登録形の走査を
	// このファイル群に限ることで、同名メソッド（logging.Any 等）の誤検出を避けます。
	echoImportRe = regexp.MustCompile(`"github\.com/labstack/echo/v5"`)
	// routeCallRe は、HTTP メソッド名によるルート登録呼び出しを捕捉します。レシーバ名は問いません
	// （手書きは `e`、生成コードは `router` を使う）。
	routeCallRe = regexp.MustCompile(`\.(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|CONNECT|TRACE|QUERY)\(`)
	// literalPathRe は、ルート登録の第 1 引数がパス文字列リテラルであることを要求し、その値を捕捉します。
	// 生成コードは options.BaseURL を前置するため、その形も許容します。
	literalPathRe = regexp.MustCompile(`^\s*(?:options\.BaseURL\s*\+\s*)?"(/[^"]*)"`)
	// opaqueCallRe は、メソッドとパスの組を静的に確定できない Echo の登録形を捕捉します。
	// 引数の形は問わない。パスがリテラルかどうかで見逃しが生まれるより、Echo 依存ファイルで
	// 同名メソッドを呼んだときに loud に落ちるほうが安全な倒れ方だからです。
	opaqueCallRe = regexp.MustCompile(`\.(Any|Match|Static|StaticFS|File|FileFS|RouteNotFound|Group|AddRoute)\(`)
	// opaqueAddCallRe は、Add 系のルート登録を捕捉します。Add は http.Header・url.Values にも
	// あるため、第 1 引数が HTTP メソッドか echo.Route リテラルであることを要求して区別します。
	opaqueAddCallRe = regexp.MustCompile(
		`\.Add\(\s*(?:"(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|CONNECT|TRACE|QUERY)"` +
			`|http\.Method\w+|(?:echo\.)?Route\{)`)
	// specParamRe は、OpenAPI のパスパラメータ表記 {name} を捕捉します。
	specParamRe = regexp.MustCompile(`\{(\w+)\}`)
)

// routeRegistration は、ルートの同一性を決めるメソッドとパステンプレートの組です。
// パスは Echo の表記（:name）に正規化されます。
type routeRegistration struct {
	method string
	path   string
}

// echoRouteScan は、1 ファイルから読み取ったルート登録と、静的に確定できない登録形の記述です。
type echoRouteScan struct {
	routes []routeRegistration
	opaque []string
}

// routeIndex は、走査したファイル群から集めたルート登録の索引です。
// registered はルート → 宣言元ファイル、opaque は静的に確定できない登録形、
// scanned は Echo に依存していたファイル数です。
type routeIndex struct {
	registered map[routeRegistration]string
	opaque     []string
	scanned    int
}

// TestRouteSpecParity は、Echo に登録されるルートと OpenAPI spec の operation が 1:1 で対応することを
// 機械検証する。OpenAPI リクエストバリデーション・details の opt-in ゲート・404/405 の判定はいずれも
// 「登録されたルート = spec の operation」を暗黙の前提にしており、spec に無いルートを足すとテストは
// 緑のまま実行時の挙動だけが変わるため、その前提をここで明示的な契約に変える。
//
// 意図的な spec 外ルートのための許可リストは持たない。許可リスト自体がドリフト源になるため、
// spec 外のルートは常に失敗させる。
//
// ルートの列挙は実 DI グラフではなくソースの静的走査で行う（depguard が go/ast を禁じるため、
// 既存 architest と同じく gofmt 済みソースのテキスト走査を採る）。ソースを見る以上、
// di へ BindHandler を登録し忘れて実行時にルートが生えない配線漏れは本テストでは見えない。
// その配線漏れは TestBindHandlerDIParity が「BindHandler の宣言 = fx.Invoke の列挙」および
// 「RegisterHandlers を持つ生成物あり ⇒ BindHandler の実装あり」として補完する。
//
// spec 側も手書き正本を読むため、サンプル API を削除すると両者は同時に縮み、この契約は保たれる。
func TestRouteSpecParity(t *testing.T) {
	t.Parallel()

	index := scanRepositoryRoutes(t)
	require.NotEmpty(t, index.scanned, "Echo を import するファイルが 1 件も見つからない（走査ルートの誤りを疑う）")
	t.Logf("走査した Echo 依存ファイル: %d 件 / 検出したルート登録: %d 件", index.scanned, len(index.registered))

	assert.Empty(t, index.opaque,
		"メソッドとパスを静的に確定できない登録形がある。spec との 1:1 を検証できないため、"+
			"生成された RegisterHandlers 経由の登録に置き換えること")

	spec := specRouteSet(t)

	assert.Empty(t, index.routesMissingFrom(spec),
		"spec に無いルートが Echo に登録されている。OpenAPI リクエストバリデーションと "+
			"details の opt-in ゲートはいずれも spec 上の operation を前提とするため、"+
			"まず OpenAPI に operation を定義すること")
	assert.Empty(t, index.unregisteredIn(spec),
		"spec の operation に対応するルート登録がソース上に無い。ハンドラパッケージの生成漏れを疑うこと")
}

// Test_scanEchoRouteRegistrations は、TestRouteSpecParity の走査精度に直結する読み取り漏れを防ぐため、
// リポジトリの現状に依存しない合成ソースで各登録形の判定を固定する。
func Test_scanEchoRouteRegistrations(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Echo を import しないファイルは走査対象外になる", func(t *testing.T) {
			t.Parallel()

			scan, found := scanEchoRouteRegistrations("package sample\n\nfunc f(e any) { e.GET(\"/metrics\", nil) }\n")

			assert.False(t, found)
			assert.Empty(t, scan.routes)
			assert.Empty(t, scan.opaque)
		})

		t.Run("手書きのルート登録を読み取る", func(t *testing.T) {
			t.Parallel()

			scan, found := scanEchoRouteRegistrations(echoFuncSource("\te.GET(\"/metrics\", nil)"))

			assert.True(t, found)
			assert.Equal(t, []routeRegistration{{method: "GET", path: "/metrics"}}, scan.routes)
			assert.Empty(t, scan.opaque)
		})

		t.Run("生成コードの BaseURL 前置を剥がして読み取る", func(t *testing.T) {
			t.Parallel()

			scan, found := scanEchoRouteRegistrations(
				echoFuncSource("\trouter.PATCH(options.BaseURL+\"/v1/users/:userId\", wrapper.PatchUsersDetail)"))

			assert.True(t, found)
			assert.Equal(t, []routeRegistration{{method: "PATCH", path: "/v1/users/:userId"}}, scan.routes)
			assert.Empty(t, scan.opaque)
		})

		t.Run("同一行に複数のルート登録があればすべて読み取る", func(t *testing.T) {
			t.Parallel()

			scan, found := scanEchoRouteRegistrations(echoFuncSource("\te.GET(\"/a\", nil); e.POST(\"/b\", nil)"))

			assert.True(t, found)
			assert.Equal(t, []routeRegistration{
				{method: "GET", path: "/a"},
				{method: "POST", path: "/b"},
			}, scan.routes)
			assert.Empty(t, scan.opaque)
		})

		t.Run("ドットを伴わないメソッド様の記述はルート登録として扱わない", func(t *testing.T) {
			t.Parallel()

			// 生成コードの EchoRouter インターフェース宣言がこの形に当たる。
			scan, found := scanEchoRouteRegistrations(
				echoSource("type EchoRouter interface {\n\tGET(path string, h echo.HandlerFunc) echo.RouteInfo\n}"))

			assert.True(t, found)
			assert.Empty(t, scan.routes)
			assert.Empty(t, scan.opaque)
		})

		t.Run("Echo 以外の同名メソッド呼び出しはルート登録として扱わない", func(t *testing.T) {
			t.Parallel()

			scan, found := scanEchoRouteRegistrations(
				echoFuncSource("\treq.Header.Add(k, v)\n\tq.Add(p.Name, p.Value)"))

			assert.True(t, found)
			assert.Empty(t, scan.routes)
			assert.Empty(t, scan.opaque)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パスをリテラルとして読み取れない登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.GET(routePath, nil)", "GET の登録パスをリテラルとして読み取れない")
		})

		t.Run("Any による全メソッド登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.Any(\"/proxy\", nil)", "Any による登録")
		})

		t.Run("Match による複数メソッド登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.Match(methods, \"/proxy\", nil)", "Match による登録")
		})

		t.Run("Group による相対パス登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\tg := e.Group(prefix)", "Group による登録")
		})

		t.Run("Static によるファイル配信登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.Static(\"/assets\", \"public\")", "Static による登録")
		})

		t.Run("AddRoute による構造体渡しの登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.AddRoute(echo.Route{Method: \"GET\", Path: \"/proxy\"})", "AddRoute による登録")
		})

		t.Run("Router 経由の Add による登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.Router().Add(echo.Route{Method: \"GET\", Path: \"/proxy\"})", "Add による登録")
		})

		t.Run("メソッドを実行時に決める Add による登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.Add(http.MethodGet, \"/proxy\", nil)", "Add による登録")
		})

		t.Run("メソッド名リテラルを渡す Add による登録を違反として報告する", func(t *testing.T) {
			t.Parallel()

			assertOpaque(t, "\te.Add(\"GET\", \"/proxy\", nil)", "Add による登録")
		})
	})
}

// Test_echoPath は、spec と Echo でパスパラメータの表記が異なることによる取りこぼしを防ぐため、
// 正規化がパラメータの個数に依らず働くことを固定する。
func Test_echoPath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パラメータを含まないパスはそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "/v1/products", echoPath("/v1/products"))
		})

		t.Run("パスパラメータを Echo の表記へ変換する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "/v1/users/:userId", echoPath("/v1/users/{userId}"))
		})

		t.Run("複数のパスパラメータをすべて変換する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				"/v1/users/:userId/purchases/:purchaseId",
				echoPath("/v1/users/{userId}/purchases/{purchaseId}"))
		})
	})
}

// Test_routeIndex_routesMissingFrom は、spec 外ルートの報告が宣言元を伴い安定した順序で返ることを固定する。
func Test_routeIndex_routesMissingFrom(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("すべてのルートが spec にあれば空を返す", func(t *testing.T) {
			t.Parallel()

			index := routeIndex{registered: map[routeRegistration]string{
				{method: "GET", path: "/metrics"}: "internal/controller/handler/metrics/metrics_handler.go",
			}}

			assert.Empty(t, index.routesMissingFrom(specSetOf(routeRegistration{method: "GET", path: "/metrics"})))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("spec に無いルートを宣言元付きで昇順に返す", func(t *testing.T) {
			t.Parallel()

			index := routeIndex{registered: map[routeRegistration]string{
				{method: "GET", path: "/debug"}:  "internal/controller/handler/debug/debug_handler.go",
				{method: "POST", path: "/admin"}: "internal/controller/handler/admin/admin_handler.go",
				{method: "GET", path: "/health"}: "internal/controller/handler/health/gen/server.gen.go",
			}}

			assert.Equal(t, []string{
				"GET /debug (internal/controller/handler/debug/debug_handler.go)",
				"POST /admin (internal/controller/handler/admin/admin_handler.go)",
			}, index.routesMissingFrom(specSetOf(routeRegistration{method: "GET", path: "/health"})))
		})
	})
}

// Test_routeIndex_unregisteredIn は、登録の無い operation の報告が安定した順序で返ることを固定する。
func Test_routeIndex_unregisteredIn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("すべての operation が登録済みなら空を返す", func(t *testing.T) {
			t.Parallel()

			index := routeIndex{registered: map[routeRegistration]string{
				{method: "GET", path: "/health"}: "internal/controller/handler/health/gen/server.gen.go",
			}}

			assert.Empty(t, index.unregisteredIn(specSetOf(routeRegistration{method: "GET", path: "/health"})))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録の無い operation を昇順に返す", func(t *testing.T) {
			t.Parallel()

			index := routeIndex{registered: map[routeRegistration]string{
				{method: "GET", path: "/health"}: "internal/controller/handler/health/gen/server.gen.go",
			}}
			spec := specSetOf(
				routeRegistration{method: "GET", path: "/health"},
				routeRegistration{method: "GET", path: "/version"},
				routeRegistration{method: "DELETE", path: "/v1/users/:userId"},
			)

			assert.Equal(t, []string{"DELETE /v1/users/:userId", "GET /version"}, index.unregisteredIn(spec))
		})
	})
}

// Test_isScannableGoFile は、走査対象の絞り込みを固定する。テストファイルを除外し損なうと、
// このファイル自身に埋め込んだ合成フィクスチャが実在のルート登録として索引に混入する。
func Test_isScannableGoFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("production の Go ソースは走査対象になる", func(t *testing.T) {
			t.Parallel()

			assert.True(t, isScannableGoFile("internal/controller/handler/metrics/metrics_handler.go"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テストファイルは走査対象外になる", func(t *testing.T) {
			t.Parallel()

			assert.False(t, isScannableGoFile("internal/architest/route_spec_parity_test.go"))
		})

		t.Run("Go ソース以外は走査対象外になる", func(t *testing.T) {
			t.Parallel()

			assert.False(t, isScannableGoFile("internal/controller/handler/README.md"))
		})
	})
}

// Test_repoRelative は、違反報告に絶対パスが漏れないことを固定する。
func Test_repoRelative(t *testing.T) {
	t.Parallel()

	root := filepath.Join(string(filepath.Separator)+"tmp", "repo")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ルート配下のパスをスラッシュ区切りの相対パスにする", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				"internal/architest/route_spec_parity_test.go",
				repoRelative(root, filepath.Join(root, "internal", "architest", "route_spec_parity_test.go")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ルート配下でないパスはそのまま返す", func(t *testing.T) {
			t.Parallel()

			outside := filepath.Join(string(filepath.Separator)+"other", "x.go")

			assert.Equal(t, filepath.ToSlash(outside), repoRelative(root, outside))
		})
	})
}

// String は、"GET /v1/users/:userId" 形式の報告用表記を返します。
func (r routeRegistration) String() string { return r.method + " " + r.path }

// readRouteCalls は、HTTP メソッド名によるルート登録を 1 行から読み取ります。
// パスが文字列リテラルでない登録は spec と突き合わせられないため、opaque として記録します。
func (s *echoRouteScan) readRouteCalls(line string) {
	for _, loc := range routeCallRe.FindAllStringSubmatchIndex(line, -1) {
		method := line[loc[2]:loc[3]]

		m := literalPathRe.FindStringSubmatch(line[loc[1]:])
		if m == nil {
			s.opaque = append(s.opaque, method+" の登録パスをリテラルとして読み取れない: "+strings.TrimSpace(line))
			continue
		}
		s.routes = append(s.routes, routeRegistration{method: method, path: m[1]})
	}
}

// readOpaqueCalls は、メソッドとパスの組を静的に確定できない登録形を 1 行から読み取ります。
func (s *echoRouteScan) readOpaqueCalls(line string) {
	if m := opaqueCallRe.FindStringSubmatch(line); m != nil {
		s.opaque = append(s.opaque, m[1]+" による登録: "+strings.TrimSpace(line))
	}
	if opaqueAddCallRe.MatchString(line) {
		s.opaque = append(s.opaque, "Add による登録: "+strings.TrimSpace(line))
	}
}

// addFile は、1 ファイル分の走査結果を索引へ取り込みます。Echo に依存しないファイルは無視します。
func (idx *routeIndex) addFile(file, src string) {
	scan, found := scanEchoRouteRegistrations(src)
	if !found {
		return
	}
	idx.scanned++

	for _, r := range scan.routes {
		idx.registered[r] = file
	}
	for _, o := range scan.opaque {
		idx.opaque = append(idx.opaque, file+": "+o)
	}
}

// routesMissingFrom は、登録されているが spec に無いルートを "METHOD /path (宣言元)" 形式で返します。
func (idx *routeIndex) routesMissingFrom(spec map[routeRegistration]struct{}) []string {
	out := make([]string, 0, len(idx.registered))
	for route, file := range idx.registered {
		if _, ok := spec[route]; !ok {
			out = append(out, route.String()+" ("+file+")")
		}
	}
	sort.Strings(out)
	return out
}

// unregisteredIn は、spec にあるがどこにも登録されていない operation を "METHOD /path" 形式で返します。
func (idx *routeIndex) unregisteredIn(spec map[routeRegistration]struct{}) []string {
	out := make([]string, 0, len(spec))
	for route := range spec {
		if _, ok := idx.registered[route]; !ok {
			out = append(out, route.String())
		}
	}
	sort.Strings(out)
	return out
}

// specRouteSet は、spec が公開する operation を Echo 表記のルート集合として返します。
//
// 読み出すのは生成物（openapi.gen.yaml / 埋め込み spec）ではなく手書き正本の openapi.yaml です。
// 生成物と正本の同期は生成物ドリフト検査が別途担保しており、正本を読めば「定義を削ったが再生成前」の
// 状態（サンプル API 削除の直後）でも実装側と歩調が合うためです。
//
// 除外するのは /_internal/ 配下の operation だけで、これは spec 自身が「コード生成のためだけに存在し、
// 実際には実装・公開されない」と宣言しているアンカーです。除外は spec 上の宣言から導出するため、
// 手で維持する許可リストにはなりません（登録されていれば spec 外のルートとして失敗します）。
func specRouteSet(t *testing.T) map[routeRegistration]struct{} {
	t.Helper()

	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	spec, err := loader.LoadFromFile(filepath.Join(moduleRoot(t), filepath.FromSlash(specSourceFile)))
	require.NoError(t, err)
	require.NotNil(t, spec.Paths)

	out := make(map[routeRegistration]struct{}, spec.Paths.Len())
	for path, item := range spec.Paths.Map() {
		if strings.HasPrefix(path, internalPathPrefix) {
			continue
		}
		for method := range item.Operations() {
			out[routeRegistration{method: method, path: echoPath(path)}] = struct{}{}
		}
	}
	require.NotEmpty(t, out, "spec から operation を 1 件も読み出せない")
	return out
}

// echoPath は、OpenAPI のパスパラメータ表記 {name} を Echo の :name へ正規化します。
func echoPath(specPath string) string { return specParamRe.ReplaceAllString(specPath, ":$1") }

// scanRepositoryRoutes は、モジュール内の非テスト Go ソースを走査してルート登録の索引を返します。
func scanRepositoryRoutes(t *testing.T) routeIndex {
	t.Helper()

	root := moduleRoot(t)
	index := routeIndex{registered: map[routeRegistration]string{}}

	for _, dir := range moduleSubdirs(t, "internal", "pkg", "cmd") {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() || !isScannableGoFile(path) {
				return walkErr
			}
			src, readErr := pkgfs.OS{}.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			index.addFile(repoRelative(root, path), string(src))
			return nil
		})
		require.NoError(t, err)
	}
	sort.Strings(index.opaque)

	return index
}

// isScannableGoFile は、ルート登録の走査対象となる Go ソース（非テスト）かを判定します。
func isScannableGoFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

// repoRelative は、報告用にモジュールルートからの相対パスをスラッシュ区切りで返します。
func repoRelative(root, path string) string {
	return filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
}

// scanEchoRouteRegistrations は、Go ソース 1 ファイル分のテキストからルート登録を読み取ります。
// 第 2 戻り値は、そのファイルが Echo v5 を import しているか（= 走査対象だったか）です。
func scanEchoRouteRegistrations(src string) (echoRouteScan, bool) {
	if !echoImportRe.MatchString(src) {
		return echoRouteScan{}, false
	}

	var scan echoRouteScan
	for line := range strings.SplitSeq(src, "\n") {
		scan.readRouteCalls(line)
		scan.readOpaqueCalls(line)
	}
	return scan, true
}

// echoSource は、Echo v5 を import するファイルの体裁に宣言 body を埋め込んだ走査用ソースを返します。
func echoSource(body string) string {
	return strings.Join([]string{
		"package sample",
		"",
		"import (",
		"\t\"github.com/labstack/echo/v5\"",
		")",
		"",
		body,
	}, "\n")
}

// echoFuncSource は、Echo を受け取る関数の本体に stmts を置いた走査用ソースを返します。
func echoFuncSource(stmts string) string {
	return echoSource("func f(e *echo.Echo) {\n" + stmts + "\n}")
}

// assertOpaque は、stmts が spec と突き合わせられない登録形として 1 件だけ報告されることを検証します。
func assertOpaque(t *testing.T, stmts, wantReason string) {
	t.Helper()

	scan, found := scanEchoRouteRegistrations(echoFuncSource(stmts))

	assert.True(t, found)
	assert.Empty(t, scan.routes)
	require.Len(t, scan.opaque, 1)
	assert.Contains(t, scan.opaque[0], wantReason)
}

// specSetOf は、指定したルートだけを含む spec 側のルート集合を組み立てます。
func specSetOf(routes ...routeRegistration) map[routeRegistration]struct{} {
	out := make(map[routeRegistration]struct{}, len(routes))
	for _, r := range routes {
		out[r] = struct{}{}
	}
	return out
}
