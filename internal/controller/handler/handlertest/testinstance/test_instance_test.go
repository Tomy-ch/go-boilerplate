package testinstance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewBindHandlerTestInstance(t *testing.T) {
	t.Parallel()
	e, ctrl, tp, log := NewTestInstanceForBindHandler(t)

	require.NotNil(t, e)
	require.NotNil(t, ctrl)
	require.NotNil(t, tp)
	require.NotNil(t, log)
}

func TestNewImplementHandlerTestInstances(t *testing.T) {
	t.Parallel()
	expectedCtx := context.Background()
	expectedLoc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	actualCtx, actualCtrl, actualLoc, actualLt := NewTestInstancesForImplementedUsecase(t)

	require.Equal(t, expectedCtx, actualCtx)
	require.NotNil(t, actualCtrl)
	require.Equal(t, expectedLoc, actualLoc)
	require.NotNil(t, actualLt)
}
