package user

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &repository{
		tracer: tf.Infra(),
		db:     testDB,
	}
	actual := New(testDB, tf)
	assert.Equal(t, expected, actual)
}

func Test_repository_FindByActive(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	db := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{
		tracer: lt,
		db:     testDB,
	}

	firstUserID, err := uuid.Parse("eaabee3e-3b7a-4f61-8fa9-030944625e92")
	require.NoError(t, err)

	lastUserID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("activeがnilの場合", func(t *testing.T) {
			t.Parallel()

			t.Run("limitとoffsetを指定した場合、作成順で複数件が取得できる", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					limit := int32(100)
					offset := int32(0)

					actual, err := repo.FindByActive(ctx, nil, limit, offset)
					require.NoError(t, err)

					actualFirst := actual[0]
					actualLast := actual[len(actual)-1]

					assert.Equal(t, firstUserID, actualFirst.ID())
					assert.Equal(t, lastUserID, actualLast.ID())
				})
			})

			t.Run("limit=1でoffset=0の場合先頭のユーザーが取得できる", func(t *testing.T) {
				t.Parallel()

				expectedLength := 1

				txm.WithinTx(func(ctx context.Context) {
					limit := int32(1)
					offset := int32(0)

					actual, err := repo.FindByActive(ctx, nil, limit, offset)
					require.NoError(t, err)
					assert.Len(t, actual, expectedLength)

					assert.Equal(t, firstUserID, actual[0].ID())
				})
			})

			t.Run("limit=1でoffset=9の場合、末尾のユーザーが取得できる", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					limit := int32(1)
					offset := int32(9)

					all, getAllUsersErr := repo.FindByActive(ctx, nil, limit, offset)
					require.NoError(t, getAllUsersErr)

					actual := all[len(all)-1]

					assert.Equal(t, lastUserID, actual.ID())
				})
			})

			t.Run("limit=0の場合、空配列になる", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					limit := int32(0)
					offset := int32(0)
					actual, err := repo.FindByActive(ctx, nil, limit, offset)
					require.NoError(t, err)
					require.Empty(t, actual)
				})
			})
		})

		t.Run("activeがtrueの場合", func(t *testing.T) {
			t.Parallel()

			t.Run("limitとoffsetを指定した場合、作成順で複数件が取得できる", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					limit := int32(100)
					offset := int32(0)

					actual, err := repo.FindByActive(ctx, new(true), limit, offset)
					require.NoError(t, err)

					actualFirst := actual[0]
					actualLast := actual[len(actual)-1]

					assert.Equal(t, firstUserID, actualFirst.ID())
					assert.Equal(t, lastUserID, actualLast.ID())
				})
			})
		})

		t.Run("activeがfalseの場合", func(t *testing.T) {
			t.Parallel()

			t.Run("limitとoffsetを指定した場合、作成順で複数件が取得できる", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					limit := int32(100)
					offset := int32(0)

					firstUserID, err := uuid.Parse("d711970c-8e86-4875-8a34-e90bd79096a5")
					require.NoError(t, err)

					lastUserID, err := uuid.Parse("e99b0380-522c-4636-a2b6-452acdd7c4ff")
					require.NoError(t, err)

					actual, err := repo.FindByActive(ctx, new(false), limit, offset)
					require.NoError(t, err)

					actualFirst := actual[0]
					actualLast := actual[len(actual)-1]

					assert.Equal(t, firstUserID, actualFirst.ID())
					assert.Equal(t, lastUserID, actualLast.ID())
				})
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("limitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 負数 LIMIT は PostgreSQL の 2201W（map 未定義）となり ErrInternal へ写像される。
				actual, err := repo.FindByActive(ctx, nil, -1, 0)
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("offsetが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 負数 OFFSET は PostgreSQL の 2201X（map 未定義）となり ErrInternal へ写像される。
				actual, err := repo.FindByActive(ctx, nil, 10, -1)
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("activeがtrueでlimitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByActive(ctx, new(true), -1, 0)
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("activeがfalseでlimitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByActive(ctx, new(false), -1, 0)
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("無効なユーザーが挿入されていてもDomain化の時にエラーになる", func(t *testing.T) {
			t.Parallel()

			t.Run("activeがnilの場合", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					drv := driver.New(ctx, db)
					_, execErr := drv.Exec(ctx,
						"INSERT INTO users "+
							"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code) "+
							"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
						"07e5b6d3-0000-4000-8000-000000000000",
						"Tx",
						"",
						"tx-insert@example.com",
						"000-000-0000",
						"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
						"City",
						"Street",
						"000-0000",
					)
					require.NoError(t, execErr)

					res, actualErr := repo.FindByActive(ctx, nil, 100, 0)
					require.Nil(t, res)
					require.ErrorIs(t, actualErr, apperror.ErrInternal)
				})
			})

			t.Run("activeがtrueの場合", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					drv := driver.New(ctx, db)
					_, execErr := drv.Exec(ctx,
						"INSERT INTO users "+
							"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code) "+
							"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
						"07e5b6d3-0000-4000-8000-000000000000",
						"Tx",
						"",
						"tx-insert@example.com",
						"000-000-0000",
						"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
						"City",
						"Street",
						"000-0000",
					)
					require.NoError(t, execErr)

					res, actualErr := repo.FindByActive(ctx, new(true), 100, 0)
					require.Nil(t, res)
					require.ErrorIs(t, actualErr, apperror.ErrInternal)
				})
			})

			t.Run("activeがfalseの場合", func(t *testing.T) {
				t.Parallel()

				txm.WithinTx(func(ctx context.Context) {
					drv := driver.New(ctx, db)
					_, execErr := drv.Exec(ctx,
						"INSERT INTO users "+
							"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, deleted_at) "+
							"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
						"07e5b6d3-0000-4000-8000-000000000000",
						"Tx",
						"",
						"tx-insert@example.com",
						"000-000-0000",
						"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
						"City",
						"Street",
						"000-0000",
						"2024-01-01T00:00:00Z",
					)
					require.NoError(t, execErr)

					res, actualErr := repo.FindByActive(ctx, new(false), 100, 0)
					require.Nil(t, res)
					require.ErrorIs(t, actualErr, apperror.ErrInternal)
				})
			})
		})
	})
}

func Test_repository_Create(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	db := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{
		tracer: lt,
		db:     testDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なユーザーエンティティの場合、ユーザーが作成できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				now := time.Now()

				userID, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				require.NoError(t, err)
				prefectureID, err := uuid.Parse("a03aaec4-3bd6-4bfb-8e47-2fbfa026d344")
				require.NoError(t, err)

				userEntity, err := user.New(
					userID,
					"Alice",
					"Smith",
					"alice.smith@example.com",
					"555-555-5555",
					prefectureID,
					"新宿区",
					"5-5-5",
					new("Building X"),
					"160-0022",
					now,
					now,
					nil,
				)
				require.NoError(t, err)

				createErr := repo.Create(ctx, userEntity)
				require.NoError(t, createErr)

				row, err := gen.New(driver.New(ctx, db)).GetUserByID(ctx, userEntity.ID())
				require.NoError(t, err)
				// 各列の往復を確認し、INSERT のパラメータバインドずれ（例: 列削除に伴う $n オフセット誤り）を検知する。
				got := row.Users
				assert.Equal(t, userEntity.ID(), got.ID)
				assert.Equal(t, userEntity.FirstName(), got.FirstName)
				assert.Equal(t, userEntity.LastName(), got.LastName)
				assert.Equal(t, userEntity.Email(), got.Email)
				assert.Equal(t, userEntity.Phone(), got.Phone)
				assert.Equal(t, userEntity.PrefectureID(), got.PrefectureID)
				assert.Equal(t, userEntity.City(), got.City)
				assert.Equal(t, userEntity.Street(), got.Street)
				assert.Equal(t, userEntity.Building(), got.Building)
				assert.Equal(t, userEntity.PostalCode(), got.PostalCode)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複するメールアドレスの場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				now := time.Now()
				userID, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				require.NoError(t, err)
				prefectureID, err := uuid.Parse("a03aaec4-3bd6-4bfb-8e47-2fbfa026d344")
				require.NoError(t, err)

				userEntity, err := user.New(
					userID,
					"John",
					"Doe",
					"john.doe@example.com",
					"555-555-5555",
					prefectureID,
					"新宿区",
					"5-5-5",
					new("Building X"),
					"160-0022",
					now,
					now,
					nil,
				)
				require.NoError(t, err)

				createErr := repo.Create(ctx, userEntity)
				require.ErrorIs(t, createErr, apperror.ErrConflict)
			})
		})
	})
}

func Test_repository_CountByActive(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{
		tracer: lt,
		db:     testDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("active=trueの場合、アクティブなユーザーの件数が取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.CountByActive(ctx, new(true))
				require.NoError(t, err)
				assert.Equal(t, int64(8), got)
			})
		})
		t.Run("active=falseの場合、非アクティブなユーザーの件数が取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.CountByActive(ctx, new(false))
				require.NoError(t, err)
				assert.Equal(t, int64(2), got)
			})
		})
		t.Run("active=nilの場合、全ユーザーの件数が取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.CountByActive(ctx, nil)
				require.NoError(t, err)
				assert.Equal(t, int64(10), got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			got, err := repo.CountByActive(ctx, new(true))
			assert.Zero(t, got)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

// newTestQueries は、テスト DB へ直接クエリを発行する sqlc の Queries ファクトリを返す。
func newTestQueries(t *testing.T) func(context.Context) *gen.Queries {
	t.Helper()
	testDB := testkit.NewTestDB(t)
	return func(ctx context.Context) *gen.Queries { return gen.New(driver.New(ctx, testDB)) }
}

func Test_fetchListUsersRows(t *testing.T) {
	t.Parallel()

	txm := testkit.NewTestTransactionRunner(t)
	queries := newTestQueries(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("limit 件までのユーザーをエンティティへ変換して返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := fetchListUsersRows(ctx, queries(ctx), &gen.ListUsersParams{LimitParam: 2, OffsetParam: 0})

				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.NotEmpty(t, got[0].ID())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			got, err := fetchListUsersRows(ctx, queries(ctx), &gen.ListUsersParams{LimitParam: 1})

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func Test_fetchListUsersRowsByActive(t *testing.T) {
	t.Parallel()

	txm := testkit.NewTestTransactionRunner(t)
	queries := newTestQueries(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未削除のユーザーだけをエンティティへ変換して返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := fetchListUsersRowsByActive(
					ctx, queries(ctx), &gen.ListActiveUsersParams{LimitParam: 100, OffsetParam: 0})

				require.NoError(t, err)
				require.NotEmpty(t, got)
				for _, u := range got {
					assert.Nil(t, u.DeletedAt())
				}
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			got, err := fetchListUsersRowsByActive(ctx, queries(ctx), &gen.ListActiveUsersParams{LimitParam: 1})

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func Test_fetchListUsersRowsByDeleted(t *testing.T) {
	t.Parallel()

	txm := testkit.NewTestTransactionRunner(t)
	queries := newTestQueries(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("削除済みのユーザーだけをエンティティへ変換して返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := fetchListUsersRowsByDeleted(
					ctx, queries(ctx), &gen.ListDeletedUsersParams{LimitParam: 100, OffsetParam: 0})

				require.NoError(t, err)
				require.NotEmpty(t, got)
				for _, u := range got {
					assert.NotNil(t, u.DeletedAt())
				}
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			got, err := fetchListUsersRowsByDeleted(ctx, queries(ctx), &gen.ListDeletedUsersParams{LimitParam: 1})

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func Test_rowToUser(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築時の検証失敗はErrInternalへ正規化され元の分類は露出しない", func(t *testing.T) {
			t.Parallel()

			// ゼロ値の行は ID が nil のため domain 構築が失敗する。
			// 成功経路は Test_repository_FindByID / Test_repository_FindByActive の実 DB テストでカバー。
			entity, err := rowToUser(gen.Users{})
			require.Error(t, err)
			require.Nil(t, entity)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_rowsToUsers(t *testing.T) {
	t.Parallel()

	identity := func(r gen.Users) gen.Users { return r }

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空スライスは空の列を返す", func(t *testing.T) {
			t.Parallel()

			got, err := rowsToUsers([]gen.Users{}, identity)

			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("変換失敗があれば nil とエラーを返す", func(t *testing.T) {
			t.Parallel()

			// ゼロ値の行は ID が nil で rowToUser が失敗する（成功経路は実 DB テストでカバー）。
			got, err := rowsToUsers([]gen.Users{{}}, identity)

			require.Error(t, err)
			assert.Nil(t, got)
		})
	})
}
