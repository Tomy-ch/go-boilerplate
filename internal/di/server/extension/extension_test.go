package extension

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPreMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("優先度順にPreミドルウェアが適用される", func(t *testing.T) {
			t.Parallel()

			e := echo.New()

			mwA := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Response().Header().Add("X-Order", "A")
					return next(c)
				}
			}
			mwB := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Response().Header().Add("X-Order", "B")
					return next(c)
				}
			}

			// Priority はあえてバラバラにして、ソートされることを確認
			mws := []PreMiddleware{
				{Name: "B", Priority: 2, Middleware: mwB},
				{Name: "A", Priority: 1, Middleware: mwA},
			}

			err := ApplyPreMiddlewares(e, logging.NewTestLogger(t), mws)
			require.NoError(t, err)

			// 適当なハンドラを登録して 1 回リクエスト
			e.GET("/", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			// Pre ミドルウェアで積まれたヘッダの順序を確認
			vals := rec.Header()["X-Order"]
			assert.Equal(t, []string{"A", "B"}, vals)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("優先度が重複する場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()

			mws := []PreMiddleware{
				{
					Name:       "pre1",
					Priority:   1,
					Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next },
				},
				{
					Name:       "pre2",
					Priority:   1, // ★ 敢えて同じ Priority
					Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next },
				},
			}

			err := ApplyPreMiddlewares(e, logging.NewTestLogger(t), mws)
			require.Error(t, err)
			// エラーメッセージの一部まで見ておきたい場合
			assert.Contains(t, err.Error(), "priority")
		})
	})
}

func TestServerExtends(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("PriorityOrderが昇順で適用されること", func(t *testing.T) {
			t.Parallel()

			e := echo.New()

			mwA := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Response().Header().Add("X-Order", "A")
					return next(c)
				}
			}
			mwB := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Response().Header().Add("X-Order", "B")
					return next(c)
				}
			}

			// provide out-of-order priorities to ensure sorting happens
			mws := []UseMiddleware{
				{Name: "B", Priority: 2, Middleware: mwB},
				{Name: "A", Priority: 1, Middleware: mwA},
			}

			err := ApplyUseMiddlewares(e, logging.NewTestLogger(t), mws)
			require.NoError(t, err)

			e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			vals := rec.Header()["X-Order"]
			// expect sorted by Priority ascending: A then B
			assert.Equal(t, []string{"A", "B"}, vals)
		})

		t.Run("拡張機能が適用されること", func(t *testing.T) {
			t.Parallel()

			e := echo.New()

			cfg := func(e *echo.Echo) {
				e.GET("/cfg", func(c echo.Context) error {
					c.Response().Header().Set("X-Cfg", "yes")
					return c.NoContent(http.StatusNoContent)
				})
			}

			ApplyConfigurators(e, logging.NewTestLogger(t), []SrvCfg{cfg})

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/cfg", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNoContent, rec.Code)
			assert.Equal(t, "yes", rec.Header().Get("X-Cfg"))
		})

		t.Run("拡張機能が統合的に適用されること", func(t *testing.T) {
			t.Parallel()

			e := echo.New()

			pre := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Response().Header().Set("X-Pre", "ok")
					return next(c)
				}
			}
			mw := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Response().Header().Add("X-Order", "1")
					return next(c)
				}
			}
			cfg := func(e *echo.Echo) {
				e.GET("/ext", func(c echo.Context) error {
					return c.String(http.StatusOK, "done")
				})
			}

			extends := ServerExtends{
				PreList: []PreMiddleware{{Priority: 0, Middleware: pre}},
				UseList: []UseMiddleware{{Priority: 0, Middleware: mw}},
				CfgList: []SrvCfg{cfg},
			}

			_, err := ApplyExtends(e, logging.NewTestLogger(t), extends)
			require.NoError(t, err)

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ext", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "ok", rec.Header().Get("X-Pre"))
			assert.Equal(t, "1", rec.Header().Get("X-Order"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("UseMiddlewareに、Priorityの重複があるとエラーになること", func(t *testing.T) {
			t.Parallel()

			e := echo.New()

			dup := []UseMiddleware{
				{Name: "A", Priority: 1, Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next }},
				{Name: "B", Priority: 1, Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next }},
			}

			err := ApplyUseMiddlewares(e, logging.NewTestLogger(t), dup)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "priority")
		})

		t.Run("PreMiddlewareに、Priorityの重複があるとエラーになること", func(t *testing.T) {
			t.Parallel()

			e := echo.New()

			extends := ServerExtends{
				PreList: []PreMiddleware{
					{Name: "pre1", Priority: 1, Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next }},
					{Name: "pre2", Priority: 1, Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next }},
				},
			}

			applied, err := ApplyExtends(e, logging.NewTestLogger(t), extends)

			require.Error(t, err)
			require.Nil(t, applied)
		})
	})
}

func TestApplyExtends(t *testing.T) {
	t.Parallel()

	t.Run("拡張機能が適用されること", func(t *testing.T) {
		t.Parallel()

		e := echo.New()

		pre := func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("X-Pre", "ok")
				return next(c)
			}
		}

		use := func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Add("X-Use", "1")
				return next(c)
			}
		}

		cfg := func(e *echo.Echo) {
			e.GET("/ext", func(c echo.Context) error {
				return c.String(http.StatusOK, "done")
			})
		}

		extends := ServerExtends{
			PreList: []PreMiddleware{
				{
					Name:       "pre1",
					Priority:   0,
					Middleware: pre,
				},
			},
			UseList: []UseMiddleware{
				{
					Name:       "use1",
					Priority:   0,
					Middleware: use,
				},
			},
			CfgList: []SrvCfg{cfg},
		}

		applied, err := ApplyExtends(e, logging.NewTestLogger(t), extends)
		require.NoError(t, err)
		require.NotNil(t, applied)

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ext", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Header().Get("X-Pre"))
		assert.Equal(t, "1", rec.Header().Get("X-Use"))
	})

	t.Run("Priorityの重複があるとエラーになること", func(t *testing.T) {
		t.Parallel()

		e := echo.New()

		// Priority 重複でエラーを誘発
		extends := ServerExtends{
			UseList: []UseMiddleware{
				{
					Name:       "mw1",
					Priority:   1,
					Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next },
				},
				{
					Name:       "mw2",
					Priority:   1, // 衝突
					Middleware: func(next echo.HandlerFunc) echo.HandlerFunc { return next },
				},
			},
		}

		applied, err := ApplyExtends(e, logging.NewTestLogger(t), extends)

		require.Error(t, err)
		require.Nil(t, applied)
	})
}

func TestApplyFunctions_HandleEmptySlices_NoPanic(t *testing.T) {
	t.Parallel()

	e := echo.New()
	// ensure calling with nil/empty slices does not panic
	err := ApplyPreMiddlewares(e, logging.NewTestLogger(t), nil)
	require.NoError(t, err)
	err = ApplyUseMiddlewares(e, logging.NewTestLogger(t), nil)
	require.NoError(t, err)
	ApplyConfigurators(e, logging.NewTestLogger(t), nil)

	// still able to register and serve a route
	e.GET("/ok", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func Test_validatePriorityConflicts(t *testing.T) {
	t.Parallel()

	t.Run("priority がユニークならエラーなし", func(t *testing.T) {
		t.Parallel()

		mws := []middlewareEntry{
			{name: "A", priority: 10},
			{name: "B", priority: 20},
			{name: "C", priority: 30},
		}

		err := validatePriorityConflicts("use", mws)
		require.NoError(t, err)
	})

	t.Run("同じ priority が複数あればエラー", func(t *testing.T) {
		t.Parallel()

		mws := []middlewareEntry{
			{name: "A", priority: 10},
			{name: "B", priority: 20},
			{name: "C", priority: 20},
		}

		err := validatePriorityConflicts("use", mws)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "priority=20")
		assert.Contains(t, err.Error(), "B")
		assert.Contains(t, err.Error(), "C")
	})

	t.Run("kind 種別がエラー文言へ反映されること", func(t *testing.T) {
		t.Parallel()

		mws := []middlewareEntry{
			{name: "A", priority: 1},
			{name: "B", priority: 1},
		}

		preErr := validatePriorityConflicts("pre", mws)
		require.Error(t, preErr)
		assert.Contains(t, preErr.Error(), "duplicate pre middleware priorities")

		useErr := validatePriorityConflicts("use", mws)
		require.Error(t, useErr)
		assert.Contains(t, useErr.Error(), "duplicate use middleware priorities")
	})
}

func TestExtractPriorityConflicts(t *testing.T) {
	t.Parallel()

	t.Run("重複なしの場合は空スライスを返す", func(t *testing.T) {
		t.Parallel()

		input := map[int][]string{
			1: {"A"},
			2: {"B"},
			3: {"C"},
		}

		got := extractPriorityConflicts(input)
		require.Empty(t, got)
	})

	t.Run("重複がある場合はそのpriorityのみ返す", func(t *testing.T) {
		t.Parallel()

		input := map[int][]string{
			1: {"A", "B"}, // ★ 重複
			2: {"C"},
		}

		got := extractPriorityConflicts(input)

		require.Len(t, got, 1)
		assert.Contains(t, got[0], "priority=1")
		assert.Contains(t, got[0], "[A B]") // names の出力
	})

	t.Run("複数のpriorityが重複している場合は複数返す", func(t *testing.T) {
		t.Parallel()

		input := map[int][]string{
			1: {"A", "B"}, // ★ 重複
			2: {"C"},
			3: {"X", "Y", "Z"}, // ★ 重複
		}

		got := extractPriorityConflicts(input)

		require.Len(t, got, 2)
		assert.Contains(t, got[0], "priority=")
		assert.Contains(t, got[1], "priority=")

		// priority=1 が含まれていること
		assert.True(t, containsSubstring(got, "priority=1"))
		// priority=3 が含まれていること
		assert.True(t, containsSubstring(got, "priority=3"))
	})

	t.Run("names が3つ以上の場合でも正しくフォーマットされる", func(t *testing.T) {
		t.Parallel()

		input := map[int][]string{
			5: {"A", "B", "C"},
		}

		got := extractPriorityConflicts(input)
		require.Len(t, got, 1)
		assert.Contains(t, got[0], "priority=5")
		assert.Contains(t, got[0], "[A B C]")
	})
}

// ヘルパー: スライス内に部分文字列を含む要素があるか判定
func containsSubstring(list []string, substr string) bool {
	for _, v := range list {
		if strings.Contains(v, substr) {
			return true
		}
	}
	return false
}
