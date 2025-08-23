package xerrors

import (
	"github.com/cockroachdb/errors"
)

func New(msg string) error {
	return errors.New(msg)
}

func Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
