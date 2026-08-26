package uuid

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzParse は、Parse が任意の文字列に対して panic せず、
// 受理した値は String() を経ても同じ値へ再解釈されることを検証します。
func FuzzParse(f *testing.F) {
	validUUID := strings.Join([]string{"b1d4e0f2", "3c5a", "4b6d", "8e7f", "1a2b3c4d5e6f"}, "-")

	for _, seed := range []string{
		validUUID,
		strings.ToUpper(validUUID),
		"00000000-0000-0000-0000-000000000000",
		"urn:uuid:" + validUUID,
		strings.ReplaceAll(validUUID, "-", ""),
		"",
		"not-a-uuid",
		validUUID[:len(validUUID)-1],
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		u, err := Parse(s)
		if err != nil {
			return
		}

		// 受理された ID は自身の文字列表現から同じ値へ戻る。
		again, err := Parse(u.String())
		require.NoError(t, err)
		require.Equal(t, u.String(), again.String())
	})
}
