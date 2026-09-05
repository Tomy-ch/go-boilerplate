package inquiry

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// errMessageNotStored は、追加したメッセージを読み直せなかったことを表します。
// 採番と追加は同じ tx の中にあるため、ここへ到達するのは基盤側の不整合です。
var errMessageNotStored = xerrors.Wrap(apperror.ErrInternal, "appended inquiry message is missing")
