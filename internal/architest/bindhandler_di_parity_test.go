package architest

import (
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	pkgfs "go-boilerplate/pkg/fs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// controllerModuleFile は、ControllerModule が fx.Invoke で BindHandler を列挙する宣言元です。
	controllerModuleFile = "internal/di/module/controller.go"
	// handlerTreeDir は、ハンドラパッケージを収めるディレクトリです。
	handlerTreeDir = "internal/controller/handler"
	// handlerImportPrefix は、ハンドラパッケージの import path 接頭辞です。
	handlerImportPrefix = "go-boilerplate/" + handlerTreeDir
	// generatedServerSuffix は、oapi-codegen が RegisterHandlers を出力するファイルの、
	// パッケージディレクトリから見た位置です。
	generatedServerSuffix = "/gen/server.gen.go"
)

var (
	// handlerImportRe は、controller.go の import 行からエイリアス（省略可）とハンドラの import path を
	// 捕捉します。サンプル API の import 行は末尾に // sample-api:line を持つため、末尾はアンカーしません。
	// 接頭辞の直後に / を要求するのは、handler と同じ綴りで始まる兄弟パッケージを取り違えないためです。
	handlerImportRe = regexp.MustCompile(`^\t(?:(\w+)\s+)?"(` + regexp.QuoteMeta(handlerImportPrefix) + `/[^"]*)"`)
	// bindHandlerInvokeRe は、fx.Invoke の引数として縦に並んだ <ident>.BindHandler 行を捕捉します。
	// gofmt は縦並びの引数に末尾カンマを強制するためカンマまで要求し、コメントアウトされた行は
	// 行頭がタブ + \w に一致しないため拾いません。
	bindHandlerInvokeRe = regexp.MustCompile(`^\t+(\w+)\.BindHandler,`)
	// packageClauseRe は、gofmt 済みソースの package 宣言からパッケージ名を捕捉します。
	packageClauseRe = regexp.MustCompile(`(?m)^package (\w+)$`)
)

// handlerTreeIndex は、ハンドラツリーの走査結果です。キーはいずれもモジュールルートからの相対
// パッケージディレクトリで、declared は BindHandler の宣言元ファイル、generated は RegisterHandlers を
// 持つ生成ファイル、packages は宣言元が名乗るパッケージ名を値に取ります。
type handlerTreeIndex struct {
	declared  map[string]string
	generated map[string]string
	packages  map[string]string
}

// TestBindHandlerDIParity は、ハンドラの DI 配線が漏れていないことを機械検証する。
//
// 突き合わせるのは 3 つの集合で、いずれもモジュールルートからの相対パッケージディレクトリで表す。
//
//	X: func BindHandler を宣言するパッケージ（ハンドラツリーの走査）
//	Y: ControllerModule() の fx.Invoke が列挙する BindHandler の所属パッケージ（controller.go の走査）
//	Z: gen/server.gen.go に RegisterHandlers を持つパッケージ（同じくハンドラツリーの走査）
//
// 主張する契約は X == Y と Z ⊆ X で、それぞれ次の書き忘れを検出する。
//
//   - X ⊆ Y: ハンドラを実装したが fx.Invoke へ足し忘れた状態。実行時に Echo へルートが生えず 404 に
//     なるが、TestRouteSpecParity は spec から機械生成される gen/server.gen.go のルート登録を見るため
//     緑のまま通る。本テストの主目的。
//   - Y ⊆ X: fx.Invoke が呼ぶ BindHandler の宣言を走査で見つけられない状態。通常はコンパイラが
//     保証する向きで、ここで主張する意味は、走査条件が実態からずれて X が黙って縮んだときに
//     loud に落とすこと。
//   - Z ⊆ X: make gen-api までは済ませたがハンドラ実装（BindHandler）を書いていない状態。生成物は
//     単独でコンパイルでき、TestRouteSpecParity も生成物のルート登録を見て緑になるため、
//     この向きが無いと実行時 404 に気付けない。
//
// Z == X としないのは、生成コードを持たないハンドラが規約上ありうるため。/metrics は応答を
// サードパーティのハンドラが作るため oapi-codegen を通さず、BindHandler の中で echo.HandlerFunc を
// 直接登録する（internal/controller/handler/README.md の carve-out）。Z ⊆ X なら、この形は許可リスト
// 無しでそのまま通る。
//
// 検出しないもの: BindHandler が配線されていて中身が誤ったルートを生やす状態は各ハンドラの
// TestBindHandler が、fx.Invoke に書かれた依存が解決できない状態は internal/di/module の
// TestControllerModule_GraphIsValid が担う。
//
// 許可リストは持たない。サンプル API を削除するとハンドラディレクトリと controller.go の
// sample-api マーカー行が同時に消えるため、3 集合は歩調を合わせて縮む。
func TestBindHandlerDIParity(t *testing.T) {
	t.Parallel()

	// 走査が成立したかの確認は require、契約 3 方向の違反は assert で報告する。走査が壊れた状態の
	// 差集合は配線漏れの一覧ではなく読み取り漏れの一覧で、そのまま出すと誤った修正を促すため。
	index := scanHandlerTree(t)
	require.NotEmpty(t, index.declared, "BindHandler の宣言を 1 件も検出できない（走査ルートの誤りを疑う）")
	require.NotEmpty(t, index.generated, "RegisterHandlers を持つ生成物を 1 件も検出できない（走査ルートの誤りを疑う）")

	src := readRepoFile(t, moduleRoot(t), controllerModuleFile)
	invoked, unresolved := collectInvokedBindHandlers(src, collectHandlerImports(src, index.packages))
	require.NotEmpty(t, invoked, "fx.Invoke から BindHandler を 1 件も読み取れない（列挙の書式変更を疑う）")
	require.Empty(t, unresolved,
		"fx.Invoke が呼ぶ BindHandler のパッケージを import から解決できない。"+
			"ハンドラは internal/controller/handler 配下から import すること")

	t.Logf("BindHandler の宣言: %d 件 / fx.Invoke の列挙: %d 件 / RegisterHandlers を持つ生成物: %d 件",
		len(index.declared), len(invoked), len(index.generated))

	assert.Empty(t, index.missingFromDI(invoked),
		"BindHandler を宣言しているのに ControllerModule() の fx.Invoke へ列挙されていない。"+
			"実行時に Echo へルートが生えず 404 になるため、"+
			"internal/di/module/controller.go の fx.Invoke へ <pkg>.BindHandler, を 1 行として加えること")
	assert.Empty(t, index.unknownInvocations(invoked),
		"fx.Invoke が列挙する BindHandler の宣言をハンドラツリーの走査で見つけられない。"+
			"走査条件（生成物の除外・func 宣言の検出）が実態からずれていることを疑うこと")
	assert.Empty(t, index.generatedWithoutBindHandler(),
		"RegisterHandlers を持つ生成物があるのに BindHandler の実装が無い。"+
			"make gen-api だけを済ませてハンドラ実装を書き忘れていることを疑うこと")
}

// Test_collectHandlerImports は、fx.Invoke の ident をパッケージへ解決する精度を、リポジトリの
// 現状に依存しない合成ソースで固定する。解決を取りこぼすと本テストが黙って縮む。
func Test_collectHandlerImports(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エイリアス付きの import を読み取る", func(t *testing.T) {
			t.Parallel()

			imports := collectHandlerImports(controllerModuleSource(
				"\tprefectureshandler \""+handlerImportPrefix+"/v1/prefectures\"", ""), nil)

			assert.Equal(t,
				map[string]string{"prefectureshandler": handlerTreeDir + "/v1/prefectures"},
				imports)
		})

		t.Run("エイリアス無しの import は宣言されたパッケージ名で解決する", func(t *testing.T) {
			t.Parallel()

			// ハンドラのディレクトリ名は HTTP リソース名に合わせる規約のため、パッケージ名と
			// 食い違う（categories は package productcategories）。ディレクトリ名で代用すると
			// 正しく配線されたハンドラを解決できず落とすため、宣言名の優先をここで固定する。
			imports := collectHandlerImports(
				controllerModuleSource("\t\""+handlerImportPrefix+"/v1/products/categories\"", ""),
				map[string]string{handlerTreeDir + "/v1/products/categories": "productcategories"})

			assert.Equal(t,
				map[string]string{"productcategories": handlerTreeDir + "/v1/products/categories"},
				imports)
		})

		t.Run("パッケージ名が未知ならパス最終セグメントで近似する", func(t *testing.T) {
			t.Parallel()

			imports := collectHandlerImports(controllerModuleSource(
				"\t\""+handlerImportPrefix+"/v1/users/detail\"", ""), nil)

			assert.Equal(t, map[string]string{"detail": handlerTreeDir + "/v1/users/detail"}, imports)
		})

		t.Run("行末のマーカーコメントを伴う import を読み取る", func(t *testing.T) {
			t.Parallel()

			imports := collectHandlerImports(controllerModuleSource(
				"\tdashboardhandler \""+handlerImportPrefix+"/v1/dashboard\" // sample-api:line", ""), nil)

			assert.Equal(t,
				map[string]string{"dashboardhandler": handlerTreeDir + "/v1/dashboard"},
				imports)
		})

		t.Run("ハンドラ以外の import は読み取らない", func(t *testing.T) {
			t.Parallel()

			imports := collectHandlerImports(controllerModuleSource(
				"\t\"go-boilerplate/internal/usecase/user\"", ""), nil)

			assert.Empty(t, imports)
		})

		t.Run("接頭辞が同じ綴りで始まる兄弟パッケージは読み取らない", func(t *testing.T) {
			t.Parallel()

			imports := collectHandlerImports(controllerModuleSource(
				"\t\""+handlerImportPrefix+"util\"", ""), nil)

			assert.Empty(t, imports)
		})
	})
}

// Test_packageNameOf は、エイリアス無し import の解決に使うパッケージ名の読み取りを固定する。
func Test_packageNameOf(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ディレクトリ名と異なるパッケージ名を読み取る", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "productcategories",
				packageNameOf("// Package productcategories は …\npackage productcategories\n"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("package 宣言が無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, packageNameOf("// package sample\n"))
		})
	})
}

// Test_collectInvokedBindHandlers は、fx.Invoke の列挙を読み取る精度を合成ソースで固定する。
// 読み取り漏れは検証すべき差集合を空にし、テストを黙って通す。
func Test_collectInvokedBindHandlers(t *testing.T) {
	t.Parallel()

	imports := map[string]string{
		"health": handlerTreeDir + "/health",
		"detail": handlerTreeDir + "/v1/users/detail",
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("縦並びの列挙をすべて読み取る", func(t *testing.T) {
			t.Parallel()

			invoked, unresolved := collectInvokedBindHandlers(
				controllerModuleSource("", "\t\t\thealth.BindHandler,\n\t\t\tdetail.BindHandler,"), imports)

			assert.Equal(t, map[string]struct{}{
				handlerTreeDir + "/health":          {},
				handlerTreeDir + "/v1/users/detail": {},
			}, invoked)
			assert.Empty(t, unresolved)
		})

		t.Run("コメントアウトされた列挙は読み取らない", func(t *testing.T) {
			t.Parallel()

			invoked, unresolved := collectInvokedBindHandlers(
				controllerModuleSource("", "\t\t\t// health.BindHandler,"), imports)

			assert.Empty(t, invoked)
			assert.Empty(t, unresolved)
		})

		t.Run("BindHandler 以外の invoke は読み取らない", func(t *testing.T) {
			t.Parallel()

			invoked, unresolved := collectInvokedBindHandlers(
				controllerModuleSource("", "\t\t\thealth.BindMiddleware,"), imports)

			assert.Empty(t, invoked)
			assert.Empty(t, unresolved)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("import へ解決できない列挙を昇順に報告する", func(t *testing.T) {
			t.Parallel()

			invoked, unresolved := collectInvokedBindHandlers(
				controllerModuleSource("", "\t\t\tzhandler.BindHandler,\n\t\t\tahandler.BindHandler,"), imports)

			assert.Empty(t, invoked)
			assert.Equal(t, []string{"ahandler.BindHandler,", "zhandler.BindHandler,"}, unresolved)
		})
	})
}

// Test_declaresFunc は、トップレベル関数の宣言判定を固定する。メソッドを関数と取り違えると、
// BindHandler メソッドを持つだけのパッケージが宣言済みとして数えられてしまう。
func Test_declaresFunc(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トップレベルの関数宣言を検出する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, declaresFunc("package sample\n\nfunc BindHandler(e *echo.Echo) {\n}\n", "BindHandler"))
		})

		t.Run("引数を改行して並べた関数宣言を検出する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, declaresFunc("package sample\n\nfunc BindHandler(\n\te *echo.Echo,\n) {\n}\n", "BindHandler"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同名のメソッド宣言は関数として扱わない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declaresFunc("package sample\n\nfunc (s *server) BindHandler() {\n}\n", "BindHandler"))
		})

		t.Run("コメント中の関数宣言は検出しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declaresFunc("package sample\n\n// func BindHandler() {}\n", "BindHandler"))
		})

		t.Run("名前が前方一致するだけの関数宣言は検出しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, declaresFunc("package sample\n\nfunc BindHandlerV2() {\n}\n", "BindHandler"))
		})
	})
}

// Test_handlerTreeIndex_addFile は、走査対象の振り分けを固定する。生成物の親ディレクトリを取り違えると
// Z の突き合わせが常に空振りする。
func Test_handlerTreeIndex_addFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("BindHandler を宣言するファイルを宣言元として取り込む", func(t *testing.T) {
			t.Parallel()

			idx := newHandlerTreeIndex()
			idx.addFile(handlerTreeDir+"/health/health_handler.go", "package health\n\nfunc BindHandler() {\n}\n")

			assert.Equal(t,
				map[string]string{handlerTreeDir + "/health": handlerTreeDir + "/health/health_handler.go"},
				idx.declared)
			assert.Equal(t, map[string]string{handlerTreeDir + "/health": "health"}, idx.packages)
			assert.Empty(t, idx.generated)
		})

		t.Run("生成物は gen の親をパッケージとして取り込む", func(t *testing.T) {
			t.Parallel()

			idx := newHandlerTreeIndex()
			idx.addFile(handlerTreeDir+"/health/gen/server.gen.go", "package gen\n\nfunc RegisterHandlers() {\n}\n")

			assert.Equal(t,
				map[string]string{handlerTreeDir + "/health": handlerTreeDir + "/health/gen/server.gen.go"},
				idx.generated)
			assert.Empty(t, idx.declared)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テストファイルは走査対象外になる", func(t *testing.T) {
			t.Parallel()

			idx := newHandlerTreeIndex()
			idx.addFile(handlerTreeDir+"/health/health_handler_test.go", "package health\n\nfunc BindHandler() {\n}\n")

			assert.Empty(t, idx.declared)
			assert.Empty(t, idx.generated)
		})

		t.Run("RegisterHandlers を持たない生成物は取り込まない", func(t *testing.T) {
			t.Parallel()

			idx := newHandlerTreeIndex()
			idx.addFile(handlerTreeDir+"/health/gen/server.gen.go", "package gen\n\ntype ServerInterface interface{}\n")

			assert.Empty(t, idx.generated)
		})

		t.Run("BindHandler を宣言しない通常ファイルは取り込まない", func(t *testing.T) {
			t.Parallel()

			idx := newHandlerTreeIndex()
			idx.addFile(handlerTreeDir+"/health/response.go", "package health\n\ntype Response struct{}\n")

			assert.Empty(t, idx.declared)
			assert.Empty(t, idx.packages)
		})
	})
}

// Test_handlerTreeIndex_missingFromDI は、配線漏れの報告が宣言元を伴い安定した順序で返ることを固定する。
func Test_handlerTreeIndex_missingFromDI(t *testing.T) {
	t.Parallel()

	idx := handlerTreeIndex{declared: map[string]string{
		handlerTreeDir + "/health":       handlerTreeDir + "/health/health_handler.go",
		handlerTreeDir + "/v1/users":     handlerTreeDir + "/v1/users/v1_users_handler.go",
		handlerTreeDir + "/v1/dashboard": handlerTreeDir + "/v1/dashboard/v1_dashboard_summary_handler.go",
	}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("すべての宣言が列挙されていれば空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, idx.missingFromDI(dirSetOf(
				handlerTreeDir+"/health", handlerTreeDir+"/v1/users", handlerTreeDir+"/v1/dashboard")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("列挙されていない宣言を宣言元付きで昇順に返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, []string{
				handlerTreeDir + "/v1/dashboard (" + handlerTreeDir + "/v1/dashboard/v1_dashboard_summary_handler.go)",
				handlerTreeDir + "/v1/users (" + handlerTreeDir + "/v1/users/v1_users_handler.go)",
			}, idx.missingFromDI(dirSetOf(handlerTreeDir+"/health")))
		})
	})
}

// Test_handlerTreeIndex_unknownInvocations は、走査が宣言を取りこぼしたときの報告が安定した順序で
// 返ることを固定する。
func Test_handlerTreeIndex_unknownInvocations(t *testing.T) {
	t.Parallel()

	idx := handlerTreeIndex{declared: map[string]string{
		handlerTreeDir + "/health": handlerTreeDir + "/health/health_handler.go",
	}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("列挙がすべて宣言済みなら空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, idx.unknownInvocations(dirSetOf(handlerTreeDir+"/health")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("宣言を見つけられない列挙を昇順に返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, []string{
				handlerTreeDir + "/metrics",
				handlerTreeDir + "/version",
			}, idx.unknownInvocations(dirSetOf(
				handlerTreeDir+"/health", handlerTreeDir+"/version", handlerTreeDir+"/metrics")))
		})
	})
}

// Test_handlerTreeIndex_generatedWithoutBindHandler は、実装漏れの報告が生成元を伴い安定した順序で
// 返ること、および生成コードを持たないハンドラが違反にならないことを固定する。
func Test_handlerTreeIndex_generatedWithoutBindHandler(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("すべての生成物に BindHandler があれば空を返す", func(t *testing.T) {
			t.Parallel()

			idx := handlerTreeIndex{
				declared:  map[string]string{handlerTreeDir + "/health": handlerTreeDir + "/health/health_handler.go"},
				generated: map[string]string{handlerTreeDir + "/health": handlerTreeDir + "/health/gen/server.gen.go"},
			}

			assert.Empty(t, idx.generatedWithoutBindHandler())
		})

		t.Run("生成コードを持たないハンドラは違反にならない", func(t *testing.T) {
			t.Parallel()

			// 生成物の側に実在のエントリを置く。generated が空だと走査が 0 回で終わり、
			// declared を回す実装へ取り違えても緑のままになる。
			idx := handlerTreeIndex{
				declared: map[string]string{
					handlerTreeDir + "/health":  handlerTreeDir + "/health/health_handler.go",
					handlerTreeDir + "/metrics": handlerTreeDir + "/metrics/metrics_handler.go",
				},
				generated: map[string]string{
					handlerTreeDir + "/health": handlerTreeDir + "/health/gen/server.gen.go",
				},
			}

			assert.Empty(t, idx.generatedWithoutBindHandler())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("BindHandler の無い生成物を生成元付きで昇順に返す", func(t *testing.T) {
			t.Parallel()

			idx := handlerTreeIndex{
				declared: map[string]string{handlerTreeDir + "/health": handlerTreeDir + "/health/health_handler.go"},
				generated: map[string]string{
					handlerTreeDir + "/health":   handlerTreeDir + "/health/gen/server.gen.go",
					handlerTreeDir + "/v1/users": handlerTreeDir + "/v1/users/gen/server.gen.go",
					handlerTreeDir + "/v1/feed":  handlerTreeDir + "/v1/feed/gen/server.gen.go",
				},
			}

			assert.Equal(t, []string{
				handlerTreeDir + "/v1/feed (" + handlerTreeDir + "/v1/feed/gen/server.gen.go)",
				handlerTreeDir + "/v1/users (" + handlerTreeDir + "/v1/users/gen/server.gen.go)",
			}, idx.generatedWithoutBindHandler())
		})
	})
}

// addFile は、1 ファイル分の走査結果を索引へ取り込みます。走査対象外のファイルは無視します。
func (idx *handlerTreeIndex) addFile(file, src string) {
	if !isScannableGoFile(file) {
		return
	}
	if strings.HasSuffix(file, generatedServerSuffix) {
		if declaresFunc(src, "RegisterHandlers") {
			idx.generated[path.Dir(path.Dir(file))] = file
		}
		return
	}
	if declaresFunc(src, "BindHandler") {
		idx.declared[path.Dir(file)] = file
		idx.packages[path.Dir(file)] = packageNameOf(src)
	}
}

// missingFromDI は、BindHandler を宣言しているが fx.Invoke に列挙されていないパッケージを
// "パッケージ (宣言元)" 形式で返します。
func (idx *handlerTreeIndex) missingFromDI(invoked map[string]struct{}) []string {
	out := make([]string, 0, len(idx.declared))
	for pkg, file := range idx.declared {
		if _, ok := invoked[pkg]; !ok {
			out = append(out, pkg+" ("+file+")")
		}
	}
	sort.Strings(out)
	return out
}

// unknownInvocations は、fx.Invoke に列挙されているが BindHandler の宣言を走査で見つけられなかった
// パッケージを返します。
func (idx *handlerTreeIndex) unknownInvocations(invoked map[string]struct{}) []string {
	out := make([]string, 0, len(invoked))
	for pkg := range invoked {
		if _, ok := idx.declared[pkg]; !ok {
			out = append(out, pkg)
		}
	}
	sort.Strings(out)
	return out
}

// generatedWithoutBindHandler は、RegisterHandlers を持つ生成物があるのに BindHandler の宣言が無い
// パッケージを "パッケージ (生成元)" 形式で返します。
func (idx *handlerTreeIndex) generatedWithoutBindHandler() []string {
	out := make([]string, 0, len(idx.generated))
	for pkg, file := range idx.generated {
		if _, ok := idx.declared[pkg]; !ok {
			out = append(out, pkg+" ("+file+")")
		}
	}
	sort.Strings(out)
	return out
}

// scanHandlerTree は、ハンドラツリー配下の Go ソースを走査して索引を返します。
func scanHandlerTree(t *testing.T) handlerTreeIndex {
	t.Helper()

	root := moduleRoot(t)
	index := newHandlerTreeIndex()

	err := filepath.WalkDir(handlerRoot(t), func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		src, readErr := pkgfs.OS{}.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		index.addFile(repoRelative(root, p), string(src))
		return nil
	})
	require.NoError(t, err)

	return index
}

// newHandlerTreeIndex は、空の索引を返します。
func newHandlerTreeIndex() handlerTreeIndex {
	return handlerTreeIndex{
		declared:  map[string]string{},
		generated: map[string]string{},
		packages:  map[string]string{},
	}
}

// collectHandlerImports は、controller.go の import 節から ident → パッケージディレクトリ
// （モジュールルートからの相対）の対応を返します。
//
// エイリアスの無い import が束縛する ident はパッケージが名乗る名前であって、ディレクトリ名では
// ありません。ハンドラのディレクトリ名は HTTP リソース名に合わせる規約（docs/rules.md）のため
// 両者は実際に食い違い（categories は package productcategories）、ディレクトリ名で代用すると
// 正しく配線されたハンドラを解決できずに落とします。そこで走査で得た packages を先に引き、
// 走査対象外のディレクトリに限りパスの最終セグメントで近似します。
func collectHandlerImports(src string, packages map[string]string) map[string]string {
	out := map[string]string{}
	for line := range strings.SplitSeq(src, "\n") {
		m := handlerImportRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		dir := handlerTreeDir + strings.TrimPrefix(m[2], handlerImportPrefix)
		ident := m[1]
		if ident == "" {
			ident = packages[dir]
		}
		if ident == "" {
			ident = path.Base(dir)
		}
		out[ident] = dir
	}
	return out
}

// packageNameOf は、gofmt 済みソースが宣言するパッケージ名を返します。
func packageNameOf(src string) string {
	m := packageClauseRe.FindStringSubmatch(src)
	if m == nil {
		return ""
	}
	return m[1]
}

// collectInvokedBindHandlers は、fx.Invoke が列挙する BindHandler の所属パッケージ集合と、
// imports へ解決できなかった列挙の記述を返します。
func collectInvokedBindHandlers(src string, imports map[string]string) (map[string]struct{}, []string) {
	invoked := map[string]struct{}{}
	var unresolved []string

	for line := range strings.SplitSeq(src, "\n") {
		m := bindHandlerInvokeRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		dir, ok := imports[m[1]]
		if !ok {
			unresolved = append(unresolved, strings.TrimSpace(line))
			continue
		}
		invoked[dir] = struct{}{}
	}
	sort.Strings(unresolved)

	return invoked, unresolved
}

// declaresFunc は、gofmt 済みソースが name という名のトップレベル関数を宣言しているかを返します。
// レシーバを伴うメソッド宣言は前置きが挟まるため一致しません。
func declaresFunc(src, name string) bool {
	decl := "func " + name + "("
	for line := range strings.SplitSeq(src, "\n") {
		if strings.HasPrefix(line, decl) {
			return true
		}
	}
	return false
}

// controllerModuleSource は、ControllerModule の体裁に import 行群と fx.Invoke の引数行群を埋め込んだ
// 走査用ソースを返します。
func controllerModuleSource(imports, invokes string) string {
	return strings.Join([]string{
		"package module",
		"",
		"import (",
		imports,
		"",
		"\t\"go.uber.org/fx\"",
		")",
		"",
		"func ControllerModule() fx.Option {",
		"\treturn fx.Module(\"controller\",",
		"\t\tfx.Invoke(",
		invokes,
		"\t\t),",
		"\t)",
		"}",
	}, "\n")
}

// dirSetOf は、指定したパッケージディレクトリだけを含む集合を組み立てます。
func dirSetOf(dirs ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(dirs))
	for _, d := range dirs {
		out[d] = struct{}{}
	}
	return out
}
