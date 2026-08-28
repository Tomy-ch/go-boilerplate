package publisher

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/pkg/xerrors"
)

// classifyOutcome は、publish 試行の結果を dead 判定用の分類（ErrPermanent / ErrRetryable）で包んで返します。
// 判定そのものは substrate の再試行 verdict をそのまま用います。ここで HTTP ステータスの表を書き直すと、
// substrate 内部にしか無い決定的失敗（レスポンス上限超過など）を取りこぼし、分類が二重管理になります。
// ctx キャンセルは配送の失敗ではなく停止なので分類せず、そのまま返します（呼び出し側は未分類を一時失敗として扱う）。
func classifyOutcome(resp *httpclient.Response, err error) error {
	if err == nil {
		return nil
	}
	if xerrors.Is(err, apperror.ErrCanceled) {
		return err
	}
	if httpclient.RetryableOutcome(resp, err) {
		return xerrors.Join(apperror.ErrRetryable, err)
	}
	return xerrors.Join(apperror.ErrPermanent, err)
}
