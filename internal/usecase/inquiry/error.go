package inquiry

import (
	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// errMessageNotStored は、追加したメッセージを読み直せなかったことを表します。
// 採番と追加は同じ tx の中にあるため、ここへ到達するのは基盤側の不整合です。
var errMessageNotStored = xerrors.Wrap(apperror.ErrInternal, "appended inquiry message is missing")

// errInquiryCreationRace は、問い合わせを作ろうとして他の要求に先を越されたことを表します。
// やり直しても解けなかった場合はこのまま外へ出るため、409 へ写る apperror.ErrConflict を根に持ちます
// （docs/spec/usecase/inquiry.md の AppendMessage）。
var errInquiryCreationRace = xerrors.Wrap(apperror.ErrConflict, "inquiry creation lost a race")
