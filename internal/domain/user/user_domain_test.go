package user

import (
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	id := uuidtestkit.NewTestFromSalt(t, "user")
	prefectureID := uuidtestkit.NewTestFromSalt(t, "prefecture")
	firstName := "John"
	lastName := "Doe"
	email := "john.doe@example.com"
	phone := "1234567890"
	city := "Shibuya"
	street := "1-2-3"
	postalCode := "150-0001"
	building := new("Building A")
	createdAt := baseTime
	updatedAt := baseTime.Add(time.Hour)
	deletedAt := new(updatedAt.Add(time.Minute))

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("全ての入力が正しい場合、エンティティが生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, Attributes{
				Profile: Profile{
					FirstName:    firstName,
					LastName:     lastName,
					Email:        email,
					Phone:        phone,
					PrefectureID: prefectureID,
					City:         city,
					Street:       street,
					Building:     building,
					PostalCode:   postalCode,
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: deletedAt,
			})

			require.NoError(t, err)
			assert.Equal(t, id, actual.id)
			assert.Equal(t, firstName, actual.firstName)
			assert.Equal(t, lastName, actual.lastName)
			assert.Equal(t, email, actual.email.Value())
			assert.Equal(t, phone, actual.phone)
			assert.Equal(t, prefectureID, actual.prefectureID)
			assert.Equal(t, city, actual.city)
			assert.Equal(t, street, actual.street)
			assert.Equal(t, *building, *actual.building)
			assert.Equal(t, postalCode, actual.postalCode.Value())
			assert.Equal(t, createdAt, actual.createdAt)
			assert.Equal(t, updatedAt, actual.updatedAt)
			assert.Equal(t, *deletedAt, *actual.deletedAt)
		})

		t.Run("オプションのbuildingがnilの場合、エンティティが生成されBuildingはnilを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, Attributes{
				Profile: Profile{
					FirstName:    firstName,
					LastName:     lastName,
					Email:        email,
					Phone:        phone,
					PrefectureID: prefectureID,
					City:         city,
					Street:       street,
					PostalCode:   postalCode,
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: deletedAt,
			})

			require.NoError(t, err)
			assert.Nil(t, actual.Building())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("IDがゼロ値の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(uuid.UUID{}, Attributes{
				Profile: Profile{
					FirstName:    firstName,
					LastName:     lastName,
					Email:        email,
					Phone:        phone,
					PrefectureID: prefectureID,
					City:         city,
					Street:       street,
					Building:     building,
					PostalCode:   postalCode,
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: deletedAt,
			})

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("firstNameが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("firstNameの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    strings.Repeat("名", minLength-1),
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidFirstName)
			})

			t.Run("firstNameの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    strings.Repeat("名", maxFirstNameLength+1),
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidFirstName)
			})
		})

		t.Run("lastNameが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("lastNameの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     strings.Repeat("姓", minLength-1),
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidLastName)
			})

			t.Run("lastNameの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     strings.Repeat("姓", maxLastNameLength+1),
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidLastName)
			})
		})
		t.Run("emailが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("emailの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        strings.Repeat("a", minLength-1),
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidEmail)
			})

			t.Run("emailの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        strings.Repeat("a", maxEmailLength+1),
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidEmail)
			})

			t.Run("emailの形式が不正な場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        "not-an-email",
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidEmail)
			})
		})

		t.Run("phoneが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("phoneの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        strings.Repeat("1", minLength-1),
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPhone)
			})

			t.Run("phoneの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        strings.Repeat("1", maxPhoneLength+1),
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPhone)
			})
		})

		t.Run("cityが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("cityの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         strings.Repeat("市", minLength-1),
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidCity)
			})

			t.Run("cityの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         strings.Repeat("市", maxCityLength+1),
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidCity)
			})
		})

		t.Run("streetが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("streetの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       strings.Repeat("番", minLength-1),
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidStreet)
			})

			t.Run("streetの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       strings.Repeat("番", maxStreetLength+1),
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidStreet)
			})
		})

		t.Run("buildingが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("buildingの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     new(strings.Repeat("建", minLength-1)),
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidBuilding)
			})

			t.Run("buildingの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     new(strings.Repeat("建", maxBuildingLength+1)),
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidBuilding)
			})
		})

		t.Run("postalCodeが不正な場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("postalCodeが空文字の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   "",
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPostalCode)
			})

			t.Run("postalCodeがハイフン無しの場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   "1500001",
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: deletedAt,
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPostalCode)
			})
		})

		t.Run("prefectureIDがゼロ値の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, Attributes{
				Profile: Profile{
					FirstName:    firstName,
					LastName:     lastName,
					Email:        email,
					Phone:        phone,
					PrefectureID: uuid.UUID{},
					City:         city,
					Street:       street,
					Building:     building,
					PostalCode:   postalCode,
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: deletedAt,
			})

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPrefectureID)
		})

		t.Run("updatedAtがcreatedAtより前の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, Attributes{
				Profile: Profile{
					FirstName:    firstName,
					LastName:     lastName,
					Email:        email,
					Phone:        phone,
					PrefectureID: prefectureID,
					City:         city,
					Street:       street,
					Building:     building,
					PostalCode:   postalCode,
				},
				CreatedAt: createdAt,
				UpdatedAt: createdAt.Add(-time.Minute),
				DeletedAt: deletedAt,
			})

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUpdatedAt)
		})

		t.Run("deletedAtが不正な場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("deletedAtがcreatedAtより前の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: new(createdAt.Add(-time.Minute)),
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidDeletedAt)
			})

			t.Run("deletedAtがupdatedAtより前の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(id, Attributes{
					Profile: Profile{
						FirstName:    firstName,
						LastName:     lastName,
						Email:        email,
						Phone:        phone,
						PrefectureID: prefectureID,
						City:         city,
						Street:       street,
						Building:     building,
						PostalCode:   postalCode,
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
					DeletedAt: new(updatedAt.Add(-time.Minute)),
				})

				assert.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidDeletedAt)
			})
		})
	})
}

// newAccessorUser は、全フィールドが非ゼロ（building / deletedAt も非nil）の有効なユーザーを生成する
// ゲッター検証用のテストヘルパー。
func newAccessorUser(t *testing.T) *User {
	t.Helper()
	baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	u, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
		Profile: Profile{
			FirstName:    "John",
			LastName:     "Doe",
			Email:        "john.doe@example.com",
			Phone:        "1234567890",
			PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
			City:         "Shibuya",
			Street:       "1-2-3",
			Building:     new("Building A"),
			PostalCode:   "150-0001",
		},
		CreatedAt: baseTime,
		UpdatedAt: baseTime.Add(time.Hour),
		DeletedAt: new(baseTime.Add(time.Hour).Add(time.Minute)),
	})
	require.NoError(t, err)
	return u
}

func TestUser_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存したIDを返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.id, u.ID())
		})
	})
}

func TestUser_FirstName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した名前を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.firstName, u.FirstName())
		})
	})
}

func TestUser_LastName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した名字を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.lastName, u.LastName())
		})
	})
}

func TestUser_Email(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存したメールアドレスを返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.email.Value(), u.Email())
		})
	})
}

func TestUser_Phone(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した電話番号を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.phone, u.Phone())
		})
	})
}

func TestUser_PrefectureID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した都道府県IDを返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.prefectureID, u.PrefectureID())
		})
	})
}

func TestUser_City(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した市区町村名を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.city, u.City())
		})
	})
}

func TestUser_Street(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した番地を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.street, u.Street())
		})
	})
}

func TestUser_Building(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した建物名を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.building, u.Building())
		})

		t.Run("buildingがnilの場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			u, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
				Profile: Profile{
					FirstName:    "John",
					LastName:     "Doe",
					Email:        "john.doe@example.com",
					Phone:        "1234567890",
					PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
					City:         "Shibuya",
					Street:       "1-2-3",
					PostalCode:   "150-0001",
				},
				CreatedAt: baseTime,
				UpdatedAt: baseTime.Add(time.Hour),
			})
			require.NoError(t, err)

			assert.Nil(t, u.Building())
		})

		// 共有ポインタ building を直接 mutate して不変性を検証するため、
		// 同じポインタを読むサブテスト同士を並列実行すると -race で競合する。意図的に直列化する。
		t.Run("返り値を変更しても内部状態は不変", func(t *testing.T) {
			baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			building := new("Building A")
			user, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
				Profile: Profile{
					FirstName:    "John",
					LastName:     "Doe",
					Email:        "john.doe@example.com",
					Phone:        "1234567890",
					PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
					City:         "Shibuya",
					Street:       "1-2-3",
					Building:     building,
					PostalCode:   "150-0001",
				},
				CreatedAt: baseTime,
				UpdatedAt: baseTime.Add(time.Hour),
				DeletedAt: new(baseTime.Add(time.Hour).Add(time.Minute)),
			})
			require.NoError(t, err)

			t.Run("buildingのポインタを変更しても、ユーザーのbuildingが変更されていないことを確認する", func(t *testing.T) {
				t.Parallel()

				original := *building

				// buildingの値を変更
				*building = "Building B"

				assert.NotEqual(t, *building, *user.Building())
				assert.Equal(t, original, *user.Building())
			})

			t.Run("Buildingメソッドの返り値のポインタを変更しても、ユーザーのbuildingが変更されていないことを確認する", func(t *testing.T) {
				t.Parallel()

				original := *user.Building()

				// Buildingメソッドの返り値のポインタを変更
				actualBuilding := user.Building()
				*actualBuilding = "Building B"

				assert.NotEqual(t, *actualBuilding, *user.Building())
				assert.Equal(t, original, *user.Building())
			})
		})
	})
}

func TestUser_PostalCode(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した郵便番号を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.postalCode.Value(), u.PostalCode())
		})
	})
}

func TestUser_DeletedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した削除日時を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.deletedAt, u.DeletedAt())
		})

		t.Run("deletedAtがnilの場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			u, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
				Profile: Profile{
					FirstName:    "John",
					LastName:     "Doe",
					Email:        "john.doe@example.com",
					Phone:        "1234567890",
					PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
					City:         "Shibuya",
					Street:       "1-2-3",
					PostalCode:   "150-0001",
				},
				CreatedAt: baseTime,
				UpdatedAt: baseTime.Add(time.Hour),
			})
			require.NoError(t, err)

			assert.Nil(t, u.DeletedAt())
		})

		// 共有ポインタ deletedAt を直接 mutate して不変性を検証するため、
		// 同じポインタを読むサブテスト同士を並列実行すると -race で競合する。意図的に直列化する。
		t.Run("返り値を変更しても内部状態は不変", func(t *testing.T) {
			baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			deletedAt := new(baseTime.Add(time.Hour).Add(time.Minute))
			user, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
				Profile: Profile{
					FirstName:    "John",
					LastName:     "Doe",
					Email:        "john.doe@example.com",
					Phone:        "1234567890",
					PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
					City:         "Shibuya",
					Street:       "1-2-3",
					Building:     new("Building A"),
					PostalCode:   "150-0001",
				},
				CreatedAt: baseTime,
				UpdatedAt: baseTime.Add(time.Hour),
				DeletedAt: deletedAt,
			})
			require.NoError(t, err)

			t.Run("deletedAtのポインタを変更しても、ユーザーのdeletedAtが変更されていないことを確認する", func(t *testing.T) {
				t.Parallel()

				original := *deletedAt

				// deletedAtの値を変更
				*deletedAt = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

				assert.NotEqual(t, *deletedAt, *user.DeletedAt())
				assert.Equal(t, original, *user.DeletedAt())
			})

			t.Run("DeletedAtメソッドの返り値のポインタを変更しても、ユーザーのdeletedAtが変更されていないことを確認する", func(t *testing.T) {
				t.Parallel()

				original := *user.DeletedAt()

				// DeletedAtメソッドの返り値のポインタを変更
				actualDeletedAt := user.DeletedAt()
				*actualDeletedAt = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

				assert.NotEqual(t, *actualDeletedAt, *user.DeletedAt())
				assert.Equal(t, original, *user.DeletedAt())
			})
		})
	})
}

func TestUser_CreatedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した作成日時を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.createdAt, u.CreatedAt())
		})
	})
}

func TestUser_UpdatedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存した更新日時を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.updatedAt, u.UpdatedAt())
		})
	})
}

func TestUser_FullName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("名前と名字を空白で連結した値を返す", func(t *testing.T) {
			t.Parallel()
			u := newAccessorUser(t)

			assert.Equal(t, u.firstName+" "+u.lastName, u.FullName())
		})
	})
}

// newValidUser は、削除されていない有効なユーザーと基準時刻を返すテストヘルパー。
func newValidUser(t *testing.T) (*User, time.Time) {
	t.Helper()
	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	u, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
		Profile: Profile{
			FirstName:    "John",
			LastName:     "Doe",
			Email:        "john@example.com",
			Phone:        "1234567890",
			PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
			City:         "Shibuya",
			Street:       "1-2-3",
			Building:     new("Building A"),
			PostalCode:   "150-0001",
		},
		CreatedAt: base,
		UpdatedAt: base,
	})
	require.NoError(t, err)
	return u, base
}

// newUserWithUpdatedAt は、createdAt=base・updatedAt=base+offset のユーザーを生成します（単調性検証用）。
func newUserWithUpdatedAt(t *testing.T, offset time.Duration) (*User, time.Time) {
	t.Helper()
	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	u, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
		Profile: Profile{
			FirstName:    "John",
			LastName:     "Doe",
			Email:        "john@example.com",
			Phone:        "1234567890",
			PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
			City:         "Shibuya",
			Street:       "1-2-3",
			Building:     new("Building A"),
			PostalCode:   "150-0001",
		},
		CreatedAt: base,
		UpdatedAt: base.Add(offset),
	})
	require.NoError(t, err)
	return u, base
}

func TestUser_UpdateProfile(t *testing.T) {
	t.Parallel()

	newPrefID := uuidtestkit.NewTestFromSalt(t, "prefecture2")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全フィールドと更新日時が置き換わる", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)
			newUpdatedAt := base.Add(time.Hour)

			err := u.UpdateProfile(Profile{
				FirstName:    "Jane",
				LastName:     "Smith",
				Email:        "jane@example.com",
				Phone:        "0987654321",
				PrefectureID: newPrefID,
				City:         "Minato",
				Street:       "4-5-6",
				Building:     new("Tower"),
				PostalCode:   "200-0002",
			}, newUpdatedAt)
			require.NoError(t, err)

			assert.Equal(t, "Jane", u.firstName)
			assert.Equal(t, "Smith", u.lastName)
			assert.Equal(t, "jane@example.com", u.email.Value())
			assert.Equal(t, "0987654321", u.phone)
			assert.Equal(t, newPrefID, u.prefectureID)
			assert.Equal(t, "Minato", u.city)
			assert.Equal(t, "4-5-6", u.street)
			require.NotNil(t, u.building)
			assert.Equal(t, "Tower", *u.building)
			assert.Equal(t, "200-0002", u.postalCode.Value())
			assert.Equal(t, newUpdatedAt, u.updatedAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一フィールドが不正な場合、そのフィールド識別子のみが付与される", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			err := u.UpdateProfile(Profile{
				FirstName:    "",
				LastName:     "Smith",
				Email:        "jane@example.com",
				Phone:        "0987654321",
				PrefectureID: newPrefID,
				City:         "Minato",
				Street:       "4-5-6",
				PostalCode:   "200-0002",
			}, base.Add(time.Hour))
			require.ErrorIs(t, err, ErrInvalidFirstName)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{FieldFirstName}, meta.Details())
		})

		t.Run("prefectureIDがゼロ値の場合、ErrInvalidPrefectureIDが返される", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			err := u.UpdateProfile(Profile{
				FirstName:    "Jane",
				LastName:     "Smith",
				Email:        "jane@example.com",
				Phone:        "0987654321",
				PrefectureID: uuid.UUID{},
				City:         "Minato",
				Street:       "4-5-6",
				PostalCode:   "200-0002",
			}, base.Add(time.Hour))
			require.ErrorIs(t, err, ErrInvalidPrefectureID)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{FieldPrefectureID}, meta.Details())
		})

		t.Run("複数フィールドが同時に不正な場合、全フィールドのエラーと識別子が収集される", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			err := u.UpdateProfile(Profile{
				FirstName:    "",
				LastName:     "Smith",
				Email:        strings.Repeat("e", maxEmailLength+1),
				Phone:        "0987654321",
				PrefectureID: newPrefID,
				City:         "Minato",
				Street:       "4-5-6",
				PostalCode:   "200-0002",
			}, base.Add(time.Hour))
			require.ErrorIs(t, err, ErrInvalidFirstName)
			require.ErrorIs(t, err, ErrInvalidEmail)
			require.ErrorIs(t, err, apperror.ErrValidation)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{FieldFirstName, FieldEmail}, meta.Details())
		})

		t.Run("updatedAtがcreatedAtより前の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			err := u.UpdateProfile(Profile{
				FirstName:    "Jane",
				LastName:     "Smith",
				Email:        "jane@example.com",
				Phone:        "0987654321",
				PrefectureID: newPrefID,
				City:         "Minato",
				Street:       "4-5-6",
				PostalCode:   "200-0002",
			}, base.Add(-time.Hour))
			require.ErrorIs(t, err, ErrInvalidUpdatedAt)
		})

		t.Run("updatedAtが現在のupdatedAtより前の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			u, base := newUserWithUpdatedAt(t, 2*time.Hour)

			// createdAt 以降だが現在の updatedAt(base+2h) より前の時刻は単調性違反で拒否される
			err := u.UpdateProfile(Profile{
				FirstName:    "Jane",
				LastName:     "Smith",
				Email:        "jane@example.com",
				Phone:        "0987654321",
				PrefectureID: newPrefID,
				City:         "Minato",
				Street:       "4-5-6",
				PostalCode:   "200-0002",
			}, base.Add(time.Hour))
			require.ErrorIs(t, err, ErrInvalidUpdatedAt)
		})

		t.Run("論理削除済みユーザーは更新できない", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)
			require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

			err := u.UpdateProfile(Profile{
				FirstName:    "Jane",
				LastName:     "Smith",
				Email:        "jane@example.com",
				Phone:        "0987654321",
				PrefectureID: newPrefID,
				City:         "Minato",
				Street:       "4-5-6",
				PostalCode:   "200-0002",
			}, base.Add(2*time.Hour))
			require.ErrorIs(t, err, ErrAlreadyDeleted)
		})
	})
}

func Test_User_ensureNotDeleted(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("削除されていない場合、nil を返す", func(t *testing.T) {
			t.Parallel()
			u, _ := newValidUser(t)

			require.NoError(t, u.ensureNotDeleted())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("論理削除済みの場合、ErrAlreadyDeleted を返す", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)
			require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

			require.ErrorIs(t, u.ensureNotDeleted(), ErrAlreadyDeleted)
		})
	})
}

func Test_User_ensureUpdatedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在の updatedAt 以降の場合、nil を返す", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			require.NoError(t, u.ensureUpdatedAt(base.Add(time.Hour)))
		})

		t.Run("現在の updatedAt と等しい場合、nil を返す", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			require.NoError(t, u.ensureUpdatedAt(base))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("createdAt より前の場合、ErrInvalidUpdatedAt を返す", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			require.ErrorIs(t, u.ensureUpdatedAt(base.Add(-time.Hour)), ErrInvalidUpdatedAt)
		})

		t.Run("現在の updatedAt より前の場合、ErrInvalidUpdatedAt を返す", func(t *testing.T) {
			t.Parallel()
			u, base := newUserWithUpdatedAt(t, 2*time.Hour)

			// createdAt 以降だが現在の updatedAt(base+2h) より前の時刻は単調性違反となる
			require.ErrorIs(t, u.ensureUpdatedAt(base.Add(time.Hour)), ErrInvalidUpdatedAt)
		})
	})
}

func Test_validateDeletedAt(t *testing.T) {
	t.Parallel()

	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := base.Add(time.Hour)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("createdAt / updatedAt 以降の場合、nil を返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateDeletedAt(updatedAt.Add(time.Minute), base, updatedAt))
		})

		t.Run("updatedAt と等しい場合、nil を返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateDeletedAt(updatedAt, base, updatedAt))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("createdAt より前の場合、ErrInvalidDeletedAt を返す", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, validateDeletedAt(base.Add(-time.Minute), base, updatedAt), ErrInvalidDeletedAt)
		})

		t.Run("updatedAt より前の場合、ErrInvalidDeletedAt を返す", func(t *testing.T) {
			t.Parallel()

			// createdAt 以降だが updatedAt より前の削除時刻は拒否される
			require.ErrorIs(t, validateDeletedAt(base.Add(time.Minute), base, updatedAt), ErrInvalidDeletedAt)
		})
	})
}

func Test_validateProfileFields(t *testing.T) {
	t.Parallel()

	prefectureID := uuidtestkit.NewTestFromSalt(t, "prefecture")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全フィールドが正しい場合、VO を返しエラーは nil", func(t *testing.T) {
			t.Parallel()

			emailVO, postalCodeVO, err := validateProfileFields(Profile{
				FirstName:    "John",
				LastName:     "Doe",
				Email:        "john@example.com",
				Phone:        "1234567890",
				PrefectureID: prefectureID,
				City:         "Shibuya",
				Street:       "1-2-3",
				Building:     new("Building A"),
				PostalCode:   "150-0001",
			})
			require.NoError(t, err)
			assert.Equal(t, "john@example.com", emailVO.Value())
			assert.Equal(t, "150-0001", postalCodeVO.Value())
		})

		t.Run("building が nil の場合でも、VO を返しエラーは nil", func(t *testing.T) {
			t.Parallel()

			_, _, err := validateProfileFields(Profile{
				FirstName:    "John",
				LastName:     "Doe",
				Email:        "john@example.com",
				Phone:        "1234567890",
				PrefectureID: prefectureID,
				City:         "Shibuya",
				Street:       "1-2-3",
				PostalCode:   "150-0001",
			})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一フィールドが不正な場合、そのフィールド識別子のみが付与される", func(t *testing.T) {
			t.Parallel()

			_, _, err := validateProfileFields(Profile{
				FirstName:    "",
				LastName:     "Doe",
				Email:        "john@example.com",
				Phone:        "1234567890",
				PrefectureID: prefectureID,
				City:         "Shibuya",
				Street:       "1-2-3",
				PostalCode:   "150-0001",
			})
			require.ErrorIs(t, err, ErrInvalidFirstName)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{FieldFirstName}, meta.Details())
		})

		t.Run("building が範囲外の場合、ErrInvalidBuilding を返す", func(t *testing.T) {
			t.Parallel()

			_, _, err := validateProfileFields(Profile{
				FirstName:    "John",
				LastName:     "Doe",
				Email:        "john@example.com",
				Phone:        "1234567890",
				PrefectureID: prefectureID,
				City:         "Shibuya",
				Street:       "1-2-3",
				Building:     new(strings.Repeat("建", maxBuildingLength+1)),
				PostalCode:   "150-0001",
			})
			require.ErrorIs(t, err, ErrInvalidBuilding)
		})

		t.Run("複数フィールドが同時に不正な場合、全フィールドのエラーと識別子が収集される", func(t *testing.T) {
			t.Parallel()

			_, _, err := validateProfileFields(Profile{
				FirstName:    "",
				LastName:     "Doe",
				Email:        strings.Repeat("e", maxEmailLength+1),
				Phone:        "1234567890",
				PrefectureID: prefectureID,
				City:         "Shibuya",
				Street:       "1-2-3",
				PostalCode:   "150-0001",
			})
			require.ErrorIs(t, err, ErrInvalidFirstName)
			require.ErrorIs(t, err, ErrInvalidEmail)
			require.ErrorIs(t, err, apperror.ErrValidation)
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{FieldFirstName, FieldEmail}, meta.Details())
		})
	})
}

func TestUser_MarkAsDeleted(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deletedAtが設定される", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)
			deletedAt := base.Add(time.Hour)

			err := u.MarkAsDeleted(deletedAt)
			require.NoError(t, err)
			require.NotNil(t, u.deletedAt)
			assert.Equal(t, deletedAt, *u.deletedAt)
			// 論理削除時に updatedAt も削除時刻へ追従する
			assert.Equal(t, deletedAt, u.updatedAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既に削除済みの場合、ErrAlreadyDeletedを返す", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)
			require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

			err := u.MarkAsDeleted(base.Add(2 * time.Hour))
			require.ErrorIs(t, err, ErrAlreadyDeleted)
		})

		t.Run("deletedAtがcreatedAtより前の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			u, base := newValidUser(t)

			err := u.MarkAsDeleted(base.Add(-time.Hour))
			require.ErrorIs(t, err, ErrInvalidDeletedAt)
		})

		t.Run("deletedAtがupdatedAtより前の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
			u, err := New(uuidtestkit.NewTestFromSalt(t, "user"), Attributes{
				Profile: Profile{
					FirstName:    "John",
					LastName:     "Doe",
					Email:        "john@example.com",
					Phone:        "1234567890",
					PrefectureID: uuidtestkit.NewTestFromSalt(t, "prefecture"),
					City:         "Shibuya",
					Street:       "1-2-3",
					Building:     new("Building A"),
					PostalCode:   "150-0001",
				},
				CreatedAt: base,
				UpdatedAt: base.Add(2 * time.Hour),
			})
			require.NoError(t, err)

			err = u.MarkAsDeleted(base.Add(time.Hour))
			require.ErrorIs(t, err, ErrInvalidDeletedAt)
		})
	})
}
