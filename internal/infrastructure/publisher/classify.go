package publisher

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/pkg/xerrors"
)

// classifyOutcome は、publish 試行の結果を dead 判定用の分類（ErrPermanent / ErrRetryable）で包んで返します。
// 判定は httpclient.RetryableOutcome へ委譲します（理由は README の Design Policy を参照）。
// ctx キャンセルは配送の失敗ではなく停止なので分類せず、そのまま返します。
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
