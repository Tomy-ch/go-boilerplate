package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

var errDial = xerrors.New("dial refused")

func refuse(context.Context, string, string) (net.Conn, error) { return nil, errDial }

func realDial(ctx context.Context, network, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

func Test_parseOptions(t *testing.T) {
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

func Test_hostPort(t *testing.T) {
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

	t.Run("scheme 省略は :80 に化けず入力ミスとして止める", func(t *testing.T) {
		t.Parallel()

		_, err := hostPort("localhost:8000")
		require.ErrorIs(t, err, errEndpoint)
	})

	t.Run("http / https 以外の scheme は拒否", func(t *testing.T) {
		t.Parallel()

		_, err := hostPort("ftp://localhost:8000")
		require.ErrorIs(t, err, errEndpoint)
	})
}

func Test_waitTCP(t *testing.T) {
	t.Parallel()

	t.Run("接続できるまで再試行する", func(t *testing.T) {
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

			return realDial(ctx, network, addr)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		require.NoError(t, waitTCP(ctx, dial, ln.Addr().String()))
		assert.Equal(t, 2, attempts)
	})

	t.Run("期限までに繋がらなければ最後の dial エラーを返す", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		err := waitTCP(ctx, refuse, "127.0.0.1:1")
		require.ErrorIs(t, err, errDial)
	})
}

func Test_waitReady(t *testing.T) {
	t.Parallel()

	t.Run("繋がらない endpoint は errNotReady と dial エラーの両方を chain に持つ", func(t *testing.T) {
		t.Parallel()

		err := waitReady(t.Context(), refuse, 10*time.Millisecond, "http://localhost:8000")
		require.ErrorIs(t, err, errNotReady)
		require.ErrorIs(t, err, errDial)
	})

	t.Run("不正な endpoint は dial せずに止める", func(t *testing.T) {
		t.Parallel()

		dialed := false
		dial := func(context.Context, string, string) (net.Conn, error) {
			dialed = true

			return nil, errDial
		}

		err := waitReady(t.Context(), dial, 10*time.Millisecond, "localhost:8000")
		require.ErrorIs(t, err, errEndpoint)
		assert.False(t, dialed)
	})

	t.Run("全 endpoint に繋がれば nil", func(t *testing.T) {
		t.Parallel()

		f := newAWSFake(t)
		require.NoError(t, waitReady(t.Context(), realDial, time.Second, f.server.URL, f.server.URL))
	})
}

func Test_newClients(t *testing.T) {
	t.Parallel()

	c, err := newClients(t.Context(), options{
		dynamoDBEndpoint: "http://localhost:8000",
		goAWSEndpoint:    "http://localhost:4100",
		region:           "ap-northeast-1",
	})
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8000", aws.ToString(c.dynamoDB.Options().BaseEndpoint))
	assert.Equal(t, "http://localhost:4100", aws.ToString(c.sns.Options().BaseEndpoint))
	assert.Equal(t, "http://localhost:4100", aws.ToString(c.sqs.Options().BaseEndpoint))
	assert.Equal(t, "ap-northeast-1", c.sqs.Options().Region)
}

func Test_write(t *testing.T) {
	t.Parallel()

	results := []Result{{ID: "D1", Subject: "s", Check: "c", Verdict: VerdictCompatible, Detail: "d"}}

	var md, text bytes.Buffer
	require.NoError(t, write(&md, formatMarkdown, results))
	require.NoError(t, write(&text, formatText, results))

	assert.Contains(t, md.String(), "| D1 | s | c | **互換** | d |")
	assert.NotContains(t, text.String(), "| D1 |")
	assert.Contains(t, text.String(), "D1 ")
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("不正な flag は表を出さず 1", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		code, err := run(t.Context(), []string{"-subscribers", "0"}, &out, rand.Reader, refuse)

		require.ErrorIs(t, err, errSubscribers)
		assert.Equal(t, 1, code)
		assert.Empty(t, out.String())
	})

	t.Run("-h は 0 で終わる", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		code, err := run(t.Context(), []string{"-h"}, &out, rand.Reader, refuse)

		require.NoError(t, err)
		assert.Equal(t, 0, code)
	})

	t.Run("endpoint が ready にならなければ検査を 1 件も実行せず 1", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		code, err := run(t.Context(), []string{"-ready-timeout", "10ms"}, &out, rand.Reader, refuse)

		require.ErrorIs(t, err, errNotReady)
		require.ErrorIs(t, err, errDial)
		assert.Equal(t, 1, code)
		assert.Empty(t, out.String(), "検査を 1 件も実行していないので表を出さない")
	})

	t.Run("乱数源の失敗は 1", func(t *testing.T) {
		t.Parallel()

		f := newAWSFake(t)

		var out bytes.Buffer
		code, err := run(t.Context(), []string{"-dynamodb-endpoint", f.server.URL, "-goaws-endpoint", f.server.URL},
			&out, failingReader{}, realDial)

		require.ErrorIs(t, err, errRandom)
		assert.Equal(t, 1, code)
	})

	t.Run("全検査を通すと互換だけの表を出して 0", func(t *testing.T) {
		t.Parallel()

		f := newAWSFake(t)
		installDynamoDB(f)
		installGoAWS(f)

		var out bytes.Buffer
		code, err := run(t.Context(), []string{
			"-dynamodb-endpoint", f.server.URL, "-goaws-endpoint", f.server.URL, "-subscribers", "2", "-strict",
		}, &out, rand.Reader, realDial)

		require.NoError(t, err)
		assert.Equal(t, 0, code)
		assert.Contains(t, out.String(), "| D1 | DynamoDB Local |")
		assert.Contains(t, out.String(), "| G9 | GoAWS SNS/SQS |")
		assert.Contains(t, out.String(), "非互換 0 / 未対応 0 / 検証不能 0")
	})

	t.Run("Policy を拒否する emulator は -strict で 1", func(t *testing.T) {
		t.Parallel()

		f := newAWSFake(t)
		installDynamoDB(f)
		installGoAWS(f).rejectPolicy = true

		var out bytes.Buffer
		code, err := run(t.Context(), []string{
			"-dynamodb-endpoint", f.server.URL, "-goaws-endpoint", f.server.URL, "-strict",
		}, &out, rand.Reader, realDial)

		require.NoError(t, err)
		assert.Equal(t, 1, code)
		assert.Contains(t, out.String(), "- G4 ")
		assert.Contains(t, out.String(), "- G4b ")
	})
}
