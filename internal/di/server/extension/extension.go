// Package extension は、サーバー拡張機能の適用に関するユーティリティを提供します。
package extension

import (
	"fmt"
	"sort"
	"strings"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

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

// SrvCfg は、サーバーの設定関数を表します。
type SrvCfg func(*echo.Echo)

// ServeCfgOut は、サーバーの設定の出力時に使用される構造体です。
type ServeCfgOut struct {
	fx.Out
	SrvCfg SrvCfg `group:"server.configurators"`
}

// PreMiddleware は、Use ミドルウェアとその適用順序を表します。
type PreMiddleware struct {
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

// UseMiddleware は、Use ミドルウェアとその適用順序を表します。
type UseMiddleware struct {
	// Name は、ミドルウェアの名前です（ログ出力用）
	Name string
	// Priority は、ミドルウェアの適用順序を表します（小さい方が先に適用される）
	Priority int
	// Middleware は、適用対象の Echo ミドルウェアです。
	Middleware echo.MiddlewareFunc
}

// UseMiddlewareOut は、fx の group 出力用のラッパーです。
type UseMiddlewareOut struct {
	fx.Out
	Middleware UseMiddleware `group:"middlewares.use"`
}

// ApplyExtends は、サーバー拡張を適用します。
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

// ApplyPreMiddlewares は、Echoに対してPreのミドルウェアを適用します。
func ApplyPreMiddlewares(e *echo.Echo, logger logging.Logger, mws []PreMiddleware) error {
	if err := validatePreMiddlewarePriorityConflicts(mws); err != nil {
		return err
	}

	logger.Named("ApplyPreMiddlewares").CallerSkip(callerSkip).Info(
		"Applying pre middleware",
		logging.Int("count", len(mws)),
	)

	sort.Slice(mws, func(i, j int) bool {
		return mws[i].Priority < mws[j].Priority
	})

	for _, mw := range mws {
		logger.Named("ApplyPreMiddlewares").CallerSkip(callerSkip).Info(
			"Applying pre middleware",
			logging.Int("priority", mw.Priority),
			logging.String("middleware", mw.Name),
		)
		e.Pre(mw.Middleware)
	}
	return nil
}

// ApplyUseMiddlewares は、Echoに対してUseのミドルウェアを適用します。
func ApplyUseMiddlewares(e *echo.Echo, logger logging.Logger, mws []UseMiddleware) error {
	if err := validateUseMiddlewarePriorityConflicts(mws); err != nil {
		return err
	}

	logger.Named("ApplyUseMiddlewares").CallerSkip(callerSkip).Info(
		"Applying use middleware",
		logging.Int("count", len(mws)),
	)

	sort.Slice(mws, func(i, j int) bool {
		return mws[i].Priority < mws[j].Priority
	})

	for _, mw := range mws {
		logger.Named("ApplyUseMiddlewares").CallerSkip(callerSkip).Info(
			"Applying use middleware",
			logging.Int("priority", mw.Priority),
			logging.String("middleware", mw.Name),
		)
		e.Use(mw.Middleware)
	}
	return nil
}

// validateUseMiddlewarePriorityConflicts は、Priority の重複がないか検証します。
func validateUseMiddlewarePriorityConflicts(mws []UseMiddleware) error {
	byPriority := make(map[int][]string)

	for _, mw := range mws {
		byPriority[mw.Priority] = append(byPriority[mw.Priority], mw.Name)
	}

	conflicts := extractPriorityConflicts(byPriority)

	if len(conflicts) > 0 {
		return xerrors.New(fmt.Sprintf("duplicate use middleware priorities: %s",
			strings.Join(conflicts, "; "),
		))
	}

	return nil
}

// validatePreMiddlewarePriorityConflicts は、Priority の重複がないか検証します。
func validatePreMiddlewarePriorityConflicts(mws []PreMiddleware) error {
	byPriority := make(map[int][]string)

	for _, mw := range mws {
		byPriority[mw.Priority] = append(byPriority[mw.Priority], mw.Name)
	}

	conflicts := extractPriorityConflicts(byPriority)

	if len(conflicts) > 0 {
		return xerrors.New(fmt.Sprintf("duplicate use middleware priorities: %s",
			strings.Join(conflicts, "; "),
		))
	}

	return nil
}

// extractPriorityConflicts は、priority ごとに名前が重複しているものを抽出して返します。
// 戻り値は conflict 表現の文字列スライス。
func extractPriorityConflicts(byPriority map[int][]string) []string {
	var conflicts []string

	for p, names := range byPriority {
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
	logger.Named("ApplyConfigurators").CallerSkip(callerSkip).Info(
		"Applying server configurator",
		logging.Int("count", len(cfgs)),
	)
	for _, cfg := range cfgs {
		cfg(e)
	}
}
