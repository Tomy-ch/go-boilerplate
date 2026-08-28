package outbox

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

const (
	// ChannelHTTP は、受信エンドポイントへ HTTP で配送するレーンです。
	ChannelHTTP Channel = "http"
	// ChannelRealtime は、Realtime Delivery の EventLog へ配送するレーンです。順序保証を持ちます。
	ChannelRealtime Channel = "realtime"
)

// ErrUnknownChannel は、既知でない配送チャネルが指定されたことを示すエラーです。
var ErrUnknownChannel = xerrors.Wrap(apperror.ErrInvalidArgument, "unknown outbox delivery channel")

// Channel は、outbox 行の配送レーンです。relay は 1 つの Channel だけを claim するため、
// あるレーンの遅延・障害・再試行待ちが別のレーンの進行を止めません。
type Channel string

// ParseChannel は、文字列を Channel へ変換します。既知でない値は ErrUnknownChannel を返します。
// 未知のチャネルの行はどの relay も claim しないため、値の検証は emit と relay 起動の両方で行います。
func ParseChannel(s string) (Channel, error) {
	switch c := Channel(s); c {
	case ChannelHTTP, ChannelRealtime:
		return c, nil
	default:
		return "", xerrors.Wrap(ErrUnknownChannel, "delivery channel must be one of: http, realtime")
	}
}

// String は、Channel の文字列表現を返します。
func (c Channel) String() string { return string(c) }
