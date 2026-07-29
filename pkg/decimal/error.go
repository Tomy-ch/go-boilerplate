package decimal

import "go-boilerplate/pkg/xerrors"

// ErrInvalid は、十進量への解析・復元に失敗したことを示すエラーです。
var ErrInvalid = xerrors.New("invalid decimal")

// ErrOverflow は、最小単位整数への変換結果が int64 の範囲を超えたことを示すエラーです。
var ErrOverflow = xerrors.New("decimal overflow")
