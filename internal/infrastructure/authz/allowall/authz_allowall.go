// Package allowall は、すべての操作を許可する開発用の Authorizer 実装を提供します。
package allowall

import (
	"context"

	"go-boilerplate/internal/config"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"

	"go-boilerplate/pkg/xerrors"
)

// errNonLocalEnv は、全許可 Authorizer を local / ci / test / dast 以外の環境で生成しようとした場合のエラーです。
var errNonLocalEnv = xerrors.New("allow-all authorizer must not run outside local/ci/test/dast")

// authorizer は、すべての認可要求を許可する Authorizer です（ローカル/開発用の割り切り実装）。
type authorizer struct{}

// New は、全許可 Authorizer のコンストラクタです。
//
// 全許可はすべてのリクエストを無条件に通すため、本番相当の環境で誤って配線されると認可が
// 事実上無効になる危険があります。この安全保証を呼び出し側（DI 配線）の正しさに委ねると、
// 配線ミス一つで破れてしまうため、危険な実装である allowall 自身が前提条件として担保します。
// 非本番環境（local / ci / test / dast）以外では生成を拒否してエラーを返します（fail-closed）。
func New(appCfg *config.ApplicationConfig) (authzbd.Authorizer, error) {
	switch appCfg.Env() {
	case config.EnvLocal, config.EnvCI, config.EnvTest, config.EnvDast:
		return &authorizer{}, nil
	default:
		return nil, xerrors.Wrap(errNonLocalEnv, appCfg.Env())
	}
}

// Authorize は、action / resource によらず常に許可（nil）を返します。
func (a *authorizer) Authorize(_ context.Context, _ *authbd.Authn, _ authzbd.Action, _ *authzbd.Resource) error {
	return nil
}
