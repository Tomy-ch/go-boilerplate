package publisher

import (
	"go-boilerplate/internal/apperror"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/pkg/xerrors"
)

// ErrChannelUnsupported は、起動した配送チャネルを配送できる publish 実装が無いことを示すエラーです。
var ErrChannelUnsupported = xerrors.Wrap(apperror.ErrInvalidArgument, "unsupported outbox delivery channel")

// VerifyChannel は、この package が提供する publish 実装が channel を配送できるかを検査します。
// 配送できない組み合わせで relay を起動すると、そのチャネルの行が意図しない送信先へ流れるか、
// 誰にも配送されないまま滞留します。どちらも無言で起きるため、起動時に落とします（fail-closed）。
// 現在の実装（HTTP / SQS 互換ブローカー）はいずれも順序を持たない HTTP チャネルの配送先です。
func VerifyChannel(channel outboxbndry.Channel) error {
	if channel == outboxbndry.ChannelHTTP {
		return nil
	}
	return xerrors.Wrap(ErrChannelUnsupported, "no publisher implementation serves delivery channel "+channel.String())
}
