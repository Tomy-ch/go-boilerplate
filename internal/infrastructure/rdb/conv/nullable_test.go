package conv

import (
	"database/sql"
	"testing"
	"time"

	"boilerplate-go/pkg/uuid"

	googleUUID "github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUUIDPtrFromNull(t *testing.T) {
	t.Parallel()

	t.Run("Validがtrueの場合、ポインタが返される", func(t *testing.T) {
		nullUUID := googleUUID.NullUUID{UUID: googleUUID.MustParse("123e4567-e89b-12d3-a456-426614174000"), Valid: true}
		expected, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
		require.NoError(t, err)

		actual, err := UUIDPtrFromNull(nullUUID)
		require.NoError(t, err)
		require.NotNil(t, actual)
		require.Equal(t, &expected, actual)
	})

	t.Run("Validがfalseの場合、nilが返される", func(t *testing.T) {
		nullUUID := googleUUID.NullUUID{Valid: false}

		actual, err := UUIDPtrFromNull(nullUUID)
		require.NoError(t, err)
		require.Nil(t, actual)
	})
}

func TestNullUUIDFromPtr(t *testing.T) {
	t.Parallel()

	t.Run("ポインタがnilでない場合、ValidがtrueのgoogleUUID.NullUUIDが返される", func(t *testing.T) {
		input, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
		require.NoError(t, err)
		expected := googleUUID.NullUUID{UUID: googleUUID.MustParse("123e4567-e89b-12d3-a456-426614174000"), Valid: true}

		actual := NullUUIDFromPtr(&input)
		require.Equal(t, expected, actual)
	})

	t.Run("ポインタがnilの場合、ValidがfalseのgoogleUUID.NullUUIDが返される", func(t *testing.T) {
		expected := googleUUID.NullUUID{Valid: false}

		actual := NullUUIDFromPtr(nil)
		require.Equal(t, expected, actual)
	})
}

func TestNewNullUUID(t *testing.T) {
	t.Parallel()

	t.Run("UUIDをgoogleUUID.NullUUIDに変換する", func(t *testing.T) {
		t.Parallel()
		input, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
		require.NoError(t, err)
		expected := googleUUID.NullUUID{UUID: googleUUID.MustParse("123e4567-e89b-12d3-a456-426614174000"), Valid: true}

		actual := NewNullUUID(input)
		require.Equal(t, expected, actual)
	})
}

func TestStringPtrFromNull(t *testing.T) {
	t.Parallel()

	t.Run("Validがtrueの場合、ポインタが返される", func(t *testing.T) {
		nullString := sql.NullString{String: "test", Valid: true}
		expected := "test"

		actual := StringPtrFromNull(nullString)
		require.NotNil(t, actual)
		require.Equal(t, &expected, actual)
	})

	t.Run("Validがfalseの場合、nilが返される", func(t *testing.T) {
		nullString := sql.NullString{Valid: false}

		actual := StringPtrFromNull(nullString)
		require.Nil(t, actual)
	})
}

func TestNullStringFromPtr(t *testing.T) {
	t.Parallel()
	t.Run("ポインタがnilでない場合、Validがtrueのsql.NullStringが返される", func(t *testing.T) {
		input := "test"
		expected := sql.NullString{String: "test", Valid: true}

		actual := NullStringFromPtr(&input)
		require.Equal(t, expected, actual)
	})

	t.Run("ポインタがnilの場合、Validがfalseのsql.NullStringが返される", func(t *testing.T) {
		expected := sql.NullString{Valid: false}

		actual := NullStringFromPtr(nil)
		require.Equal(t, expected, actual)
	})
}

func TestInt16PtrFromNull(t *testing.T) {
	t.Parallel()

	t.Run("Validがtrueの場合、ポインタが返される", func(t *testing.T) {
		nullInt16 := sql.NullInt16{Int16: 123, Valid: true}
		expected := int16(123)

		actual := Int16PtrFromNull(nullInt16)
		require.NotNil(t, actual)
		require.Equal(t, &expected, actual)
	})

	t.Run("Validがfalseの場合、nilが返される", func(t *testing.T) {
		nullInt16 := sql.NullInt16{Valid: false}

		actual := Int16PtrFromNull(nullInt16)
		require.Nil(t, actual)
	})
}

func TestNullInt16FromPtr(t *testing.T) {
	t.Parallel()

	t.Run("ポインタがnilでない場合、Validがtrueのsql.NullInt16が返される", func(t *testing.T) {
		input := int16(123)
		expected := sql.NullInt16{Int16: 123, Valid: true}

		actual := NullInt16FromPtr(&input)
		require.Equal(t, expected, actual)
	})

	t.Run("ポインタがnilの場合、Validがfalseのsql.NullInt16が返される", func(t *testing.T) {
		expected := sql.NullInt16{Valid: false}

		actual := NullInt16FromPtr(nil)
		require.Equal(t, expected, actual)
	})
}

func TestInt64PtrFromNull(t *testing.T) {
	t.Parallel()

	t.Run("Validがtrueの場合、ポインタが返される", func(t *testing.T) {
		nullInt64 := sql.NullInt64{Int64: 123456789, Valid: true}
		expected := int64(123456789)

		actual := Int64PtrFromNull(nullInt64)
		require.NotNil(t, actual)
		require.Equal(t, &expected, actual)
	})

	t.Run("Validがfalseの場合、nilが返される", func(t *testing.T) {
		nullInt64 := sql.NullInt64{Valid: false}

		actual := Int64PtrFromNull(nullInt64)
		require.Nil(t, actual)
	})
}

func TestNullInt64FromPtr(t *testing.T) {
	t.Parallel()

	t.Run("ポインタがnilでない場合、Validがtrueのsql.NullInt64が返される", func(t *testing.T) {
		input := int64(123456789)
		expected := sql.NullInt64{Int64: 123456789, Valid: true}

		actual := NullInt64FromPtr(&input)
		require.Equal(t, expected, actual)
	})

	t.Run("ポインタがnilの場合、Validがfalseのsql.NullInt64が返される", func(t *testing.T) {
		expected := sql.NullInt64{Valid: false}

		actual := NullInt64FromPtr(nil)
		require.Equal(t, expected, actual)
	})
}

func TestBoolPtrFromNull(t *testing.T) {
	t.Parallel()

	t.Run("Validがtrueの場合、ポインタが返される", func(t *testing.T) {
		nullBool := sql.NullBool{Bool: true, Valid: true}
		expected := true

		actual := BoolPtrFromNull(nullBool)
		require.NotNil(t, actual)
		require.Equal(t, &expected, actual)
	})

	t.Run("Validがfalseの場合、nilが返される", func(t *testing.T) {
		nullBool := sql.NullBool{Valid: false}

		actual := BoolPtrFromNull(nullBool)
		require.Nil(t, actual)
	})
}

func TestNullBoolFromPtr(t *testing.T) {
	t.Parallel()

	t.Run("ポインタがnilでない場合、Validがtrueのsql.NullBoolが返される", func(t *testing.T) {
		input := true
		expected := sql.NullBool{Bool: true, Valid: true}

		actual := NullBoolFromPtr(&input)
		require.Equal(t, expected, actual)
	})

	t.Run("ポインタがnilの場合、Validがfalseのsql.NullBoolが返される", func(t *testing.T) {
		expected := sql.NullBool{Valid: false}

		actual := NullBoolFromPtr(nil)
		require.Equal(t, expected, actual)
	})
}

func TestFloat64PtrFromNull(t *testing.T) {
	t.Parallel()

	t.Run("Validがtrueの場合、ポインタが返される", func(t *testing.T) {
		nullFloat64 := sql.NullFloat64{Float64: 123.456, Valid: true}
		expected := 123.456

		actual := Float64PtrFromNull(nullFloat64)
		require.NotNil(t, actual)
		require.Equal(t, &expected, actual)
	})

	t.Run("Validがfalseの場合、nilが返される", func(t *testing.T) {
		nullFloat64 := sql.NullFloat64{Valid: false}

		actual := Float64PtrFromNull(nullFloat64)
		require.Nil(t, actual)
	})
}

func TestNullFloat64FromPtr(t *testing.T) {
	t.Parallel()

	t.Run("ポインタがnilでない場合、Validがtrueのsql.NullFloat64が返される", func(t *testing.T) {
		input := 123.456
		expected := sql.NullFloat64{Float64: 123.456, Valid: true}

		actual := NullFloat64FromPtr(&input)
		require.Equal(t, expected, actual)
	})

	t.Run("ポインタがnilの場合、Validがfalseのsql.NullFloat64が返される", func(t *testing.T) {
		expected := sql.NullFloat64{Valid: false}

		actual := NullFloat64FromPtr(nil)
		require.Equal(t, expected, actual)
	})
}

func TestTimePtrFromNull(t *testing.T) {
	t.Parallel()

	t.Run("Validがtrueの場合、ポインタが返される", func(t *testing.T) {
		nullTime := sql.NullTime{Time: time.Unix(0, 0), Valid: true}
		expected := time.Unix(0, 0)

		actual := TimePtrFromNull(nullTime)
		require.NotNil(t, actual)
		require.Equal(t, &expected, actual)
	})

	t.Run("Validがfalseの場合、nilが返される", func(t *testing.T) {
		nullTime := sql.NullTime{Valid: false}

		actual := TimePtrFromNull(nullTime)
		require.Nil(t, actual)
	})
}

func TestNullTimeFromPtr(t *testing.T) {
	t.Parallel()

	t.Run("ポインタがnilでない場合、Validがtrueのsql.NullTimeが返される", func(t *testing.T) {
		input := time.Unix(0, 0)
		expected := sql.NullTime{Time: input, Valid: true}

		actual := NullTimeFromPtr(&input)
		require.Equal(t, expected, actual)
	})

	t.Run("ポインタがnilの場合、Validがfalseのsql.NullTimeが返される", func(t *testing.T) {
		expected := sql.NullTime{Valid: false}

		actual := NullTimeFromPtr(nil)
		require.Equal(t, expected, actual)
	})
}

func TestNewNullString(t *testing.T) {
	t.Parallel()

	t.Run("文字列をsql.NullStringに変換する", func(t *testing.T) {
		t.Parallel()

		input := "test"
		expected := sql.NullString{String: input, Valid: true}

		actual := NewNullString(input)
		require.Equal(t, expected, actual)
	})
}

func TestNewNullInt16(t *testing.T) {
	t.Parallel()

	t.Run("int16をsql.NullInt16に変換する", func(t *testing.T) {
		t.Parallel()

		input := int16(123)
		expected := sql.NullInt16{Int16: input, Valid: true}

		actual := NewNullInt16(input)
		require.Equal(t, expected, actual)
	})
}

func TestNewNullInt64(t *testing.T) {
	t.Parallel()

	t.Run("int64をsql.NullInt64に変換する", func(t *testing.T) {
		t.Parallel()

		input := int64(123456789)
		expected := sql.NullInt64{Int64: input, Valid: true}

		actual := NewNullInt64(input)
		require.Equal(t, expected, actual)
	})
}

func TestNewNullBool(t *testing.T) {
	t.Parallel()

	t.Run("boolをsql.NullBoolに変換する", func(t *testing.T) {
		t.Parallel()

		input := true
		expected := sql.NullBool{Bool: input, Valid: true}

		actual := NewNullBool(input)
		require.Equal(t, expected, actual)
	})
}

func TestNewNullFloat64(t *testing.T) {
	t.Parallel()

	t.Run("float64をsql.NullFloat64に変換する", func(t *testing.T) {
		t.Parallel()

		input := 123.456
		expected := sql.NullFloat64{Float64: input, Valid: true}

		actual := NewNullFloat64(input)
		require.Equal(t, expected, actual)
	})
}

func TestNewNullTime(t *testing.T) {
	t.Parallel()

	t.Run("time.Timeをsql.NullTimeに変換する", func(t *testing.T) {
		t.Parallel()

		input := time.Now()
		expected := sql.NullTime{Time: input, Valid: true}

		actual := NewNullTime(input)
		require.Equal(t, expected, actual)
	})
}
