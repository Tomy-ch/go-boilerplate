// Package useridentity は、外部アイデンティティ（issuer + subject）を内部ユーザーへ解決する
// IdentityResolver の RDB 実装を提供します。
package useridentity

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/xerrors"
)

type resolver struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、authbd.IdentityResolver の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) authbd.IdentityResolver {
	return &resolver{
		db:     db,
		tracer: tf.Infra(),
	}
}

// Resolve は、authn の Issuer と Subject に対応する内部ユーザーを user_identities から解決し、UserID を設定した Authn を返します。
// 対応するアイデンティティが無い場合は ErrIdentityNotFound、解決したユーザーが削除済みの場合は ErrUserUnavailable、
// 解決した UserID がゼロ値の場合は ErrUserIDZero を返します。
func (r *resolver) Resolve(ctx context.Context, authn *authbd.Authn) (*authbd.Authn, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.ResolveUserByIdentity(ctx, &gen.ResolveUserByIdentityParams{
		IssuerParam:  authn.Issuer(),
		SubjectParam: authn.Subject(),
	})
	if err != nil {
		// 該当なしは 404 ではなく未認証として扱う（issuer/subject の不一致を漏らさない）。
		nerr := pgerror.NormalizeError(err)
		if xerrors.Is(nerr, apperror.ErrNotFound) {
			return nil, authbd.ErrIdentityNotFound
		}
		return nil, nerr
	}

	if row.DeletedAt != nil {
		return nil, authbd.ErrUserUnavailable
	}

	return authn.WithUserID(row.ID)
}
