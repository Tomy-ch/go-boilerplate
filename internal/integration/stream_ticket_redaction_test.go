package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/controller/httpstack/redaction"
	"go-boilerplate/internal/logging"
)

// contextMapper は、観測したログ 1 件のフィールドを map で返す（zap の observer が持つ LoggedEntry の形）。
type contextMapper interface {
	ContextMap() map[string]any
}

// loggedText は、観測したログの全フィールドを 1 本の文字列にする（どのキーに出たかを問わず生値を探すため）。
func loggedText[E contextMapper](t *testing.T, entries []E) string {
	t.Helper()
	var sb strings.Builder
	for _, entry := range entries {
		raw, err := json.Marshal(entry.ContextMap())
		require.NoError(t, err)
		sb.Write(raw)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// entriesMentioning は、path を含むログだけを返す（並列に走る別ケースのログと混ざらないよう destination で絞る）。
func entriesMentioning[E contextMapper](t *testing.T, entries []E, path string) []E {
	t.Helper()
	var out []E
	for _, entry := range entries {
		if strings.Contains(loggedText(t, []E{entry}), path) {
			out = append(out, entry)
		}
	}
	return out
}

func TestStreamTicketRedaction_Integration(t *testing.T) {
	t.Parallel()

	logger, observed := logging.NewObservedTestLogger(t)
	srv := newStreamServer(t, logger, stubCursorValidator{})

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("拒否された接続のエラーログはticketの生値を含まず秘匿済みの印を持つ", func(t *testing.T) {
			t.Parallel()
			const raw = "rejected-raw-ticket-must-not-be-logged"

			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-rejected?ticket="+raw+"&after=1", nil, nil)
			require.Equal(t, http.StatusUnauthorized, res.StatusCode)

			entries := entriesMentioning(t, observed.All(), "/v1/streams/stream-rejected")
			require.NotEmpty(t, entries, "401 はエラーハンドラがログに出すはず")
			text := loggedText(t, entries)
			assert.NotContains(t, text, raw)
			assert.Contains(t, text, redaction.RedactedValue)
		})

		t.Run("受け入れた接続のアクセスログはticketの生値を含まず秘匿済みの印を持つ", func(t *testing.T) {
			t.Parallel()

			res := srv.DoJSON(http.MethodGet, "/v1/streams/stream-1?ticket="+streamTestTicket+"&after=10", nil, nil)
			require.Equal(t, http.StatusOK, res.StatusCode)

			entries := entriesMentioning(t, observed.All(), "/v1/streams/stream-1?")
			require.NotEmpty(t, entries, "200 はアクセスログに出るはず")
			text := loggedText(t, entries)
			assert.NotContains(t, text, streamTestTicket)
			assert.Contains(t, text, redaction.RedactedValue)
		})

		t.Run("stream以外のpathに付いたticketもアクセスログでは秘匿される", func(t *testing.T) {
			t.Parallel()
			const raw = "stray-raw-ticket-on-another-path"

			// spec が宣言する公開 operation のうち実装を持たないもの。validator を通り、アクセスログが出て、handler が無いので 404。
			res := srv.DoJSON(http.MethodGet, "/_internal/types/error-response?ticket="+raw, nil, nil)
			require.Equal(t, http.StatusNotFound, res.StatusCode)

			entries := entriesMentioning(t, observed.All(), "/_internal/types/error-response")
			require.NotEmpty(t, entries)
			text := loggedText(t, entries)
			assert.NotContains(t, text, raw)
			assert.Contains(t, text, redaction.RedactedValue)
		})
	})
}
