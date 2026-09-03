package realtime

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrCursorExpired は、cursor が replay floor より前で、そこからの replay が成り立たないことを示します。
	// client は canonical recovery path（History の再読み込み）へ戻ります。HTTP への写像は README を参照。
	ErrCursorExpired = xerrors.New("realtime: stream cursor expired")
	// ErrTicketInvalid は、提示された ticket が無い・期限切れ・destination 違いのいずれかであることを示します。
	// 認証の失敗として扱われます。
	ErrTicketInvalid = xerrors.Wrap(apperror.ErrUnauthenticated, "realtime: ticket invalid")
	// ErrFanoutUnreachable は、fan-out から通知を受け取れていないことを示します。
	// 稼働中の縮退を表すもので、既存の接続は周期の catch-up で配信を続けます。
	ErrFanoutUnreachable = xerrors.Wrap(apperror.ErrUnavailable, "realtime: fan-out unreachable")
)
