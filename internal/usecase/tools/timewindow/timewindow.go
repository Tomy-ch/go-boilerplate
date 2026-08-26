// Package timewindow は、集計・絞り込みの対象期間を表す半開区間を提供します。
package timewindow

import (
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// Window は、対象期間を表す半開区間 [After, Before) です。
//
//	下限は含み、上限は含みません（半開である理由は README の Role を参照）。
//	いずれの境界も省略でき、境界を持たない側には制限を設けません。ゼロ値は全期間を表します。
type Window struct {
	after  *time.Time
	before *time.Time
}

// Bounds は、Window の境界指定です。nil の境界は、その側に制限を設けないことを表します。
type Bounds struct {
	// After は、対象期間の下限です。この瞬時を含みます。
	After *time.Time
	// Before は、対象期間の上限です。この瞬時を含みません。
	Before *time.Time
}

// New は、境界指定から Window を生成します。
//
//	両方の境界が指定され、かつ Before が After 以前（同値を含む）の場合は
//	apperror.ErrInvalidArgument を返します（拒否する理由は README の Behavior を参照）。
func New(bounds Bounds) (Window, error) {
	if bounds.After != nil && bounds.Before != nil && !bounds.Before.After(*bounds.After) {
		return Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "orderedBefore must be after orderedAfter")
	}

	return Window{after: copyTime(bounds.After), before: copyTime(bounds.Before)}, nil
}

// After は、対象期間の下限（この瞬時を含む）を返します。下限を設けない場合は nil です。
func (w Window) After() *time.Time { return copyTime(w.after) }

// Before は、対象期間の上限（この瞬時を含まない）を返します。上限を設けない場合は nil です。
func (w Window) Before() *time.Time { return copyTime(w.before) }

// copyTime は、境界の複製を返します。呼び出し元と Window が同じ時刻を共有しないようにするためです。
func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	copied := *t
	return &copied
}
