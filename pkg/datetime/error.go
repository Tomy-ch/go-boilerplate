package datetime

import "go-boilerplate/pkg/xerrors"

// errNilLocation は、変換先のタイムゾーンに nil が渡されたことを示すエラーです。
var errNilLocation = xerrors.New("loc must not be nil")
