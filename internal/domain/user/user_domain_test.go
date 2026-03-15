package user

import (
	"strings"
	"testing"
	"time"

	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/uuid"

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
			require.Equal(t, id, actual.id)
			require.Equal(t, firstName, actual.firstName)
			require.Equal(t, lastName, actual.lastName)
			require.Equal(t, passwordHash, actual.passwordHash)
			require.Equal(t, email, actual.email)
			require.Equal(t, phone, actual.phone)
			require.Equal(t, prefectureID, actual.prefectureID)
			require.Equal(t, city, actual.city)
			require.Equal(t, street, actual.street)
			require.Equal(t, *building, *actual.building)
			require.Equal(t, postalCode, actual.postalCode)
			require.Equal(t, createdAt, actual.createdAt)
			require.Equal(t, updatedAt, actual.updatedAt)
			require.Equal(t, *deletedAt, *actual.deletedAt)
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

			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidFirstName)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidFirstName)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidLastName)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidLastName)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPasswordHash)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPasswordHash)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidEmail)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidEmail)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPhone)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPhone)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidCity)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidCity)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidStreet)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidStreet)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidBuilding)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidBuilding)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPostalCode)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPostalCode)
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

			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPrefectureID)
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

			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUpdatedAt)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidDeletedAt)
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

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidDeletedAt)
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
		require.Equal(t, expected.id, actual)
	})

	t.Run("FirstNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.FirstName()
		require.Equal(t, expected.firstName, actual)
	})

	t.Run("LastNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.LastName()
		require.Equal(t, expected.lastName, actual)
	})

	t.Run("PasswordHashメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.PasswordHash()
		require.Equal(t, expected.passwordHash, actual)
	})

	t.Run("Emailメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Email()
		require.Equal(t, expected.email, actual)
	})

	t.Run("Phoneメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Phone()
		require.Equal(t, expected.phone, actual)
	})

	t.Run("PrefectureIDメソッドが保存した文字列をuuid.UUIDに変換した値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.PrefectureID()
		require.Equal(t, expected.prefectureID, actual)
	})

	t.Run("Cityメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.City()
		require.Equal(t, expected.city, actual)
	})

	t.Run("Streetメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Street()
		require.Equal(t, expected.street, actual)
	})

	t.Run("Buildingメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Building()
		require.Equal(t, expected.building, actual)
	})

	t.Run("PostalCodeメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.PostalCode()
		require.Equal(t, expected.postalCode, actual)
	})

	t.Run("DeletedAtメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.DeletedAt()
		require.Equal(t, expected.deletedAt, actual)
	})

	t.Run("CreatedAtメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.CreatedAt()
		require.Equal(t, expected.createdAt, actual)
	})

	t.Run("UpdatedAtメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.UpdatedAt()
		require.Equal(t, expected.updatedAt, actual)
	})

	t.Run("FullNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.FullName()
		require.Equal(t, expected.firstName+" "+expected.lastName, actual)
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
	t.Run("buildingのポインタの場合", func(t *testing.T) {
		t.Parallel()
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

			require.NotEqual(t, *building, *user.Building())
			require.Equal(t, original, *user.Building())
		})

		t.Run("Buildingメソッドの返り値のポインタを変更しても、ユーザーのbuildingが変更されていないことを確認する", func(t *testing.T) {
			t.Parallel()

			original := *user.Building()

			// Buildingメソッドの返り値のポインタを変更
			actualBuilding := user.Building()
			*actualBuilding = "Building B"

			require.NotEqual(t, *actualBuilding, *user.Building())
			require.Equal(t, original, *user.Building())
		})
	})

	t.Run("deletedAtのポインタの場合", func(t *testing.T) {
		t.Parallel()
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

			require.NotEqual(t, *deletedAt, *user.DeletedAt())
			require.Equal(t, original, *user.DeletedAt())
		})
		t.Run("DeletedAtメソッドの返り値のポインタを変更しても、ユーザーのdeletedAtが変更されていないことを確認する", func(t *testing.T) {
			t.Parallel()

			original := *user.DeletedAt()

			// DeletedAtメソッドの返り値のポインタを変更
			actualDeletedAt := user.DeletedAt()
			*actualDeletedAt = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

			require.NotEqual(t, *actualDeletedAt, *user.DeletedAt())
			require.Equal(t, original, *user.DeletedAt())
		})
	})
}
