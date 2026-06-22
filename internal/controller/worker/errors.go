// Package worker は、pull-ack クラスのキューを consume する worker engine を提供します。
// engine は seam（internal/usecase/boundary/worker の port）のみに依存し、broker 実装には依存しません。
package worker

import "go-boilerplate/pkg/xerrors"

var (
	// ErrDuplicateWorker は、同名の worker が重複登録された場合のエラーです。
	ErrDuplicateWorker = xerrors.New("duplicate worker")
	// ErrUnknownWorker は、未登録の worker 名が指定された場合のエラーです。
	ErrUnknownWorker = xerrors.New("unknown worker")
)
