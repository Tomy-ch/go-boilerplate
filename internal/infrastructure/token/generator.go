// Package token は、トークン生成のインフラストラクチャ実装を提供します。
package token

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"go-boilerplate/internal/usecase/boundary/token"
	"go-boilerplate/pkg/xerrors"
)

// tokenBytes は、1 トークンあたりの乱数バイト数です。
// 256 ビットは、生成された値を総当たりで言い当てることが現実的でない水準として選んでいます。
const tokenBytes = 32

// cryptoGenerator は、乱数源から読み出してトークンを組み立てます。
// 生成元は New が固定し、差し替える口は公開しません。
type cryptoGenerator struct {
	source io.Reader
}

// New は、token.Generator の実装を生成します。
func New() token.Generator {
	return &cryptoGenerator{source: rand.Reader}
}

// Generate は、暗号論的に安全な乱数から新しいトークン文字列を返します。
// 値は base64url（パディング無し）で表現するため、URL とヘッダにそのまま載せられます。
// 乱数を必要な長さだけ読み出せなかった場合はエラーを返します。短い値で妥協すると、
// 推測できないという保証だけが静かに失われるためです。
func (g *cryptoGenerator) Generate() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := io.ReadFull(g.source, b); err != nil {
		return "", xerrors.Wrap(err, "failed to read random bytes")
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
