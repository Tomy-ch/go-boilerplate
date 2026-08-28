package realtime

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrCursorExpired は、cursor が replay floor より前で、そこからの replay が成り立たないことを示します。
	// client は canonical recovery path（History の再読み込み）へ戻ります。HTTP を知らない層の sentinel として持ち、
	// 410 への写像は controller（stream handler）が apperror.ErrGone を重ねて行います。
	ErrCursorExpired = xerrors.New("realtime: stream cursor expired")
	// ErrTicketInvalid は、提示された ticket が無い・期限切れ・destination 違いのいずれかであることを示します。
	// 認証の失敗として扱われます。
	ErrTicketInvalid = xerrors.Wrap(apperror.ErrUnauthenticated, "realtime: ticket invalid")
)
