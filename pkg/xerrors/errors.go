package xerrors

import (
	"github.com/cockroachdb/errors"
)

type CockroachDBError struct{}

func New() *CockroachDBError {
	return &CockroachDBError{}
}

func (CockroachDBError) New(msg string) error {
	return errors.New(msg)
}

func (CockroachDBError) Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

func (CockroachDBError) Is(err, target error) bool {
	return errors.Is(err, target)
}

func (CockroachDBError) As(err error, target any) bool {
	return errors.As(err, target)
}
