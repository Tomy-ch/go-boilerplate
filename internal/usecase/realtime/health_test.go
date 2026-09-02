package realtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
	"go-boilerplate/pkg/xerrors"
)

// newHealth は、構築子を通して Health を組み立てます。
func newHealth(t *testing.T) (*Health, *mock_realtime.MockEventLogStore) {
	t.Helper()

	log := mock_realtime.NewMockEventLogStore(gomock.NewController(t))

	return NewHealth(log), log
}

func TestNewHealth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fan-out を健全な状態から始める", func(t *testing.T) {
			t.Parallel()

			h, _ := newHealth(t)

			// 起動直後は 1 度も受信していないが、そこで縮退を名乗ると
			// consumer が最初の受信を終えるまで新規接続を全部断ってしまう。
			assert.False(t, h.FanoutDegraded())
		})
	})
}

func TestHealth_ObserveFanout(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受信の失敗で縮退し、成功で健全へ戻る", func(t *testing.T) {
			t.Parallel()

			h, _ := newHealth(t)

			h.ObserveFanout(xerrors.New("receive failed"))
			assert.True(t, h.FanoutDegraded())

			// 戻れることが要点。片道にすると、一時的な不調のあと接続を受け付けられなくなる。
			h.ObserveFanout(nil)
			assert.False(t, h.FanoutDegraded())
		})
	})
}

func TestHealth_FanoutDegraded(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最後に観測した受信の可否を返す", func(t *testing.T) {
			t.Parallel()

			h, _ := newHealth(t)
			h.ObserveFanout(xerrors.New("receive failed"))

			assert.True(t, h.FanoutDegraded())
		})
	})
}

func TestHealth_Check(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EventLog に届き fan-out も健全なら nil を返す", func(t *testing.T) {
			t.Parallel()

			h, log := newHealth(t)
			// 検査用の stream は誰も書かないので「無い」が正常。届いたかどうかだけを見る。
			log.EXPECT().Latest(gomock.Any(), healthProbeStreamID).Return(rt.DeliveryEvent{}, false, nil)

			require.NoError(t, h.Check(t.Context()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EventLog に届かなければその失敗を返す", func(t *testing.T) {
			t.Parallel()

			h, log := newHealth(t)
			log.EXPECT().
				Latest(gomock.Any(), healthProbeStreamID).
				Return(rt.DeliveryEvent{}, false, apperror.ErrUnavailable)

			require.ErrorIs(t, h.Check(t.Context()), apperror.ErrUnavailable)
		})

		t.Run("EventLog に届いても fan-out が不調なら縮退を返す", func(t *testing.T) {
			t.Parallel()

			h, log := newHealth(t)
			log.EXPECT().Latest(gomock.Any(), healthProbeStreamID).Return(rt.DeliveryEvent{}, false, nil)
			h.ObserveFanout(xerrors.New("receive failed"))

			require.ErrorIs(t, h.Check(t.Context()), ErrFanoutUnreachable)
		})
	})
}
