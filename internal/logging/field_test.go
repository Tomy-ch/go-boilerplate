package logging

import (
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedString := "string"
	expected := &Field{
		key:         expectedKey,
		kind:        fieldString,
		stringValue: expectedString,
	}

	actual := String(expectedKey, expectedString)
	assert.Equal(t, expected, actual)
}

func TestStrings(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedStrings := []string{"one", "two", "three"}
	expected := &Field{
		key:          expectedKey,
		kind:         fieldStrings,
		stringsValue: expectedStrings,
	}

	actual := Strings(expectedKey, expectedStrings)
	assert.Equal(t, expected, actual)
}

func TestInt(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedInt := 42
	expected := &Field{
		key:      expectedKey,
		kind:     fieldInt,
		intValue: expectedInt,
	}

	actual := Int(expectedKey, expectedInt)
	assert.Equal(t, expected, actual)
}

func TestInt64(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	var expectedInt64 int64 = 4200000000
	expected := &Field{
		key:        expectedKey,
		kind:       fieldInt64,
		int64Value: expectedInt64,
	}

	actual := Int64(expectedKey, expectedInt64)
	assert.Equal(t, expected, actual)
}

func TestFloat64(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedFloat64 := 3.14159
	expected := &Field{
		key:          expectedKey,
		kind:         fieldFloat64,
		float64Value: expectedFloat64,
	}

	actual := Float64(expectedKey, expectedFloat64)
	assert.Equal(t, expected, actual)
}

func TestBool(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedBool := true
	expected := &Field{
		key:       expectedKey,
		kind:      fieldBool,
		boolValue: expectedBool,
	}

	actual := Bool(expectedKey, expectedBool)
	assert.Equal(t, expected, actual)
}

func TestTime(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedTime := time.Date(2024, time.January, 2, 15, 4, 5, 0, time.UTC)
	expected := &Field{
		key:         expectedKey,
		kind:        fieldString,
		stringValue: expectedTime.Format(time.RFC3339Nano),
	}

	actual := Time(expectedKey, expectedTime)
	assert.Equal(t, expected, actual)
}

func TestDurationMs(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedDuration := 1500 * time.Millisecond
	expected := &Field{
		key:          expectedKey,
		kind:         fieldFloat64,
		float64Value: 1500.0,
	}

	actual := DurationMs(expectedKey, expectedDuration)
	assert.Equal(t, expected, actual)
}

func TestError(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedError := xerrors.New("something went wrong")
	expected := &Field{
		key:        expectedKey,
		kind:       fieldError,
		errorValue: expectedError,
	}

	actual := Error(expectedKey, expectedError)
	assert.Equal(t, expected, actual)
}

func TestStacktrace(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedError := xerrors.New("something went wrong")
	expected := &Field{
		key:         expectedKey,
		kind:        fieldString,
		stringValue: xerrors.StackTrace(expectedError),
	}

	actual := Stacktrace(expectedKey, expectedError)
	assert.Equal(t, expected, actual)
}

func TestAny(t *testing.T) {
	t.Parallel()

	const expectedKey = "key"
	expectedAny := map[string]int{"one": 1, "two": 2}
	expected := &Field{
		key:      expectedKey,
		kind:     fieldAny,
		anyValue: expectedAny,
	}

	actual := Any(expectedKey, expectedAny)
	assert.Equal(t, expected, actual)
}

func Test_latencyMs(t *testing.T) {
	t.Parallel()

	t.Run("ミリ秒変換が正しい", func(t *testing.T) {
		t.Parallel()
		ms := latencyMs(250 * time.Millisecond)
		require.InEpsilon(t, float64(250), ms, 0.01)
	})
}
