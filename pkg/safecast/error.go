package safecast

import (
	"errors"
	"fmt"
)

var (
	// errSafecast は、safecastパッケージで使用されるエラーです。
	errSafecast = errors.New("safecast error")

	// ErrOverflow は、型変換時にオーバーフローが発生したことを示すエラーです。
	ErrOverflow = fmt.Errorf(
		"overflow error: value exceeds maximum limit for conversion: %w",
		errSafecast,
	)
)
