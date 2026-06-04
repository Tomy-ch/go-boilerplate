package user

import (
	"strings"
	"testing"
	"time"

	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	id := uuid.NewTestFromSalt(t, "user")
	prefectureID := uuid.NewTestFromSalt(t, "prefecture")
	firstName := "John"
	lastName := "Doe"
	passwordHash := "hashed_password"
	email := "john.doe@example.com"
	phone := "1234567890"
	city := "Shibuya"
	street := "1-2-3"
	postalCode := "150-0001"
	building := ptr.To("Building A")
	createdAt := baseTime
	updatedAt := baseTime.Add(time.Hour)
	deletedAt := ptr.To(updatedAt.Add(time.Minute))

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("全ての入力が正しい場合、エンティティが生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(
				id,
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				prefectureID,
				city,
				street,
				building,
				postalCode,
				createdAt,
				updatedAt,
				deletedAt,
			)

			require.NoError(t, err)
			assert.Equal(t, id, actual.id)
			assert.Equal(t, firstName, actual.firstName)
			assert.Equal(t, lastName, actual.lastName)
			assert.Equal(t, passwordHash, actual.passwordHash)
			assert.Equal(t, email, actual.email)
			assert.Equal(t, phone, actual.phone)
			assert.Equal(t, prefectureID, actual.prefectureID)
			assert.Equal(t, city, actual.city)
			assert.Equal(t, street, actual.street)
			assert.Equal(t, *building, *actual.building)
			assert.Equal(t, postalCode, actual.postalCode)
			assert.Equal(t, createdAt, actual.createdAt)
			assert.Equal(t, updatedAt, actual.updatedAt)
			assert.Equal(t, *deletedAt, *actual.deletedAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("IDがゼロ値の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(
				uuid.UUID{},
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				prefectureID,
				city,
				street,
				building,
				postalCode,
				createdAt,
				updatedAt,
				deletedAt,
			)

			assert.Nil(t, actual)
			assert.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("firstNameが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("firstNameの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				actual, err := New(
					id,
					strings.Repeat("名", minLength-1),
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidFirstName)
			})

			t.Run("firstNameの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				actual, err := New(
					id,
					strings.Repeat("名", maxFirstNameLength+1),
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidFirstName)
			})
		})

		t.Run("lastNameが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("lastNameの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					strings.Repeat("姓", minLength-1),
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidLastName)
			})

			t.Run("lastNameの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					strings.Repeat("姓", maxLastNameLength+1),
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidLastName)
			})
		})
		t.Run("passwordが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("passwordの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					strings.Repeat("a", minLength-1),
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidPasswordHash)
			})

			t.Run("passwordの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					strings.Repeat("a", maxPasswordLength+1),
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidPasswordHash)
			})
		})

		t.Run("emailが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("emailの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					strings.Repeat("a", minLength-1),
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidEmail)
			})

			t.Run("emailの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					strings.Repeat("a", maxEmailLength+1),
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidEmail)
			})
		})

		t.Run("phoneが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("phoneの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					strings.Repeat("1", minLength-1),
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidPhone)
			})

			t.Run("phoneの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					strings.Repeat("1", maxPhoneLength+1),
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidPhone)
			})
		})

		t.Run("cityが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("cityの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					strings.Repeat("市", minLength-1),
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidCity)
			})

			t.Run("cityの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					strings.Repeat("市", maxCityLength+1),
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidCity)
			})
		})

		t.Run("streetが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("streetの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					strings.Repeat("番", minLength-1),
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidStreet)
			})

			t.Run("streetの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					strings.Repeat("番", maxStreetLength+1),
					building,
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidStreet)
			})
		})

		t.Run("buildingが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("buildingの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					ptr.To(strings.Repeat("建", minLength-1)),
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidBuilding)
			})

			t.Run("buildingの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					ptr.To(strings.Repeat("建", maxBuildingLength+1)),
					postalCode,
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidBuilding)
			})
		})

		t.Run("postalCodeが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("postalCodeの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					strings.Repeat("0", minLength-1),
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidPostalCode)
			})

			t.Run("postalCodeの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					strings.Repeat("0", maxPostalCodeLength+1),
					createdAt,
					updatedAt,
					deletedAt,
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidPostalCode)
			})
		})

		t.Run("prefectureIDがゼロ値の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(
				id,
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				uuid.UUID{},
				city,
				street,
				building,
				postalCode,
				createdAt,
				updatedAt,
				deletedAt,
			)

			assert.Nil(t, actual)
			assert.ErrorIs(t, err, ErrInvalidPrefectureID)
		})

		t.Run("updatedAtがcreatedAtより前の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(
				id,
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				prefectureID,
				city,
				street,
				building,
				postalCode,
				createdAt,
				createdAt.Add(-time.Minute),
				deletedAt,
			)

			assert.Nil(t, actual)
			assert.ErrorIs(t, err, ErrInvalidUpdatedAt)
		})

		t.Run("deletedAtが不正な場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("deletedAtがcreatedAtより前の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					ptr.To(createdAt.Add(-time.Minute)),
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidDeletedAt)
			})

			t.Run("deletedAtがupdatedAtより前の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id,
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID,
					city,
					street,
					building,
					postalCode,
					createdAt,
					updatedAt,
					ptr.To(updatedAt.Add(-time.Minute)),
				)

				assert.Nil(t, actual)
				assert.ErrorIs(t, err, ErrInvalidDeletedAt)
			})
		})
	})
}

func TestEntity_Accessors(t *testing.T) {
	t.Parallel()
	baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	id := uuid.NewTestFromSalt(t, "user")
	prefectureID := uuid.NewTestFromSalt(t, "prefecture")
	firstName := "John"
	lastName := "Doe"
	passwordHash := "hashed_password"
	email := "john.doe@example.com"
	phone := "1234567890"
	city := "Shibuya"
	street := "1-2-3"
	postalCode := "150-0001"
	building := ptr.To("Building A")
	createdAt := baseTime
	updatedAt := baseTime.Add(time.Hour)
	deletedAt := ptr.To(updatedAt.Add(time.Minute))

	expected, err := New(
		id,
		firstName,
		lastName,
		passwordHash,
		email,
		phone,
		prefectureID,
		city,
		street,
		building,
		postalCode,
		createdAt,
		updatedAt,
		deletedAt,
	)
	require.NoError(t, err)

	t.Run("IDメソッドが保存した文字列をuuid.UUIDに変換した値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.ID()
		assert.Equal(t, expected.id, actual)
	})

	t.Run("FirstNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.FirstName()
		assert.Equal(t, expected.firstName, actual)
	})

	t.Run("LastNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.LastName()
		assert.Equal(t, expected.lastName, actual)
	})

	t.Run("PasswordHashメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.PasswordHash()
		assert.Equal(t, expected.passwordHash, actual)
	})

	t.Run("Emailメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Email()
		assert.Equal(t, expected.email, actual)
	})

	t.Run("Phoneメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Phone()
		assert.Equal(t, expected.phone, actual)
	})

	t.Run("PrefectureIDメソッドが保存した文字列をuuid.UUIDに変換した値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.PrefectureID()
		assert.Equal(t, expected.prefectureID, actual)
	})

	t.Run("Cityメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.City()
		assert.Equal(t, expected.city, actual)
	})

	t.Run("Streetメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Street()
		assert.Equal(t, expected.street, actual)
	})

	t.Run("Buildingメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Building()
		assert.Equal(t, expected.building, actual)
	})

	t.Run("PostalCodeメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.PostalCode()
		assert.Equal(t, expected.postalCode, actual)
	})

	t.Run("DeletedAtメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.DeletedAt()
		assert.Equal(t, expected.deletedAt, actual)
	})

	t.Run("CreatedAtメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.CreatedAt()
		assert.Equal(t, expected.createdAt, actual)
	})

	t.Run("UpdatedAtメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.UpdatedAt()
		assert.Equal(t, expected.updatedAt, actual)
	})

	t.Run("FullNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.FullName()
		assert.Equal(t, expected.firstName+" "+expected.lastName, actual)
	})
}

func TestImmutableAccessors(t *testing.T) {
	t.Parallel()
	baseTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	id := uuid.NewTestFromSalt(t, "user")
	prefectureID := uuid.NewTestFromSalt(t, "prefecture")
	firstName := "John"
	lastName := "Doe"
	passwordHash := "hashed_password"
	email := "john.doe@example.com"
	phone := "1234567890"
	city := "Shibuya"
	street := "1-2-3"
	postalCode := "150-0001"
	building := ptr.To("Building A")
	createdAt := baseTime
	updatedAt := baseTime.Add(time.Hour)
	deletedAt := ptr.To(updatedAt.Add(time.Minute))
	// 共有ポインタ building / deletedAt を直接 mutate して不変性を検証するため、
	// 同じポインタを読む deletedAt ブロックと並列実行すると -race で競合する。意図的に直列化する。
	t.Run("buildingのポインタの場合", func(t *testing.T) {
		user, err := New(
			id,
			firstName,
			lastName,
			passwordHash,
			email,
			phone,
			prefectureID,
			city,
			street,
			building,
			postalCode,
			createdAt,
			updatedAt,
			deletedAt,
		)
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

	// building ブロックと同じ共有ポインタを mutate するため、同様に直列実行する。
	t.Run("deletedAtのポインタの場合", func(t *testing.T) {
		user, err := New(
			id,
			firstName,
			lastName,
			passwordHash,
			email,
			phone,
			prefectureID,
			city,
			street,
			building,
			postalCode,
			createdAt,
			updatedAt,
			deletedAt,
		)
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
}

// newValidUser は、削除されていない有効なユーザーと基準時刻を返すテストヘルパー。
func newValidUser(t *testing.T) (*User, time.Time) {
	t.Helper()
	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	u, err := New(
		uuid.NewTestFromSalt(t, "user"),
		"John", "Doe", "hashed_password", "john@example.com", "1234567890",
		uuid.NewTestFromSalt(t, "prefecture"),
		"Shibuya", "1-2-3", ptr.To("Building A"), "150-0001",
		base, base, nil,
	)
	require.NoError(t, err)
	return u, base
}

func TestUser_UpdateProfile(t *testing.T) {
	t.Parallel()

	newPrefID := uuid.NewTestFromSalt(t, "prefecture2")

	t.Run("正常系_全フィールドと更新日時が置き換わる", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		newUpdatedAt := base.Add(time.Hour)

		err := u.UpdateProfile("Jane", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", ptr.To("Tower"), newUpdatedAt)
		require.NoError(t, err)

		assert.Equal(t, "Jane", u.firstName)
		assert.Equal(t, "Smith", u.lastName)
		assert.Equal(t, "jane@example.com", u.email)
		assert.Equal(t, newPrefID, u.prefectureID)
		assert.Equal(t, "Minato", u.city)
		assert.Equal(t, newUpdatedAt, u.updatedAt)
	})

	t.Run("異常系_プロフィールフィールドが不正な場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.UpdateProfile("", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", nil, base.Add(time.Hour))
		assert.ErrorIs(t, err, ErrInvalidFirstName)
	})

	t.Run("異常系_updatedAtがcreatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.UpdateProfile("Jane", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", nil, base.Add(-time.Hour))
		assert.ErrorIs(t, err, ErrInvalidUpdatedAt)
	})

	t.Run("異常系_論理削除済みユーザーは更新できない", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

		err := u.UpdateProfile("Jane", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", nil, base.Add(2*time.Hour))
		assert.ErrorIs(t, err, ErrAlreadyDeleted)
	})
}

func TestUser_ChangePassword(t *testing.T) {
	t.Parallel()

	t.Run("正常系_パスワードハッシュと更新日時が置き換わる", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		newUpdatedAt := base.Add(time.Hour)

		err := u.ChangePassword("new_hashed_password", newUpdatedAt)
		require.NoError(t, err)
		assert.Equal(t, "new_hashed_password", u.passwordHash)
		assert.Equal(t, newUpdatedAt, u.updatedAt)
	})

	t.Run("異常系_パスワードハッシュが不正な場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.ChangePassword("", base.Add(time.Hour))
		assert.ErrorIs(t, err, ErrInvalidPasswordHash)
	})

	t.Run("異常系_updatedAtがcreatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.ChangePassword("new_hashed_password", base.Add(-time.Hour))
		assert.ErrorIs(t, err, ErrInvalidUpdatedAt)
	})

	t.Run("異常系_論理削除済みユーザーはパスワード変更できない", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

		err := u.ChangePassword("new_hashed_password", base.Add(2*time.Hour))
		assert.ErrorIs(t, err, ErrAlreadyDeleted)
	})
}

func TestUser_MarkAsDeleted(t *testing.T) {
	t.Parallel()

	t.Run("正常系_deletedAtが設定される", func(t *testing.T) {
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

	t.Run("異常系_既に削除済みの場合_ErrAlreadyDeletedを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

		err := u.MarkAsDeleted(base.Add(2 * time.Hour))
		assert.ErrorIs(t, err, ErrAlreadyDeleted)
	})

	t.Run("異常系_deletedAtがcreatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.MarkAsDeleted(base.Add(-time.Hour))
		assert.ErrorIs(t, err, ErrInvalidDeletedAt)
	})

	t.Run("異常系_deletedAtがupdatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		u, err := New(
			uuid.NewTestFromSalt(t, "user"),
			"John", "Doe", "hashed_password", "john@example.com", "1234567890",
			uuid.NewTestFromSalt(t, "prefecture"),
			"Shibuya", "1-2-3", ptr.To("Building A"), "150-0001",
			base, base.Add(2*time.Hour), nil,
		)
		require.NoError(t, err)

		err = u.MarkAsDeleted(base.Add(time.Hour))
		assert.ErrorIs(t, err, ErrInvalidDeletedAt)
	})
}
