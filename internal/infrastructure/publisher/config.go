package publisher

import (
	"net/url"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/pkg/xerrors"
)

// ErrInvalidEndpoint は、outbox publish 先エンドポイントの設定が不正であることを示すエラーです。
var ErrInvalidEndpoint = xerrors.Wrap(apperror.ErrInvalidArgument, "invalid outbox endpoint")

// NewEndpoint は、config からメッセージ送信先エンドポイント URL を解決して返します。
// 空・不正な URL は relay 起動時点で弾きます（未設定のまま起動すると全 publish が失敗し
// 気付かぬうちに全メッセージが dead 化するため、サイレント障害を防ぐ）。
func NewEndpoint(cfg *config.OutboxConfig) (Endpoint, error) {
	return parseEndpoint(cfg.Endpoint())
}

// parseEndpoint は、エンドポイント文字列を検証して Endpoint へ変換します。
func parseEndpoint(raw string) (Endpoint, error) {
	if raw == "" {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "OUTBOX_ENDPOINT must not be empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "OUTBOX_ENDPOINT is not a valid URL: "+err.Error())
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "OUTBOX_ENDPOINT scheme must be http or https")
	}
	if u.Host == "" {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "OUTBOX_ENDPOINT must include a host")
	}

	return Endpoint(raw), nil
}
