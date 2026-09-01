package event_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/usecase/inquiry/event"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

func TestBuildThreadUpdated(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧の更新に要る値だけを載せ本文を持たない", func(t *testing.T) {
			t.Parallel()
			updatedAt := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
			i, err := inquiry.Reconstruct(uuidtestkit.NewTestFromSalt(t, "inquiry"), inquiry.Attributes{
				UserID:    uuidtestkit.NewTestFromSalt(t, "user"),
				CreatedAt: updatedAt.Add(-time.Hour),
				UpdatedAt: updatedAt,
			})
			require.NoError(t, err)

			payload, err := event.BuildThreadUpdated(i, 7)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(payload, &got))
			assert.Equal(t, i.ID().String(), got["inquiryId"])
			assert.Equal(t, i.UserID().String(), got["userId"])
			assert.InDelta(t, float64(7), got["sequence"], 0)
			assert.Equal(t, "2026-09-01T10:00:00Z", got["updatedAt"])
			assert.NotContains(t, got, "body")
		})
	})
}
