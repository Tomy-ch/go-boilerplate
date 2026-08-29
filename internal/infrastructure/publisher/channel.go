package publisher

import (
	"go-boilerplate/internal/apperror"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/pkg/xerrors"
)

// ErrChannelUnsupported は、起動した配送チャネルを配送できる publish 実装が無いことを示すエラーです。
var ErrChannelUnsupported = xerrors.Wrap(apperror.ErrInvalidArgument, "unsupported outbox delivery channel")

// VerifyChannel は、この package が提供する publish 実装が channel を配送できるかを検査します。
// 配送できない channel は起動時に落とします（fail-closed の理由と channel ごとの担当は
// README の Design Policy を参照）。
// 現在の実装（HTTP / SQS 互換ブローカー）はいずれも順序を持たない HTTP チャネルの配送先です。
func VerifyChannel(channel outboxbndry.Channel) error {
	if channel == outboxbndry.ChannelHTTP {
		return nil
	}
	return xerrors.Wrap(ErrChannelUnsupported, "no publisher implementation serves delivery channel "+channel.String())
}
