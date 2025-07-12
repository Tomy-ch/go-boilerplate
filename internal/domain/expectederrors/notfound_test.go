package expectedErrors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDefinedNotFoundCause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    NotFoundCause
		expected bool
	}{
		{
			name:     "定義済み: DB",
			input:    NotFoundCauseDB,
			expected: true,
		},
		{
			name:     "定義済み: Cognito",
			input:    NotFoundCauseCognito,
			expected: true,
		},
		{
			name:     "未定義: 空文字列",
			input:    NotFoundCause(""),
			expected: false,
		},
		{
			name:     "未定義: 不正な文字列",
			input:    NotFoundCause("unknown"),
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := IsDefinedNotFoundCause(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
