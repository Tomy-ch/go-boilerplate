package realtime

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

var (
	// ErrCursorExpired は、cursor が replay floor より前で、そこからの replay が成り立たないことを示します。
	// client は canonical recovery path（History の再読み込み）へ戻ります。apperror の taxonomy には
	// 対応する分類（410）が無く、呼び出し元も本 package の外にまだ無いため、機構固有の sentinel として持ちます。
	ErrCursorExpired = xerrors.New("realtime: stream cursor expired")
	// ErrTicketInvalid は、提示された ticket が無い・期限切れ・destination 違いのいずれかであることを示します。
	// 認証の失敗として扱われます。
	ErrTicketInvalid = xerrors.Wrap(apperror.ErrUnauthenticated, "realtime: ticket invalid")
)
