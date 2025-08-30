package user

import (
	"strings"
	"testing"
	"time"

	"boilerplate-go/internal/domain/user/prefecture"
	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/uuid"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	id := uuid.NewTestFromSalt(t, "user")
	prefectureID := uuid.NewTestFromSalt(t, "prefecture")
	firstName := "John"
	lastName := "Doe"
	passwordHash := "hashed_password"
	email := "john.doe@example.com"
	phone := "1234567890"
	prefectureName := "Tokyo"
	city := "Shibuya"
	street := "1-2-3"
	postalCode := "150-0001"
	building := ptr.To("Building A")
	deletedAt := ptr.To(time.Now().AddDate(-1, 0, 0))

	expected := &Entity{
		id:             id,
		firstName:      firstName,
		lastName:       lastName,
		password:       passwordHash,
		email:          email,
		phone:          phone,
		prefectureID:   prefectureID,
		prefectureName: prefectureName,
		city:           city,
		street:         street,
		building:       building,
		postalCode:     postalCode,
		deletedAt:      deletedAt,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("全ての入力が正しい場合、エンティティが生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(
				id.String(),
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				prefectureID.String(),
				prefectureName,
				city,
				street,
				building,
				postalCode,
				deletedAt,
			)

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("IDが無効な場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			idStr := "invalid-id"
			actual, err := New(
				idStr,
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				prefectureID.String(),
				prefectureName,
				city,
				street,
				building,
				postalCode,
				deletedAt,
			)

			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("firstNameが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("firstNameの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				actual, err := New(
					id.String(),
					strings.Repeat("名", minLength-1),
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidFirstName)
			})

			t.Run("firstNameの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				actual, err := New(
					id.String(),
					strings.Repeat("名", maxFirstNameLength+1),
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
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
					id.String(),
					firstName,
					strings.Repeat("姓", minLength-1),
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidLastName)
			})

			t.Run("lastNameの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					strings.Repeat("姓", maxLastNameLength+1),
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidLastName)
			})
		})

		t.Run("prefectureNameが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("prefectureNameの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					strings.Repeat("県", prefecture.MinLength-1),
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPrefectureName)
			})

			t.Run("prefectureNameの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					strings.Repeat("県", prefecture.MaxPrefectureNameLength+1),
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPrefectureName)
			})
		})

		t.Run("passwordが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("passwordの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					strings.Repeat("a", minLength-1),
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPassword)
			})

			t.Run("passwordの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					strings.Repeat("a", maxPasswordLength+1),
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPassword)
			})
		})

		t.Run("emailが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("emailの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					strings.Repeat("a", minLength-1),
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidEmail)
			})

			t.Run("emailの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					strings.Repeat("a", maxEmailLength+1),
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
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
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					strings.Repeat("1", minLength-1),
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPhone)
			})

			t.Run("phoneの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					strings.Repeat("1", maxPhoneLength+1),
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPhone)
			})
		})

		t.Run("deletedAtが未来の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(
				id.String(),
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				prefectureID.String(),
				prefectureName,
				city,
				street,
				building,
				postalCode,
				ptr.To(time.Now().AddDate(0, 0, 1)),
			)

			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidDeletedAt)
		})

		t.Run("cityが範囲外の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			t.Run("cityの文字数が最小値未満の場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					strings.Repeat("市", minLength-1),
					street,
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidCity)
			})

			t.Run("cityの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					strings.Repeat("市", maxCityLength+1),
					street,
					building,
					postalCode,
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
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					strings.Repeat("番", minLength-1),
					building,
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidStreet)
			})

			t.Run("streetの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					strings.Repeat("番", maxStreetLength+1),
					building,
					postalCode,
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
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					ptr.To(strings.Repeat("建", minLength-1)),
					postalCode,
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidBuilding)
			})

			t.Run("buildingの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					ptr.To(strings.Repeat("建", maxBuildingLength+1)),
					postalCode,
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
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					strings.Repeat("0", minLength-1),
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPostalCode)
			})

			t.Run("postalCodeの文字数が最大値を超える場合、エラーを返す", func(t *testing.T) {
				t.Parallel()
				actual, err := New(
					id.String(),
					firstName,
					lastName,
					passwordHash,
					email,
					phone,
					prefectureID.String(),
					prefectureName,
					city,
					street,
					building,
					strings.Repeat("0", maxPostalCodeLength+1),
					deletedAt,
				)

				require.Nil(t, actual)
				require.ErrorIs(t, err, ErrInvalidPostalCode)
			})
		})

		t.Run("prefectureIDが無効な場合、エラーを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(
				id.String(),
				firstName,
				lastName,
				passwordHash,
				email,
				phone,
				"invalid-prefecture-id",
				prefectureName,
				city,
				street,
				building,
				postalCode,
				deletedAt,
			)

			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPrefectureID)
		})
	})
}

func TestGetterMethods(t *testing.T) {
	t.Parallel()
	id := uuid.NewTestFromSalt(t, "user")
	prefectureID := uuid.NewTestFromSalt(t, "prefecture")
	firstName := "John"
	lastName := "Doe"
	passwordHash := "hashed_password"
	email := "john.doe@example.com"
	phone := "1234567890"
	prefectureName := "Tokyo"
	city := "Shibuya"
	street := "1-2-3"
	postalCode := "150-0001"
	building := ptr.To("Building A")
	deletedAt := ptr.To(time.Now().AddDate(-1, 0, 0))

	expected := &Entity{
		id:             id,
		firstName:      firstName,
		lastName:       lastName,
		password:       passwordHash,
		email:          email,
		phone:          phone,
		prefectureID:   prefectureID,
		prefectureName: prefectureName,
		city:           city,
		street:         street,
		building:       building,
		postalCode:     postalCode,
		deletedAt:      deletedAt,
	}

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

	t.Run("Passwordメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.Password()
		require.Equal(t, expected.password, actual)
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

	t.Run("PrefectureNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.PrefectureName()
		require.Equal(t, expected.prefectureName, actual)
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

	t.Run("FullNameメソッドが保存した正しい値を返す", func(t *testing.T) {
		t.Parallel()

		expected := expected
		actual := expected.FullName()
		require.Equal(t, expected.firstName+" "+expected.lastName, actual)
	})
}
