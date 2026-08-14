// Package period は、購入の集計・絞り込みで用いる対象期間の指定と、その暦日境界への解決を提供します。
package period

import (
	"fmt"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

const (
	// KindAll は、期間で絞り込まない区分です。
	KindAll Kind = "all"
	// KindMonth は、Spec.Month が指す暦月を対象とする区分です。
	KindMonth Kind = "month"
	// KindRange は、Spec.From / Spec.To が指す期間を対象とする区分です。
	KindRange Kind = "range"
	// KindRecent は、今日から Spec.Days 日前までを対象とする区分です。
	KindRecent Kind = "recent"
)

const (
	// monthLayout は、Spec.Month の書式です。
	monthLayout = "2006-01"
	// minRecentDays は、KindRecent で遡れる最小日数です。
	minRecentDays = 1
)

// Kind は、対象期間の区分です。
type Kind string

// Spec は、利用者が要求した対象期間の指定です。区分ごとに参照するフィールドが異なり、
// 該当しないフィールドは解決時に参照しません。未知の区分と空文字は KindAll として扱います。
type Spec struct {
	// Kind は、対象期間の区分です。
	Kind Kind
	// From / To は、Kind が KindRange のときの開始日・終了日です（両端を含みます）。
	// 暦日としての年月日のみが意味を持ち、時刻部分は解釈しません。
	From *time.Time
	To   *time.Time
	// Month は、Kind が KindMonth のときの対象月（"YYYY-MM"）です。
	Month *string
	// Days は、Kind が KindRecent のときに今日から遡る日数です。
	Days *int
}

// Window は、解決済みの対象期間です。両端を含む暦日で表し、期間で絞り込まない場合は Filtered が false です。
// ゼロ値は「絞り込まない」を表します。
type Window struct {
	from     time.Time
	to       time.Time
	filtered bool
}

// Filtered は、期間で絞り込むかどうかを返します。false の場合、From / To / Bounds の値は意味を持ちません。
func (w Window) Filtered() bool { return w.filtered }

// From は、対象期間の開始日（この日を含む暦日）を返します。
func (w Window) From() time.Time { return w.from }

// To は、対象期間の終了日（この日を含む暦日）を返します。
func (w Window) To() time.Time { return w.to }

// Bounds は、SQL 述語で用いる半開区間の下限と上限をこの順で返します。終了日の翌日を上限に取ることで、
// 注文日時の時刻成分に関わらず終了日 1 日分を丸ごと含めます。
func (w Window) Bounds() (time.Time, time.Time) {
	return w.from, w.to.AddDate(0, 0, 1)
}

// Resolve は、要求された期間指定を loc の暦日基準の対象期間へ解決します。
// now は今日の判定にのみ用い、KindRange / KindMonth では参照しません。
// 区分ごとの必須指定の欠落・書式不正、および終了日が開始日より前の場合は ErrInvalidArgument を返します。
func Resolve(spec Spec, now time.Time, loc *time.Location) (Window, error) {
	switch spec.Kind {
	case KindMonth:
		return resolveMonth(spec, loc)
	case KindRange:
		return resolveRange(spec, loc)
	case KindRecent:
		return resolveRecent(spec, now, loc)
	case KindAll:
		return Window{}, nil
	default:
		return Window{}, nil
	}
}

// resolveMonth は、暦月指定を月初日から月末日までの対象期間へ解決します。
// 月末日は翌月の月初日から 1 日戻して求めるため、月の日数と閏年を場合分けせずに済みます。
func resolveMonth(spec Spec, loc *time.Location) (Window, error) {
	if spec.Month == nil {
		return Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "period=month requires month")
	}
	first, err := time.ParseInLocation(monthLayout, *spec.Month, loc)
	if err != nil {
		return Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "month must be in YYYY-MM format: "+err.Error())
	}
	return newWindow(first, first.AddDate(0, 1, -1)), nil
}

// resolveRange は、開始日・終了日の指定を対象期間へ解決します。
func resolveRange(spec Spec, loc *time.Location) (Window, error) {
	if spec.From == nil || spec.To == nil {
		return Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "period=range requires both from and to")
	}
	from, to := dateOnly(*spec.From, loc), dateOnly(*spec.To, loc)
	if to.Before(from) {
		return Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "to must not be before from")
	}
	return newWindow(from, to), nil
}

// resolveRecent は、相対指定を今日を終了日とする対象期間へ解決します。
// 開始日は今日から Days 日前で、両端を含むため対象は Days + 1 日分になります。
func resolveRecent(spec Spec, now time.Time, loc *time.Location) (Window, error) {
	if spec.Days == nil {
		return Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "period=recent requires days")
	}
	if *spec.Days < minRecentDays {
		return Window{}, xerrors.Wrap(
			apperror.ErrInvalidArgument,
			fmt.Sprintf("days must be %d or greater, got %d", minRecentDays, *spec.Days),
		)
	}
	// 今日は loc の暦日で判定するため、現在時刻を loc へ移してから年月日を取る。
	today := dateOnly(now.In(loc), loc)
	return newWindow(today.AddDate(0, 0, -*spec.Days), today), nil
}

// newWindow は、両端を含む暦日から絞り込み済みの対象期間を組み立てます。
func newWindow(from, to time.Time) Window {
	return Window{from: from, to: to, filtered: true}
}

// dateOnly は、t が表す年月日の開始時刻を loc のゾーンで返します。年月日は t 自身のロケーションで解釈し、
// loc は返り値のゾーンとしてのみ用います。利用者が指定した暦日をそのまま保つためで、t を loc へ変換すると
// UTC より西のロケーションでは前日へずれます。
func dateOnly(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
