package period

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLoc は、注入されたロケーションが使われることを設定値と独立に固定するための、UTC から離れた固定ゾーンです。
var testLoc = time.FixedZone("TEST+09", 9*60*60)

// testNow は、testLoc で 2026-01-31 12:00 に相当する時刻です。UTC のままだと暦日が 1 日ずれる位置を選んでいます。
var testNow = time.Date(2026, time.January, 31, 3, 0, 0, 0, time.UTC)

// testDate は、testLoc の暦日の開始時刻を返します。
func testDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, testLoc)
}

// utcDate は、OpenAPI の date が解析される形（UTC 深夜）で testNow と同じ年月の暦日を返します。
func utcDate(day int) time.Time {
	return time.Date(testNow.Year(), testNow.Month(), day, 0, 0, 0, 0, time.UTC)
}

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("暦月指定を暦月の解決へ振り分ける", func(t *testing.T) {
			t.Parallel()

			month := "2026-01"
			got, err := Resolve(Spec{Kind: KindMonth, Month: &month}, testNow, testLoc)
			require.NoError(t, err)

			assert.True(t, got.Filtered())
			assert.True(t, testDate(2026, time.January, 1).Equal(got.From()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.To()))
		})

		t.Run("範囲指定を範囲の解決へ振り分ける", func(t *testing.T) {
			t.Parallel()

			from, to := utcDate(21), utcDate(31)
			got, err := Resolve(Spec{Kind: KindRange, From: &from, To: &to}, testNow, testLoc)
			require.NoError(t, err)

			assert.True(t, got.Filtered())
			assert.True(t, testDate(2026, time.January, 21).Equal(got.From()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.To()))
		})

		t.Run("相対指定を相対の解決へ振り分ける", func(t *testing.T) {
			t.Parallel()

			days := 10
			got, err := Resolve(Spec{Kind: KindRecent, Days: &days}, testNow, testLoc)
			require.NoError(t, err)

			assert.True(t, got.Filtered())
			assert.True(t, testDate(2026, time.January, 21).Equal(got.From()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.To()))
		})

		t.Run("区分が未指定のとき期間で絞り込まない", func(t *testing.T) {
			t.Parallel()

			got, err := Resolve(Spec{}, testNow, testLoc)
			require.NoError(t, err)
			assert.False(t, got.Filtered())
		})

		t.Run("全期間の区分では期間で絞り込まない", func(t *testing.T) {
			t.Parallel()

			got, err := Resolve(Spec{Kind: KindAll}, testNow, testLoc)
			require.NoError(t, err)
			assert.False(t, got.Filtered())
		})

		t.Run("未知の区分は全期間として扱い期間で絞り込まない", func(t *testing.T) {
			t.Parallel()

			got, err := Resolve(Spec{Kind: "unknown"}, testNow, testLoc)
			require.NoError(t, err)
			assert.False(t, got.Filtered())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("暦月指定でmonthが無いときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			_, err := Resolve(Spec{Kind: KindMonth}, testNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("範囲指定でfromとtoが無いときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			_, err := Resolve(Spec{Kind: KindRange}, testNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("相対指定でdaysが無いときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			_, err := Resolve(Spec{Kind: KindRecent}, testNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_resolveMonth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("月初日から月末日までを対象にし現在時刻に依存しない", func(t *testing.T) {
			t.Parallel()

			month := "2020-06"
			got, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.NoError(t, err)

			assert.True(t, got.Filtered())
			assert.True(t, testDate(2020, time.June, 1).Equal(got.From()))
			assert.True(t, testDate(2020, time.June, 30).Equal(got.To()))
		})

		// 月末日は翌月の月初から 1 日戻して求めるため、場合分けなしで 28 / 29 / 31 日を返せる。
		t.Run("平年2月の月末日は28日になる", func(t *testing.T) {
			t.Parallel()

			month := "2026-02"
			got, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.NoError(t, err)
			assert.True(t, testDate(2026, time.February, 28).Equal(got.To()))
		})

		t.Run("閏年2月の月末日は29日になる", func(t *testing.T) {
			t.Parallel()

			month := "2028-02"
			got, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.NoError(t, err)
			assert.True(t, testDate(2028, time.February, 29).Equal(got.To()))
		})

		t.Run("31日ある月の月末日は31日になる", func(t *testing.T) {
			t.Parallel()

			month := "2026-12"
			got, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.NoError(t, err)
			assert.True(t, testDate(2026, time.December, 31).Equal(got.To()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("monthが無いときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			_, err := resolveMonth(Spec{}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("monthの月が範囲外のときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			month := "2026-13"
			_, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("monthの区切り文字が異なるときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			month := "2026/01"
			_, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("monthに区切り文字が無いときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			month := "202601"
			_, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("monthが数値でないときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			month := "abc"
			_, err := resolveMonth(Spec{Month: &month}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_resolveRange(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定された暦日をそのまま両端に採る", func(t *testing.T) {
			t.Parallel()

			from, to := utcDate(21), utcDate(31)
			got, err := resolveRange(Spec{From: &from, To: &to}, testLoc)
			require.NoError(t, err)

			assert.True(t, testDate(2026, time.January, 21).Equal(got.From()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.To()))
		})

		t.Run("開始日と終了日が同じ単日でも受理する", func(t *testing.T) {
			t.Parallel()

			day := utcDate(31)
			got, err := resolveRange(Spec{From: &day, To: &day}, testLoc)
			require.NoError(t, err)

			assert.True(t, testDate(2026, time.January, 31).Equal(got.From()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.To()))
		})

		t.Run("時刻成分を落として暦日の開始時刻へ揃える", func(t *testing.T) {
			t.Parallel()

			from := time.Date(2026, time.January, 21, 23, 59, 59, 0, time.UTC)
			to := time.Date(2026, time.January, 31, 13, 30, 0, 0, time.UTC)
			got, err := resolveRange(Spec{From: &from, To: &to}, testLoc)
			require.NoError(t, err)

			assert.True(t, testDate(2026, time.January, 21).Equal(got.From()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.To()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("開始日と終了日の両方が欠けているときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			_, err := resolveRange(Spec{}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("開始日だけが欠けているときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			day := utcDate(31)
			_, err := resolveRange(Spec{To: &day}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("終了日だけが欠けているときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			day := utcDate(31)
			_, err := resolveRange(Spec{From: &day}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("終了日が開始日より前のときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			from, to := utcDate(31), utcDate(21)
			_, err := resolveRange(Spec{From: &from, To: &to}, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_resolveRecent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("今日を終了日としdays日前を開始日にする", func(t *testing.T) {
			t.Parallel()

			days := 10
			got, err := resolveRecent(Spec{Days: &days}, testNow, testLoc)
			require.NoError(t, err)

			assert.True(t, got.Filtered())
			assert.True(t, testDate(2026, time.January, 21).Equal(got.From()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.To()))
		})

		t.Run("今日はUTCではなく指定ロケーションの暦日で決まる", func(t *testing.T) {
			t.Parallel()

			// UTC では 2026-01-31 22:00 だが testLoc では 2026-02-01。ロケーションを無視すると終了日が前日になる。
			crossing := time.Date(2026, time.January, 31, 22, 0, 0, 0, time.UTC)
			days := 1
			got, err := resolveRecent(Spec{Days: &days}, crossing, testLoc)
			require.NoError(t, err)

			assert.True(t, testDate(2026, time.February, 1).Equal(got.To()))
			assert.True(t, testDate(2026, time.January, 31).Equal(got.From()))
		})

		t.Run("開始日は月をまたいでも正しく遡る", func(t *testing.T) {
			t.Parallel()

			days := 40
			got, err := resolveRecent(Spec{Days: &days}, testNow, testLoc)
			require.NoError(t, err)
			assert.True(t, testDate(2025, time.December, 22).Equal(got.From()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("daysが無いときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			_, err := resolveRecent(Spec{}, testNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("daysが0のときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			// OpenAPI の minimum が先に弾く境界だが、ユースケースを直接呼ぶ経路のための安全弁として固定する。
			days := 0
			_, err := resolveRecent(Spec{Days: &days}, testNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("daysが負数のときInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			days := -1
			_, err := resolveRecent(Spec{Days: &days}, testNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_newWindow(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("両端を含む暦日から絞り込み済みの期間を組み立てる", func(t *testing.T) {
			t.Parallel()

			from, to := testDate(2026, time.January, 21), testDate(2026, time.January, 31)
			got := newWindow(from, to)

			assert.True(t, got.Filtered())
			assert.True(t, from.Equal(got.From()))
			assert.True(t, to.Equal(got.To()))
		})
	})
}

func Test_dateOnly(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("時刻成分を落として暦日の開始時刻を返す", func(t *testing.T) {
			t.Parallel()

			got := dateOnly(time.Date(2026, time.January, 31, 13, 45, 30, 999, time.UTC), testLoc)
			assert.True(t, testDate(2026, time.January, 31).Equal(got))
		})

		t.Run("年月日は引数のロケーションのまま解釈しlocへ変換し直さない", func(t *testing.T) {
			t.Parallel()

			// UTC より西のゾーンへ変換してしまうと前日へずれる。利用者が指定した暦日を保つ契約。
			west := time.FixedZone("TEST-08", -8*60*60)
			got := dateOnly(time.Date(2026, time.January, 31, 2, 0, 0, 0, time.UTC), west)

			assert.Equal(t, 2026, got.Year())
			assert.Equal(t, time.January, got.Month())
			assert.Equal(t, 31, got.Day())
			assert.Equal(t, west, got.Location())
		})
	})
}

func TestWindow_Filtered(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値は絞り込まないことを表す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, Window{}.Filtered())
		})

		t.Run("暦日から組み立てた期間は絞り込むことを表す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, newWindow(testDate(2026, time.January, 21), testDate(2026, time.January, 31)).Filtered())
		})
	})
}

func TestWindow_From(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象期間の開始日を返す", func(t *testing.T) {
			t.Parallel()

			from := testDate(2026, time.January, 21)
			assert.True(t, from.Equal(newWindow(from, testDate(2026, time.January, 31)).From()))
		})

		t.Run("絞り込まない期間ではゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, Window{}.From().IsZero())
		})
	})
}

func TestWindow_To(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象期間の終了日を返す", func(t *testing.T) {
			t.Parallel()

			to := testDate(2026, time.January, 31)
			assert.True(t, to.Equal(newWindow(testDate(2026, time.January, 21), to).To()))
		})

		t.Run("絞り込まない期間ではゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, Window{}.To().IsZero())
		})
	})
}

func TestWindow_Bounds(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限は終了日の翌日になり終了日の注文を時刻に関わらず含む", func(t *testing.T) {
			t.Parallel()

			after, before := newWindow(testDate(2026, time.January, 21), testDate(2026, time.January, 31)).Bounds()

			assert.True(t, testDate(2026, time.January, 21).Equal(after))
			assert.True(t, testDate(2026, time.February, 1).Equal(before))
		})

		t.Run("月をまたぐ終了日でも翌日が正しく求まる", func(t *testing.T) {
			t.Parallel()

			_, before := newWindow(testDate(2026, time.January, 1), testDate(2026, time.February, 28)).Bounds()
			assert.True(t, testDate(2026, time.March, 1).Equal(before))
		})

		t.Run("年をまたぐ終了日でも翌日が正しく求まる", func(t *testing.T) {
			t.Parallel()

			_, before := newWindow(testDate(2026, time.January, 1), testDate(2026, time.December, 31)).Bounds()
			assert.True(t, testDate(2027, time.January, 1).Equal(before))
		})
	})
}
