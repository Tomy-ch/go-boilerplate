// Package pgerror は、PostgreSQL エラー（SQLSTATE・接続断・context タイムアウト）をアプリケーションエラー（apperror）へ正規化する関数群と、retryable / lock-timeout / unavailable 判定述語を提供します。
package pgerror

import (
	"context"
	"net"
	"strings"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// sqlstateToAppError は、PostgreSQL の SQLSTATE をアプリケーションエラーへ対応付けます。
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
	"55P03": apperror.ErrUnavailable,
	"57014": apperror.ErrUnavailable,
}

// NormalizeError は、PostgreSQLのエラーをアプリケーション固有のエラーに変換します。
// 既に正規化済みの apperror はそのまま返します。
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}

	if apperror.IsAppError(err) {
		return err
	}

	if xerrors.Is(err, pgx.ErrNoRows) {
		return xerrors.Join(apperror.ErrNotFound, err)
	}

	var pgErr *pgconn.PgError
	if xerrors.As(err, &pgErr) {
		if appErr, ok := sqlstateToAppError[pgErr.Code]; ok {
			return xerrors.Join(appErr, err)
		}
	}

	if xerrors.Is(err, context.Canceled) {
		return xerrors.Join(apperror.ErrCanceled, err)
	}

	if IsUnavailable(err) {
		return xerrors.Join(apperror.ErrUnavailable, err)
	}

	return xerrors.Join(apperror.ErrInternal, err)
}

// NormalizeReconstructError は、DB 行からのエンティティ再構築（ドメインコンストラクタ）が
// 返したエラーをデータ不整合として ErrInternal へ平坦化します。
// 元エラーの分類センチネルと apperror.Meta を意図的にチェーンから消します（load-bearing flatten）。
// 理由文はメッセージに残るためログには出ます。背景は README を参照。err が nil の場合は nil を返します。
func NormalizeReconstructError(err error) error {
	if err == nil {
		return nil
	}
	return xerrors.Wrap(apperror.ErrInternal, "entity reconstruction from stored row failed: "+err.Error())
}

// NormalizeExecResult は、影響行数を返す書き込み系クエリの結果を正規化します。
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

// IsUnavailable は、DB が利用不可能な状態を示すエラーかを判定します。context.DeadlineExceeded・net.Error・PostgreSQL 接続例外クラス(08xxx) を対象とします。
func IsUnavailable(err error) bool {
	if err == nil {
		return false
	}

	if xerrors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var ne net.Error
	if xerrors.As(err, &ne) {
		return true
	}

	return isPgConnectionError(err)
}

// isPgConnectionError は、与えられたエラーがPostgreSQLの接続エラーであるかを判定します。
func isPgConnectionError(err error) bool {
	var pgErr *pgconn.PgError
	if !xerrors.As(err, &pgErr) {
		return false
	}
	return strings.HasPrefix(pgErr.Code, "08")
}

// IsNoRows は、クエリが 1 行も返さなかったことを示すエラーかを判定します。
// 条件付き UPDATE（楽観ロック等）では対象行なしが「未存在」ではなく「条件不一致」を意味するため、
// NormalizeError による ErrNotFound への一律の写像に代えて、呼び出し側が固有の意味を与えるための述語です。
func IsNoRows(err error) bool {
	return xerrors.Is(err, pgx.ErrNoRows)
}

// IsLockNotAvailable は、lock_timeout 失効によるエラーであるかを判定します。
func IsLockNotAvailable(err error) bool {
	var pgErr *pgconn.PgError
	return xerrors.As(err, &pgErr) && pgErr.Code == "55P03"
}

// IsRetryableTxError は、リトライで解消しうる tx エラーかを判定します。
// 40001 = serialization_failure, 40P01 = deadlock_detected。
// 写像後の sentinel（両者は ErrUnavailable へ写像される）ではなく生 SQLSTATE で判定し、
// 接続断など他の ErrUnavailable をリトライ対象に巻き込まないようにします。
func IsRetryableTxError(err error) bool {
	var pgErr *pgconn.PgError
	return xerrors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "40P01")
}
