package logging

import (
	"testing"

	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedString := "string"
	expected := &Field{
		key:         expectedKey,
		kind:        fieldString,
		stringValue: expectedString,
	}

	actual := String(expectedKey, expectedString)
	require.Equal(t, expected, actual)
}

func TestStrings(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedStrings := []string{"one", "two", "three"}
	expected := &Field{
		key:          expectedKey,
		kind:         fieldStrings,
		stringsValue: expectedStrings,
	}

	actual := Strings(expectedKey, expectedStrings)
	require.Equal(t, expected, actual)
}

func TestInt(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedInt := 42
	expected := &Field{
		key:      expectedKey,
		kind:     fieldInt,
		intValue: expectedInt,
	}

	actual := Int(expectedKey, expectedInt)
	require.Equal(t, expected, actual)
}

func TestInt64(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	var expectedInt64 int64 = 4200000000
	expected := &Field{
		key:        expectedKey,
		kind:       fieldInt64,
		int64Value: expectedInt64,
	}

	actual := Int64(expectedKey, expectedInt64)
	require.Equal(t, expected, actual)
}

func TestFloat64(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedFloat64 := 3.14159
	expected := &Field{
		key:          expectedKey,
		kind:         fieldFloat64,
		float64Value: expectedFloat64,
	}

	actual := Float64(expectedKey, expectedFloat64)
	require.Equal(t, expected, actual)
}

func TestBool(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedBool := true
	expected := &Field{
		key:       expectedKey,
		kind:      fieldBool,
		boolValue: expectedBool,
	}

	actual := Bool(expectedKey, expectedBool)
	require.Equal(t, expected, actual)
}

func TestError(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedError := xerrors.New("something went wrong")
	expected := &Field{
		key:        expectedKey,
		kind:       fieldError,
		errorValue: expectedError,
	}

	actual := Error(expectedKey, expectedError)
	require.Equal(t, expected, actual)
}

func TestStacktrace(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedError := xerrors.New("something went wrong")
	expected := &Field{
		key:         expectedKey,
		kind:        fieldString,
		stringValue: xerrors.StackTrace(expectedError),
	}

	actual := Stacktrace(expectedKey, expectedError)
	require.Equal(t, expected, actual)
}

func TestAny(t *testing.T) {
	t.Parallel()

	expectedKey := "key"
	expectedAny := map[string]int{"one": 1, "two": 2}
	expected := &Field{
		key:      expectedKey,
		kind:     fieldAny,
		anyValue: expectedAny,
	}

	actual := Any(expectedKey, expectedAny)
	require.Equal(t, expected, actual)
}
