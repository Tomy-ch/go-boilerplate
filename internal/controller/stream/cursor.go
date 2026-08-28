package stream

import (
	"regexp"
	"strconv"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// cursorRe は、wire 上の cursor の形（符号も先頭ゼロも無い 10 進）です。桁数の上限は持たず、範囲は parseCursor が判定します。
var cursorRe = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

// resolveCursor は、Last-Event-ID → after → ticket の初期位置の順で接続の cursor を決めます。
func resolveCursor(lastEventID, after *string, initial rt.Sequence) (rt.Sequence, error) {
	switch {
	case lastEventID != nil:
		return parseCursor(*lastEventID)
	case after != nil:
		return parseCursor(*after)
	default:
		return initial, nil
	}
}

// parseCursor は、10 進文字列の cursor を Sequence にします。形式不正・負数・int64 を超える値は ErrCursorMalformed です。
func parseCursor(s string) (rt.Sequence, error) {
	if !cursorRe.MatchString(s) {
		return 0, xerrors.Wrap(ErrCursorMalformed, "cursor must be a decimal without sign or leading zeros")
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, xerrors.Wrap(ErrCursorMalformed, "cursor exceeds the representable range")
	}

	return rt.Sequence(v), nil
}
