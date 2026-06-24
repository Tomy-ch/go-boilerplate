package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/pkg/xerrors"
)

// idempotencyHeader は、冪等性キーを伝搬する HTTP ヘッダ名です。
const idempotencyHeader = "Idempotency-Key"

// errCircuitOpen は、circuit breaker による fail-fast を表す内部マーカです。
// ErrUnavailable を内包するため呼び出し側の分類は従来どおりですが、metrics では transport 失敗と
// 区別して計上するために使います。
var errCircuitOpen = xerrors.Wrap(apperror.ErrUnavailable, "circuit open")

// client は、Client の実装です。otelhttp 計装済み transport を介して外部 HTTP 通信を行います。
type client struct {
	httpClient *http.Client
	sleeper    clock.Sleeper
	registry   Registry
	metrics    *observability.HTTPClientMetrics
	budget     *retryBudget
	breakers   *breakerManager
}

// New は、resilient な外部 HTTP 通信の substrate を生成します。
// transport は observability が計装した不透明な outbound transport で、公開 API に net/http 型を露出しません。
func New(
	transport *observability.HTTPClientTransport,
	sleeper clock.Sleeper,
	registry Registry,
	metrics *observability.HTTPClientMetrics,
) Client {
	return &client{
		httpClient: &http.Client{
			Transport:     transport.RoundTripper(),
			CheckRedirect: noFollowRedirect,
		},
		sleeper:  sleeper,
		registry: registry,
		metrics:  metrics,
		budget:   newRetryBudget(),
		breakers: newBreakerManager(),
	}
}

// noFollowRedirect は、リダイレクトを追従せず最終レスポンス（3xx）をそのまま返します。
// 追従先の検証を呼び出し側に委ね、未検証ホストへの自動接続（SSRF 面）を避けます。
func noFollowRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// Do は、req を送信し Response を返します。retry / backoff / deadline 規律を含みます。
func (c *client) Do(ctx context.Context, req *Request) (*Response, error) {
	if req.AllowRetry && req.IdempotencyKey == "" {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "AllowRetry requires IdempotencyKey")
	}

	profile := c.registry.Profile(req.Downstream)
	ds := string(req.Downstream)

	ctx, cancel := context.WithTimeout(ctx, profile.OverallTimeout)
	defer cancel()

	c.metrics.InFlightAdd(ctx, ds, 1)
	defer c.metrics.InFlightAdd(ctx, ds, -1)
	c.budget.refill(req.Downstream, profile.RetryBudgetRatio)

	start := time.Now()
	resp, err := c.doWithRetry(ctx, req, profile, ds)
	c.metrics.RecordLatencyMs(ctx, ds, float64(time.Since(start).Milliseconds()))

	c.recordOutcome(ctx, ds, resp, err)
	return resp, err
}

// doWithRetry は、breaker / budget / backoff を伴う retry ループを回します。
func (c *client) doWithRetry(ctx context.Context, req *Request, profile Profile, ds string) (*Response, error) {
	retrySafe := isRetrySafe(req)
	br := c.breakers.get(req.Downstream, profile.Breaker)

	// MaxAttempts が未設定/不正でも最低 1 回は試行し、(nil, nil) を返さないようにする。
	maxAttempts := max(1, profile.MaxAttempts)

	var resp *Response
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		allowed, generation := br.allow(time.Now())
		if !allowed {
			c.metrics.SetBreakerState(ctx, ds, int64(br.currentState()))
			if resp == nil {
				return nil, xerrors.Wrap(errCircuitOpen, ds)
			}
			return resp, err
		}

		resp, err = c.attempt(ctx, req, profile)
		serverFault := isRetryableOutcome(resp, err)
		br.record(!serverFault, time.Now(), generation)
		c.metrics.SetBreakerState(ctx, ds, int64(br.currentState()))

		if !retrySafe || !serverFault {
			return resp, err
		}
		if attempt == maxAttempts {
			return resp, err
		}
		if !c.budget.tryConsume(req.Downstream) {
			return resp, err
		}

		wait := retryWait(attempt, profile, resp)
		if !c.canRetryWithin(ctx, wait) {
			return resp, err
		}

		c.metrics.RecordRetry(ctx, ds)
		if serr := c.sleeper.Sleep(ctx, wait); serr != nil {
			return nil, normalizeTransportError(serr)
		}
	}
	return resp, err
}

// canRetryWithin は、backoff 待機後も overall deadline 内に次の試行を開始できるかを返します。
func (c *client) canRetryWithin(ctx context.Context, backoff time.Duration) bool {
	deadline, ok := ctx.Deadline()
	if !ok {
		return true
	}
	return time.Now().Add(backoff).Before(deadline)
}

// attempt は、1 回の HTTP 試行を行います。
func (c *client) attempt(ctx context.Context, req *Request, profile Profile) (*Response, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, profile.PerAttemptTimeout)
	defer cancel()
	attemptCtx = observability.ContextWithTracePropagation(attemptCtx, profile.PropagateTrace)
	attemptCtx = observability.ContextWithAllowPrivateNetwork(attemptCtx, profile.AllowPrivateNetwork)

	httpReq, err := buildRequest(attemptCtx, req)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, redactErrMessage(err))
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, normalizeTransportError(err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, err := readBody(httpResp.Body, profile.MaxResponseBytes)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrUnavailable, err.Error())
	}

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Header:     Header(httpResp.Header),
		Body:       body,
	}

	if appErr := statusToAppError(httpResp.StatusCode); appErr != nil {
		msg := fmt.Sprintf("downstream %s returned status %d", req.Downstream, httpResp.StatusCode)
		return resp, xerrors.Wrap(appErr, msg)
	}
	return resp, nil
}

// recordOutcome は、呼び出し結果を metrics に計上します。
func (c *client) recordOutcome(ctx context.Context, ds string, resp *Response, err error) {
	if resp != nil {
		c.metrics.RecordRequest(ctx, ds, statusClass(resp.StatusCode))
	}
	if err == nil {
		return
	}

	switch {
	case resp != nil:
		c.metrics.RecordError(ctx, ds, "http_"+statusClass(resp.StatusCode))
	case xerrors.Is(err, errCircuitOpen):
		c.metrics.RecordError(ctx, ds, "circuit_open")
	case xerrors.Is(err, apperror.ErrCanceled):
		c.metrics.RecordError(ctx, ds, "canceled")
	default:
		c.metrics.RecordError(ctx, ds, "transport")
	}
}

// buildRequest は、自前型の Request から net/http のリクエストを組み立てます。
func buildRequest(ctx context.Context, req *Request) (*http.Request, error) {
	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, string(req.Method), req.URL, body)
	if err != nil {
		return nil, err
	}

	for key, values := range req.Header {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}
	if req.IdempotencyKey != "" {
		httpReq.Header.Set(idempotencyHeader, req.IdempotencyKey)
	}
	return httpReq, nil
}

// readBody は、レスポンスボディを maxBytes まで読み込みます。上限超過時はエラーを返します。
// クライアント側のため server 用の http.MaxBytesReader ではなく io.LimitReader で打ち切ります。
func readBody(body io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, xerrors.Wrap(apperror.ErrUnavailable, "response body exceeds max bytes")
	}
	return data, nil
}
