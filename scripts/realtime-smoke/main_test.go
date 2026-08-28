package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

var errDial = xerrors.New("dial refused")

func refuse(context.Context, string, string) (net.Conn, error) { return nil, errDial }

func TestParseOptions(t *testing.T) {
	t.Parallel()

	t.Run("既定値", func(t *testing.T) {
		t.Parallel()

		opts, err := parseOptions(nil)
		require.NoError(t, err)
		assert.Equal(t, defaultDynamoDBEndpoint, opts.dynamoDBEndpoint)
		assert.Equal(t, defaultGoAWSEndpoint, opts.goAWSEndpoint)
		assert.Equal(t, defaultSubscribers, opts.subscribers)
		assert.Equal(t, formatMarkdown, opts.format)
		assert.False(t, opts.keep)
		assert.False(t, opts.strict)
	})

	t.Run("subscribers は 1 以上", func(t *testing.T) {
		t.Parallel()

		_, err := parseOptions([]string{"-subscribers", "0"})
		require.ErrorIs(t, err, errSubscribers)

		opts, err := parseOptions([]string{"-subscribers", "1"})
		require.NoError(t, err)
		assert.Equal(t, 1, opts.subscribers)
	})

	t.Run("未知の format は拒否", func(t *testing.T) {
		t.Parallel()

		_, err := parseOptions([]string{"-format", "yaml"})
		require.ErrorIs(t, err, errFormat)
	})

	t.Run("不明な flag は拒否", func(t *testing.T) {
		t.Parallel()

		_, err := parseOptions([]string{"-bogus"})
		require.Error(t, err)
	})
}

func TestHostPort(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		endpoint string
		want     string
	}{
		"port 明示":     {endpoint: "http://localhost:8000", want: "localhost:8000"},
		"http の既定":    {endpoint: "http://goaws", want: "goaws:80"},
		"https の既定":   {endpoint: "https://dynamodb.us-east-1.amazonaws.com", want: "dynamodb.us-east-1.amazonaws.com:443"},
		"IPv6 はブラケット": {endpoint: "http://[::1]:4100", want: "[::1]:4100"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := hostPort(tt.endpoint)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRun_endpointNotReady(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code, err := run(t.Context(), []string{"-ready-timeout", "10ms"}, &out, rand.Reader, refuse)

	require.ErrorIs(t, err, errNotReady)
	require.ErrorIs(t, err, errDial)
	assert.Equal(t, 1, code)
	assert.Empty(t, out.String(), "検査を 1 件も実行していないので表を出さない")
}

func TestRun_invalidFlags(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code, err := run(t.Context(), []string{"-subscribers", "0"}, &out, rand.Reader, refuse)

	require.ErrorIs(t, err, errSubscribers)
	assert.Equal(t, 1, code)
}

func TestWaitTCP_readyAfterRetry(t *testing.T) {
	t.Parallel()

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	attempts := 0
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		attempts++
		if attempts == 1 {
			return nil, errDial
		}

		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	require.NoError(t, waitTCP(ctx, dial, ln.Addr().String()))
	assert.Equal(t, 2, attempts)
}
