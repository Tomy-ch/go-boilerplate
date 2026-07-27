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

// client は、Client の実装です。
type client struct {
	httpClient *http.Client
	sleeper    clock.Sleeper
	clk        clock.Clock
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
	clk clock.Clock,
	registry Registry,
	metrics *observability.HTTPClientMetrics,
) Client {
	return &client{
		httpClient: &http.Client{
			Transport:     transport.RoundTripper(),
			CheckRedirect: noFollowRedirect,
		},
		sleeper:  sleeper,
		clk:      clk,
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

// validate は、型で防げない Request の precondition を検査します。
// 空 Downstream / ゼロ値 Method / AllowRetry の空 key はいずれも ErrInvalidArgument で弾きます。
func (r *Request) validate() error {
	switch {
	case r.downstream == "":
		return errDownstreamRequired
	case r.method == (Method{}):
		return errMethodRequired
	case r.allowRetry && r.idempotencyKey == "":
		return errIdempotencyKeyRequired
	}
	return nil
}

// Do は、req を送信し Response を返します。retry / backoff / deadline 規律を含みます。
func (c *client) Do(ctx context.Context, req *Request) (*Response, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	profile := c.registry.Profile(req.downstream)
	ds := string(req.downstream)

	// overall I/O デッドラインは実時間で設定する。context のタイマーは実時刻基準のため、
	// 注入 clock を絶対期限に使うと実時刻とのズレで期限が意図せずずれる。
	ctx, cancel := context.WithTimeout(ctx, profile.OverallTimeout)
	defer cancel()
	// retry 可否は注入 clock のタイムライン上で決定的に判定するため、期限を注入 clock 基準でも保持する。
	overallDeadline := c.clk.Now().Add(profile.OverallTimeout)

	c.metrics.InFlightAdd(ctx, ds, 1)
	defer c.metrics.InFlightAdd(ctx, ds, -1)
	c.budget.refill(req.downstream, profile.RetryBudgetRatio)

	start := c.clk.Now()
	resp, err := c.doWithRetry(ctx, req, profile, ds, overallDeadline)
	c.metrics.RecordLatencyMs(ctx, ds, float64(c.clk.Now().Sub(start).Milliseconds()))

	c.recordOutcome(ctx, ds, resp, err)
	return resp, err
}

// doWithRetry は、breaker / budget / backoff を伴う retry ループを回します。
// overallDeadline は注入 clock 基準の overall デッドラインで、retry 可否判定に用います。
func (c *client) doWithRetry(ctx context.Context, req *Request, profile Profile, ds string, overallDeadline time.Time) (*Response, error) {
	retrySafe := isRetrySafe(req)
	br := c.breakers.get(req.downstream, profile.Breaker)

	// breaker 状態は level（gauge）なので、最終状態を呼び出し終了時に 1 回だけ記録する。
	defer func() { c.metrics.SetBreakerState(ctx, ds, int64(br.currentState())) }()

	// MaxAttempts が未設定/不正でも最低 1 回は試行し、(nil, nil) を返さないようにする。
	maxAttempts := max(1, profile.MaxAttempts)

	var resp *Response
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		allowed, generation := br.allow(c.clk.Now())
		if !allowed {
			if resp == nil {
				return nil, xerrors.Wrap(errCircuitOpen, ds)
			}
			return resp, err
		}

		resp, err = c.attempt(ctx, req, profile)
		serverFault := isRetryableOutcome(resp, err)
		br.record(!serverFault, c.clk.Now(), generation)

		if !retrySafe || !serverFault {
			return resp, err
		}
		if attempt == maxAttempts {
			return resp, err
		}
		if !c.budget.tryConsume(req.downstream) {
			return resp, err
		}

		wait := retryWait(attempt, profile, resp, c.clk.Now())
		if !c.canRetryWithin(overallDeadline, wait) {
			return resp, err
		}

		c.metrics.RecordRetry(ctx, ds)
		if serr := c.sleeper.Sleep(ctx, wait); serr != nil {
			return nil, normalizeTransportError(serr)
		}
	}
	return resp, err
}

// canRetryWithin は、backoff 待機後も overall デッドライン内に次の試行を開始できるかを、
// 注入 clock のタイムライン上で判定します（実時間・jitter に依存しない決定的判定）。
func (c *client) canRetryWithin(overallDeadline time.Time, backoff time.Duration) bool {
	return c.clk.Now().Add(backoff).Before(overallDeadline)
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
		// 上限超過は決定的失敗(errResponseTooLarge=ErrInvalidArgument)なので、そのまま返して retry させない。
		// 真の read 失敗(接続断など応答未取得)は transport 失敗として ErrUnavailable で包み retry 対象に残す。
		if xerrors.Is(err, errResponseTooLarge) {
			return nil, err
		}
		return nil, xerrors.Wrap(apperror.ErrUnavailable, redactErrMessage(err))
	}

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Header:     Header(httpResp.Header),
		Body:       body,
	}

	if appErr := statusToAppError(httpResp.StatusCode); appErr != nil {
		msg := fmt.Sprintf("downstream %s returned status %d", req.downstream, httpResp.StatusCode)
		return resp, xerrors.Wrap(appErr, msg)
	}
	return resp, nil
}

// recordOutcome は、呼び出し結果を metrics に計上します。
func (c *client) recordOutcome(ctx context.Context, ds string, resp *Response, err error) {
	var class string
	if resp != nil {
		class = statusClass(resp.StatusCode)
		c.metrics.RecordRequest(ctx, ds, class)
	}
	if err == nil {
		return
	}

	switch {
	case resp != nil:
		c.metrics.RecordError(ctx, ds, "http_"+class)
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
	if len(req.body) > 0 {
		body = bytes.NewReader(req.body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method.String(), req.url, body)
	if err != nil {
		return nil, err
	}

	for key, values := range req.header {
		for _, v := range values {
			httpReq.Header.Add(key, v)
		}
	}
	if req.idempotencyKey != "" {
		httpReq.Header.Set(idempotencyHeader, req.idempotencyKey)
	}
	return httpReq, nil
}

// readBody は、レスポンスボディを maxBytes まで読み込みます。上限超過時はエラーを返します。
func readBody(body io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errResponseTooLarge
	}
	return data, nil
}
