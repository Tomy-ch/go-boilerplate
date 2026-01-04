package shutdowner

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

type mockShutdowner struct {
	shutdownFunc func(opts ...fx.ShutdownOption) error
}

func (m *mockShutdowner) Shutdown(opts ...fx.ShutdownOption) error {
	return m.shutdownFunc(opts...)
}

func Test_Shutdowner_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("Shutdownがfx.ShutdownerのShutdownを呼び出す", func(t *testing.T) {
		t.Parallel()

		called := false
		mockShutdowner := &mockShutdowner{
			shutdownFunc: func(_ ...fx.ShutdownOption) error {
				called = true
				return nil
			},
		}

		shutdowner := NewShutdowner(mockShutdowner)

		err := shutdowner.Shutdown()
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("Shutdownがfx.ShutdownerのShutdownのエラーを返す", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("shutdown error")
		mockShutdowner := &mockShutdowner{
			shutdownFunc: func(_ ...fx.ShutdownOption) error {
				return expectedErr
			},
		}

		shutdowner := NewShutdowner(mockShutdowner)

		err := shutdowner.Shutdown()
		require.ErrorIs(t, err, expectedErr)
	})
}
