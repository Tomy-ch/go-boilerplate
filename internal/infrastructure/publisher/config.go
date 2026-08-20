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
// 空・不正な URL は relay 起動時点で弾きます（理由は README.md の Choosing an implementation）。
func NewEndpoint(epCfg *config.EndpointConfig) (Endpoint, error) {
	return parseEndpoint(epCfg.Outbox())
}

// parseEndpoint は、エンドポイント文字列を検証して Endpoint へ変換します。
func parseEndpoint(raw string) (Endpoint, error) {
	if raw == "" {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "ENDPOINT_OUTBOX must not be empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", xerrors.Join(ErrInvalidEndpoint, xerrors.Wrap(err, "ENDPOINT_OUTBOX is not a valid URL"))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "ENDPOINT_OUTBOX scheme must be http or https")
	}
	if u.Host == "" {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "ENDPOINT_OUTBOX must include a host")
	}
	if u.Hostname() == "" {
		return "", xerrors.Wrap(ErrInvalidEndpoint, "ENDPOINT_OUTBOX must include a hostname")
	}

	return Endpoint(raw), nil
}
