// Package extension は、サーバー拡張機能の適用に関するユーティリティを提供します。
package extension

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// callerSkip は、applyMiddlewares → ApplyPreMiddlewares/ApplyUseMiddlewares の呼び出し経路で挟まるフレームを飛ばし、ログの caller を実呼び出し位置に合わせる段数。
const callerSkip = 2

// ServerExtends は、サーバーの拡張機能を表します。
type ServerExtends struct {
	fx.In

	// PreList は、Preミドルウェアとして適用されるミドルウェアのリストです。
	PreList []PreMiddleware `group:"middlewares.pre"`
	// UseList は、Useミドルウェアとして適用されるミドルウェアのリストです。
	UseList []UseMiddleware `group:"middlewares.use"`
	// CfgList は、サーバーに副作用で適用される設定関数のリストです。
	CfgList []SrvCfg `group:"server.configurators"`
}

// AppliedServerExtends は、サーバー拡張が適用されたことを示すトークンです。
type AppliedServerExtends struct{}

// SrvCfg は、サーバーの設定関数とその名前（ログ出力用）を表します。
type SrvCfg struct {
	// Name は、設定関数の名前です（ログ出力用）
	Name string
	// Config は、Echo に副作用で適用される設定関数です。
	Config func(*echo.Echo)
}

// ServeCfgOut は、サーバーの設定の出力時に使用される構造体です。
type ServeCfgOut struct {
	fx.Out

	SrvCfg SrvCfg `group:"server.configurators"`
}

// PreMiddleware は、Pre ミドルウェアとその適用順序を表します。
type PreMiddleware struct {
	// Name は、ミドルウェアの名前です（ログ出力用）
	Name string
	// Priority は、ミドルウェアの適用順序を表します（小さい方が先に適用される）
	Priority int
	// Middleware は、適用対象の Echo ミドルウェアです。
	Middleware echo.MiddlewareFunc
}

// UseMiddleware は、Use ミドルウェアとその適用順序を表します。
type UseMiddleware struct {
	// Name は、ミドルウェアの名前です（ログ出力用）
	Name string
	// Priority は、ミドルウェアの適用順序を表します（小さい方が先に適用される）
	Priority int
	// Middleware は、適用対象の Echo ミドルウェアです。
	Middleware echo.MiddlewareFunc
}

// PreMiddlewareOut は、fx の group 出力用のラッパーです。
type PreMiddlewareOut struct {
	fx.Out

	Middleware PreMiddleware `group:"middlewares.pre"`
}

// UseMiddlewareOut は、fx の group 出力用のラッパーです。
type UseMiddlewareOut struct {
	fx.Out

	Middleware UseMiddleware `group:"middlewares.use"`
}

// middlewareEntry は、Pre/Use 共通の適用処理が扱う内部表現です。
type middlewareEntry struct {
	name       string
	priority   int
	middleware echo.MiddlewareFunc
}

// ApplyExtends は、Pre・Use ミドルウェアおよびサーバー設定関数を Priority 昇順に Echo へ適用します。
// 同一 kind 内で Priority が重複するミドルウェアが存在する場合はエラーを返します。
func ApplyExtends(e *echo.Echo, logger logging.Logger, extends ServerExtends) (*AppliedServerExtends, error) {
	if err := ApplyPreMiddlewares(e, logger, extends.PreList); err != nil {
		return nil, err
	}
	if err := ApplyUseMiddlewares(e, logger, extends.UseList); err != nil {
		return nil, err
	}
	ApplyConfigurators(e, logger, extends.CfgList)
	return &AppliedServerExtends{}, nil
}

// ApplyPreMiddlewares は、mws を Priority 昇順に Echo.Pre として適用します。
// Priority が重複するエントリが存在する場合はエラーを返します。
func ApplyPreMiddlewares(e *echo.Echo, logger logging.Logger, mws []PreMiddleware) error {
	entries := make([]middlewareEntry, len(mws))
	for i, mw := range mws {
		entries[i] = middlewareEntry{name: mw.Name, priority: mw.Priority, middleware: mw.Middleware}
	}
	return applyMiddlewares(logger, "pre", entries, e.Pre)
}

// ApplyUseMiddlewares は、mws を Priority 昇順に Echo.Use として適用します。
// Priority が重複するエントリが存在する場合はエラーを返します。
func ApplyUseMiddlewares(e *echo.Echo, logger logging.Logger, mws []UseMiddleware) error {
	entries := make([]middlewareEntry, len(mws))
	for i, mw := range mws {
		entries[i] = middlewareEntry{name: mw.Name, priority: mw.Priority, middleware: mw.Middleware}
	}
	return applyMiddlewares(logger, "use", entries, e.Use)
}

// applyMiddlewares は、kind 種別のミドルウェアを優先度順に apply で適用します。
func applyMiddlewares(logger logging.Logger, kind string, mws []middlewareEntry, apply func(...echo.MiddlewareFunc)) error {
	if err := validatePriorityConflicts(kind, mws); err != nil {
		return err
	}

	log := logger.Named("ApplyMiddlewares").CallerSkip(callerSkip)
	log.Info(
		context.Background(),
		fmt.Sprintf("Applying %s middleware", kind),
		logging.Int("count", len(mws)),
	)

	sort.Slice(mws, func(i, j int) bool {
		return mws[i].priority < mws[j].priority
	})

	for _, mw := range mws {
		log.Info(
			context.Background(),
			fmt.Sprintf("Applying %s middleware", kind),
			logging.Int("priority", mw.priority),
			logging.String("middleware", mw.name),
		)
		apply(mw.middleware)
	}
	return nil
}

// validatePriorityConflicts は、kind 種別のミドルウェアに Priority の重複がないか検証します。
func validatePriorityConflicts(kind string, mws []middlewareEntry) error {
	byPriority := make(map[int][]string)

	for _, mw := range mws {
		byPriority[mw.priority] = append(byPriority[mw.priority], mw.name)
	}

	conflicts := extractPriorityConflicts(byPriority)

	if len(conflicts) > 0 {
		return xerrors.New(fmt.Sprintf("duplicate %s middleware priorities: %s",
			kind, strings.Join(conflicts, "; "),
		))
	}

	return nil
}

// extractPriorityConflicts は、priority ごとに名前が重複しているものを抽出して返します。
// 戻り値は priority 昇順の conflict 表現の文字列スライス。
func extractPriorityConflicts(byPriority map[int][]string) []string {
	priorities := make([]int, 0, len(byPriority))
	for p := range byPriority {
		priorities = append(priorities, p)
	}
	sort.Ints(priorities)

	var conflicts []string
	for _, p := range priorities {
		names := byPriority[p]
		if len(names) > 1 {
			conflicts = append(conflicts,
				fmt.Sprintf("priority=%d: %v", p, names),
			)
		}
	}
	return conflicts
}

// ApplyConfigurators は、Echoに対して設定関数を適用します。
func ApplyConfigurators(e *echo.Echo, logger logging.Logger, cfgs []SrvCfg) {
	log := logger.Named("ApplyConfigurators").CallerSkip(callerSkip)
	log.Info(
		context.Background(),
		"Applying server configurator",
		logging.Int("count", len(cfgs)),
	)
	for _, cfg := range cfgs {
		log.Info(
			context.Background(),
			"Applying server configurator",
			logging.String("configurator", cfg.Name),
		)
		cfg.Config(e)
	}
}
