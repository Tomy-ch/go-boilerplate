package server

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name string `validate:"required"`
}

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	require.NotNil(t, v)
	require.IsType(t, &CustomValidator{}, v)
}

func TestCustomValidator_Validate(t *testing.T) {
	cv := &CustomValidator{
		validator: validator.New(),
	}

	t.Run("valid struct", func(t *testing.T) {
		validStruct := testStruct{Name: "Valid Name"}
		err := cv.Validate(validStruct)
		require.NoError(t, err)
	})

	t.Run("異常系/無効な構造体", func(t *testing.T) {
		invalidStruct := testStruct{Name: ""}
		err := cv.Validate(invalidStruct)
		require.Error(t, err)
	})
}
