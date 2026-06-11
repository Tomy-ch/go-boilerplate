package xerrors

// stdErrors は、Errors を cockroachdb/errors ベースで実装する標準実装です。
type stdErrors struct{}

// NewErrors は、Errors の標準実装を返します。
func NewErrors() Errors {
	return stdErrors{}
}

func (stdErrors) New(msg string) error {
	return New(msg)
}

func (stdErrors) Wrap(err error, msg string) error {
	return Wrap(err, msg)
}

func (stdErrors) Is(err, target error) bool {
	return Is(err, target)
}

func (stdErrors) As(err error, target any) bool {
	return As(err, target)
}

func (stdErrors) StackTrace(err error) string {
	return StackTrace(err)
}
