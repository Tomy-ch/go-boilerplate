// Package jwt は、access token（JWT）を検証する Authenticator 実装を提供します。
//
// 署名鍵は固定 RSA 公開鍵（New）と JWKS エンドポイントからの動的解決（NewJWKS）の両方に対応します。
// 本実装はデファクト標準プロファイル（非対称署名 JWT・iss/aud/exp/nbf/sub・標準 scope）のみを一級で扱います。
// 特定 IdP の方言（Cognito token_use / Azure scp / opaque token 等）は扱わず、README の拡張ポイントに委ねます。
package jwt

import (
	"context"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/pkg/xerrors"
)

const (
	// defaultLeeway は、IdP と検証側のクロックずれを吸収する既定の許容幅です。
	defaultLeeway = 60 * time.Second
	// scopeClaim は、OAuth2 標準のスコープクレーム名です（スペース区切りの単一文字列）。
	scopeClaim = "scope"
)

// defaultAllowedAlgs は、許可アルゴリズムを注入しなかった場合の既定 allowlist です。
// 非対称署名のみに限定し、alg=none / HS256（対称鍵）を構造的に排除します。
var defaultAllowedAlgs = []string{"RS256"}

var (
	// ErrJWTAuthenticatorInvalidToken は、トークン検証に失敗した場合に返す認証エラーです。
	// 検証失敗はすべてこのセンチネル（= apperror.ErrUnauthenticated）へ正規化し（fail-closed）、
	// 原因（署名不一致 / exp / iss / aud / typ 等）はエラーチェーンに保持して運用側で切り分け可能にします。
	ErrJWTAuthenticatorInvalidToken = xerrors.Wrap(apperror.ErrUnauthenticated, "jwt authenticator: invalid token")

	// ErrJWTAuthenticatorInvalidPublicKey は、コンストラクタで公開鍵 PEM のパースに失敗した場合の設定エラーです。
	ErrJWTAuthenticatorInvalidPublicKey = xerrors.New("jwt authenticator: invalid public key")
	// ErrJWTAuthenticatorInvalidParams は、コンストラクタの必須パラメータが不足している場合の設定エラーです。
	ErrJWTAuthenticatorInvalidParams = xerrors.New("jwt authenticator: invalid params")
)

// Params は、JWT Authenticator の検証パラメータです。
type Params struct {
	// PublicKeyPEM は署名検証に用いる RSA 公開鍵（PEM 形式）です（必須）。
	PublicKeyPEM string
	// Issuer は検証する iss クレームの期待値です（必須）。
	Issuer string
	// Audience は検証する aud クレームの期待値です（必須。標準プロファイルでは aud 必須）。
	Audience string
	// AllowedAlgs は許可する署名アルゴリズムの allowlist です（空なら defaultAllowedAlgs）。
	AllowedAlgs []string
	// Leeway は exp / nbf 検証時のクロックずれ許容幅です（0 以下なら defaultLeeway）。
	Leeway time.Duration
	// ExpectedType は typ ヘッダの期待値です（空なら typ 検証をスキップ。RFC 9068 なら "at+jwt"）。
	ExpectedType string
	// Clock は exp / nbf 検証に用いる時刻源です（必須。テスト決定性のため注入）。
	Clock clock.Clock
}

// JWKSParams は、JWKS backed の JWT Authenticator の構築パラメータです。
// 署名検証パラメータ（Params）に加えて JWKS 取得の設定を持ちます。PublicKeyPEM は使用しません。
type JWKSParams struct {
	Params

	// JWKSURL は公開鍵を取得する JWKS エンドポイント URL です（必須）。
	JWKSURL string
	// CacheTTL は取得した JWKS を再取得するまでの間隔です（0 以下は既定 defaultJWKSCacheTTL）。
	CacheTTL time.Duration
}

// authenticator は JWT を検証する Authenticator です。
// 署名鍵の解決は keyResolver に委ね、固定公開鍵と JWKS の双方を同一の検証ロジックで扱います。
type authenticator struct {
	parser       *jwtlib.Parser
	keyResolver  jwtlib.Keyfunc
	expectedType string
}

// New は固定 RSA 公開鍵で検証する JWT Authenticator を生成します。
// 公開鍵 PEM のパース失敗・必須パラメータ不足は設定エラーとして返します（認証エラーとは区別）。
func New(params Params) (authbd.Authenticator, error) {
	publicKey, err := jwtlib.ParseRSAPublicKeyFromPEM([]byte(params.PublicKeyPEM))
	if err != nil {
		return nil, xerrors.Join(ErrJWTAuthenticatorInvalidPublicKey, err)
	}

	return buildAuthenticator(params, func(*jwtlib.Token) (any, error) { return publicKey, nil })
}

// NewWithKeyfunc は、与えられた鍵解決関数で検証する JWT Authenticator を生成します（PublicKeyPEM は不使用）。
// JWKS backed の鍵解決を注入するために使用します。
func NewWithKeyfunc(params Params, keyResolver jwtlib.Keyfunc) (authbd.Authenticator, error) {
	if keyResolver == nil {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidParams, "key resolver is required")
	}

	return buildAuthenticator(params, keyResolver)
}

// NewJWKS は、JWKS エンドポイントから kid で公開鍵を解決する JWT Authenticator を生成します。
// JWKS の取得は httpclient substrate（timeout / retry / breaker / budget / o11y）に委ね、遅延取得します。
func NewJWKS(params JWKSParams, client httpclient.Client) (authbd.Authenticator, error) {
	if strings.TrimSpace(params.JWKSURL) == "" {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidParams, "jwks url is required")
	}
	if client == nil {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidParams, "http client is required")
	}

	resolver := newJWKSResolver(client, params.JWKSURL, params.CacheTTL, params.Clock)
	return NewWithKeyfunc(params.Params, resolver.keyfunc)
}

// buildAuthenticator は共通の検証パラメータを検証して authenticator を生成します。
func buildAuthenticator(params Params, keyResolver jwtlib.Keyfunc) (authbd.Authenticator, error) {
	if params.Clock == nil {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidParams, "clock is required")
	}
	if strings.TrimSpace(params.Issuer) == "" {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidParams, "issuer is required")
	}
	if strings.TrimSpace(params.Audience) == "" {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidParams, "audience is required")
	}

	algs := params.AllowedAlgs
	if len(algs) == 0 {
		algs = defaultAllowedAlgs
	}

	leeway := params.Leeway
	if leeway <= 0 {
		leeway = defaultLeeway
	}

	parser := jwtlib.NewParser(
		jwtlib.WithValidMethods(algs),
		jwtlib.WithIssuer(params.Issuer),
		jwtlib.WithAudience(params.Audience),
		jwtlib.WithExpirationRequired(),
		jwtlib.WithLeeway(leeway),
		jwtlib.WithTimeFunc(params.Clock.Now),
	)

	return &authenticator{
		parser:       parser,
		keyResolver:  keyResolver,
		expectedType: strings.TrimSpace(params.ExpectedType),
	}, nil
}

// Authenticate は access token（JWT）を検証し、検証済みの Authn を返します。
// cred が nil の場合は無効トークンとして扱います。
// 検証失敗はすべて ErrJWTAuthenticatorInvalidToken（= apperror.ErrUnauthenticated）へ正規化します。
func (a *authenticator) Authenticate(_ context.Context, cred *authbd.Credential) (*authbd.Authn, error) {
	if cred == nil {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidToken, "credential is nil")
	}

	claims := jwtlib.MapClaims{}
	token, err := a.parser.ParseWithClaims(cred.Token(), claims, a.keyFunc)
	if err != nil || !token.Valid {
		return nil, xerrors.Join(ErrJWTAuthenticatorInvalidToken, err)
	}

	if err := a.verifyType(token); err != nil {
		return nil, err
	}

	subject, err := claims.GetSubject()
	if err != nil || strings.TrimSpace(subject) == "" {
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidToken, "subject missing")
	}

	return authbd.New(subject, authbd.ProviderJWT, extractScopes(claims), map[string]any(claims))
}

// keyFunc は署名検証に用いる公開鍵を keyResolver 経由で返します。
// 解決する鍵は RSA 公開鍵であるため、RSA 系（RS* / PS*）以外の署名方式は
// alg allowlist をすり抜けても鍵種別不一致として拒否します（鍵混同の防御）。
func (a *authenticator) keyFunc(token *jwtlib.Token) (any, error) {
	switch token.Method.(type) {
	case *jwtlib.SigningMethodRSA, *jwtlib.SigningMethodRSAPSS:
		return a.keyResolver(token)
	default:
		return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidToken, "unexpected signing method type")
	}
}

// verifyType は expectedType が設定されている場合に typ ヘッダを検証し、ID Token 等の誤用を拒否します。
func (a *authenticator) verifyType(token *jwtlib.Token) error {
	if a.expectedType == "" {
		return nil
	}
	typ, _ := token.Header["typ"].(string)
	if !strings.EqualFold(typ, a.expectedType) {
		return xerrors.Wrap(ErrJWTAuthenticatorInvalidToken, "unexpected token type")
	}
	return nil
}

// extractScopes は OAuth2 標準 scope クレーム（スペース区切り文字列）を []string に分割します。
// scope クレームが無い場合、文字列型でない場合、または空の場合は nil を返します。
func extractScopes(claims jwtlib.MapClaims) []string {
	raw, ok := claims[scopeClaim].(string)
	if !ok {
		return nil
	}
	return strings.Fields(raw)
}
