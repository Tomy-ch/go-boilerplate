// Package pgerror は、PostgreSQL固有の処理を提供します。
package pgerror

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"strings"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/pkg/xerrors"

	"github.com/jackc/pgx/v5/pgconn"
)

// NormalizeError は、PostgreSQLのエラーをアプリケーション固有のエラーに変換します。
// Infrastructure層から返されるPostgreSQLエラーを一貫した形で処理するために使用します。
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return xerrors.Wrap(apperror.ErrNotFound, err.Error())
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // ユニーク制約違反
			return xerrors.Wrap(apperror.ErrConflict, err.Error())
		case "23503", "23502", "23514", "22001", "22P02":
			// 外部キー制約違反 / NOT NULL制約違反 / チェック制約違反 / 文字数超過 / 型変換エラー
			return xerrors.Wrap(apperror.ErrInvalidArgument, err.Error())
		case "42501": // 権限不足
			return xerrors.Wrap(apperror.ErrPermissionDenied, err.Error())
		case "40001", "40P01": // トランザクションのデッドロック / トランザクションの失敗(リトライ可能)
			return xerrors.Wrap(apperror.ErrUnavailable, err.Error())
		case "57014": // クエリのキャンセル
			return xerrors.Wrap(apperror.ErrUnavailable, err.Error())
		}
	}
	if IsUnavailable(err) { // 接続関連エラー
		return xerrors.Wrap(apperror.ErrUnavailable, err.Error())
	}
	return xerrors.Wrap(apperror.ErrInternal, err.Error())
}

// IsUnavailable は、与えられたエラーがデータベースの接続不可エラーであるかを判定します。
func IsUnavailable(err error) bool {
	if err == nil {
		return false
	}

	// context のタイムアウト
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// ネットワーク／ドライバ系
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}

	// PostgreSQL の connection exception (SQLSTATE 08XXX)
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
