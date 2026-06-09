//go:generate oapi-codegen --include-tags=debug/cookie --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=debug/cookie --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package cookie パッケージは、クッキーの操作に関連する機能を提供します。
//
//	このハンドラは、動作確認が目的であり、boilerplateから各サービスを作成する際にはセキュリティ上の懸念が多いので削除してください。
package cookie

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"go-boilerplate/internal/controller/handler/debug/cookie/gen"

	"github.com/labstack/echo/v4"
)

const rawSetCookieSample = "__Host-access_token=rawtoken; Path=/hoge; Domain=example.com; SameSite=None"

type server struct{}

func BindHandler(e *echo.Echo) {
	gen.RegisterHandlers(e, &server{})
}

// GetDebugCookie は、リクエストに来た Cookie を返す（Cookie jar を使ったE2E確認用）
func (h *server) GetDebugCookie(ctx echo.Context) error {
	raw := ctx.Request().Header.Get("Cookie")

	out := map[string]string{}
	for _, c := range ctx.Cookies() {
		out[c.Name] = c.Value
	}

	res := gen.DebugCookieInspectResponse{
		RawCookieHeader: raw,
		Cookies:         out,
	}
	return ctx.JSON(http.StatusOK, res)
}

// PostDebugCookie は、任意の Set-Cookie を返す（SecurityCookie MW の rewrite 確認用）
func (h *server) PostDebugCookie(ctx echo.Context) error {
	var body gen.DebugIssueCookieRequest
	if err := ctx.Bind(&body); err != nil {
		return fmt.Errorf("debug cookie bind: %w", err)
	}

	ctx.Response().Header().Add("Set-Cookie", body.SetCookie)
	return ctx.NoContent(http.StatusNoContent)
}

// DeleteDebugCookie は、 Cookie を削除する Set-Cookie を返す（name/path は query）
func (h *server) DeleteDebugCookie(ctx echo.Context, params gen.DeleteDebugCookieParams) error {
	name := "__Host-access_token"
	if params.Name != nil && *params.Name != "" {
		name = *params.Name
	}

	path := "/"
	if params.Path != nil && *params.Path != "" {
		path = *params.Path
	}

	// Max-Age=0 + Expires 過去で削除
	exp := time.Unix(0, 0).UTC().Format(http.TimeFormat)
	setCookie := fmt.Sprintf("%s=; Path=%s; Max-Age=0; Expires=%s", name, path, exp)

	ctx.Response().Header().Add("Set-Cookie", setCookie)
	return ctx.NoContent(http.StatusNoContent)
}

// GetDebugCookieRawCopy は、ReadFrom/io.Copy 経路でレスポンスを書く。
func (h *server) GetDebugCookieRawCopy(ctx echo.Context) error {
	// rewrite確認用の素材をセット（cookie MW が上書きする想定）
	ctx.Response().Header().Add("Set-Cookie", rawSetCookieSample)
	ctx.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")

	// ReadFrom/io.Copy 経路（壊れない＋Set-Cookie rewrite を確認）
	repeat := 1024
	src := strings.NewReader(strings.Repeat("hello-cookie\n", repeat))
	if _, err := io.Copy(ctx.Response().Writer, src); err != nil {
		return fmt.Errorf("debug cookie raw copy: %w", err)
	}
	return nil
}

// GetDebugCookieRawStream は、SSE風のストリームレスポンスを返すエンドポイントです。
func (h *server) GetDebugCookieRawStream(ctx echo.Context) error {
	// rewrite確認用の素材をセット（cookie MW が上書きする想定）
	ctx.Response().Header().Add("Set-Cookie", rawSetCookieSample)
	ctx.Response().Header().Set(echo.HeaderContentType, "text/event-stream")

	// Flush を確実に踏む（SSE風）
	ctx.Response().WriteHeader(http.StatusOK)
	ctx.Response().Flush()

	sleepMillis := 150
	for i := 0; i < 3; i++ {
		_, _ = ctx.Response().Write([]byte("data: ping\n\n"))
		ctx.Response().Flush()
		time.Sleep(time.Duration(sleepMillis) * time.Millisecond)
	}

	return nil
}

// GetDebugCookieRawWs は、WebSocketのハンドシェイクで Set-Cookie を返すエンドポイントです。
func (h *server) GetDebugCookieRawWs(ctx echo.Context) error {
	// WebSocket Upgrade はハンドシェイクレスポンス(101)を `responseHeader` で組み立てるため、ここでは responseHeader に Set-Cookie を明示的に渡す。
	respHdr := http.Header{}
	respHdr.Add("Set-Cookie", rawSetCookieSample)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	// ここで内部的に Hijack 相当の経路が走る想定
	conn, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), respHdr)
	if err != nil {
		// Upgrade 失敗時は gorilla が応答済みのため echo へ伝播しない（二重書き込み回避）。
		return nil
	}
	defer func() { _ = conn.Close() }()

	// 受けたメッセージをそのまま返す（疎通確認）
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		_ = conn.WriteMessage(mt, msg)
	}
}
