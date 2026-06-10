// Package pgerror は、PostgreSQL固有の処理を提供します。
package pgerror

import (
	"context"
	"errors"
	"net"
	"strings"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// sqlstateToAppError は、PostgreSQL の SQLSTATE をアプリケーションエラーへ対応付けます。
// 対応表は README の SQLSTATE 表を正とします。
var sqlstateToAppError = map[string]error{
	"23505": apperror.ErrConflict,
	"23503": apperror.ErrInvalidArgument,
	"23502": apperror.ErrInvalidArgument,
	"23514": apperror.ErrInvalidArgument,
	"22001": apperror.ErrInvalidArgument,
	"22P02": apperror.ErrInvalidArgument,
	"42501": apperror.ErrPermissionDenied,
	"40001": apperror.ErrUnavailable,
	"40P01": apperror.ErrUnavailable,
	"57014": apperror.ErrUnavailable,
}

// NormalizeError は、PostgreSQLのエラーをアプリケーション固有のエラーに変換します。
// Infrastructure層から返されるPostgreSQLエラーを一貫した形で処理するために使用します。
// 既に正規化済みの apperror はそのまま返します。
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}

	if apperror.IsAppError(err) {
		return err
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return xerrors.Wrap(apperror.ErrNotFound, err.Error())
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if appErr, ok := sqlstateToAppError[pgErr.Code]; ok {
			return xerrors.Wrap(appErr, err.Error())
		}
	}

	if errors.Is(err, context.Canceled) {
		return xerrors.Wrap(apperror.ErrCanceled, err.Error())
	}

	if IsUnavailable(err) {
		return xerrors.Wrap(apperror.ErrUnavailable, err.Error())
	}

	return xerrors.Wrap(apperror.ErrInternal, err.Error())
}

// NormalizeExecResult は、影響行数を返す書き込み系（sqlc `:execrows`）の結果を正規化します。
// エラーは NormalizeError と同じ規則で変換し、エラーが無くても影響行数が 0 の場合は
// 対象が存在しないとみなして ErrNotFound を返します（UPDATE / DELETE のサイレント成功を防ぐ）。
func NormalizeExecResult(affected int64, err error) error {
	if err != nil {
		return NormalizeError(err)
	}
	if affected == 0 {
		return xerrors.Wrap(apperror.ErrNotFound, "no rows affected")
	}
	return nil
}

// IsUnavailable は、与えられたエラーがデータベースの接続不可エラーであるかを判定します。
func IsUnavailable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}

	return isPgConnectionError(err)
}

// isPgConnectionError は、与えられたエラーがPostgreSQLの接続エラーであるかを判定します。
func isPgConnectionError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return strings.HasPrefix(pgErr.Code, "08")
}
