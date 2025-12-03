package usecasetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExpectedDBError(t *testing.T) {
	t.Parallel()

	actual := ExpectedDBError(t)
	require.Error(t, actual)
}

func Test_NewTestInstanceForNew(t *testing.T) {
	t.Parallel()
	ctrl, tf := NewTestInstanceForNew(t)

	require.NotNil(t, ctrl)
	require.NotNil(t, tf)
}

func Test_NewTestInstancesForImplementedUsecase(t *testing.T) {
	t.Parallel()
	expectedCtx := context.Background()
	expectedLoc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	actualCtx, actualCtrl, actualLocation, actualLt := NewTestInstanceForImplementedUsecase(t)
	require.Equal(t, expectedCtx, actualCtx)
	require.NotNil(t, actualCtrl)
	require.Equal(t, expectedLoc, actualLocation)
	require.NotNil(t, actualLt)
}
