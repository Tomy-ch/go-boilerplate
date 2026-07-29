package event_test

import (
	"encoding/json"
	"testing"
	"time"

	domainuser "go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/usecase/user/event"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWithdrawn(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)

	// active は、未削除のユーザーを salt から生成するローカルヘルパーです。
	active := func(t *testing.T, salt string) *domainuser.User {
		t.Helper()
		entity, err := domainuser.New(uuid.NewTestFromSalt(t, salt+"_id"), domainuser.Attributes{
			Profile: domainuser.Profile{
				FirstName:    "first_name",
				LastName:     "last_name",
				Email:        salt + "@example.com",
				Phone:        "090-0000-0000",
				PrefectureID: uuid.NewTestFromSalt(t, salt+"_prefecture"),
				City:         "city_name",
				Street:       "town_address",
				PostalCode:   "150-0001",
			},
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		})
		require.NoError(t, err)
		return entity
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("退会済みユーザーの自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			entity := active(t, "bw")
			deletedAt := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
			require.NoError(t, entity.MarkAsDeleted(deletedAt))

			payload, err := event.BuildWithdrawn(entity)
			require.NoError(t, err)

			var decoded struct {
				UserID    string `json:"userId"`
				DeletedAt string `json:"deletedAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Equal(t, entity.ID().String(), decoded.UserID)
			assert.Equal(t, deletedAt.Format(time.RFC3339Nano), decoded.DeletedAt)
		})

		t.Run("deletedAtがnilのユーザーはdeletedAtが空文字列になる", func(t *testing.T) {
			t.Parallel()

			payload, err := event.BuildWithdrawn(active(t, "bw_nil"))
			require.NoError(t, err)

			var decoded struct {
				DeletedAt string `json:"deletedAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Empty(t, decoded.DeletedAt)
		})
	})
}
