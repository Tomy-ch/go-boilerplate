package event_test

import (
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainuser "go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/usecase/user/event"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithdrawn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("BuildWithdrawn が生成した payload をそのまま復元する", func(t *testing.T) {
			t.Parallel()
			// producing 側と consuming 側が同じ payload 形を見ていることを、両者を突き合わせて固定する。
			createdAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)
			entity, err := domainuser.New(uuid.NewTestFromSalt(t, "pw_id"), domainuser.Attributes{
				Profile: domainuser.Profile{
					FirstName:    "first_name",
					LastName:     "last_name",
					Email:        "pw@example.com",
					Phone:        "090-0000-0000",
					PrefectureID: uuid.NewTestFromSalt(t, "pw_prefecture"),
					City:         "city_name",
					Street:       "town_address",
					PostalCode:   "150-0001",
				},
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			})
			require.NoError(t, err)
			deletedAt := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
			require.NoError(t, entity.MarkAsDeleted(deletedAt))
			payload, err := event.BuildWithdrawn(entity)
			require.NoError(t, err)

			got, err := event.ParseWithdrawn(payload)

			require.NoError(t, err)
			assert.Equal(t, entity.ID().String(), got.UserID)
			assert.Equal(t, deletedAt.Format(time.RFC3339Nano), got.DeletedAt)
		})

		t.Run("未知のフィールドは無視する", func(t *testing.T) {
			t.Parallel()
			// version を上げずに producing 側がフィールドを足しても消費側が止まらないことを固定する。
			got, err := event.ParseWithdrawn([]byte(`{"userId":"u1","deletedAt":"","unknown":1}`))

			require.NoError(t, err)
			assert.Equal(t, "u1", got.UserID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON として読めない payload を弾く", func(t *testing.T) {
			t.Parallel()

			_, err := event.ParseWithdrawn([]byte("not-json"))

			require.ErrorIs(t, err, event.ErrInvalidWithdrawn)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("型が合わない payload を弾く", func(t *testing.T) {
			t.Parallel()

			_, err := event.ParseWithdrawn([]byte(`{"userId":42}`))

			require.ErrorIs(t, err, event.ErrInvalidWithdrawn)
		})
	})
}

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
