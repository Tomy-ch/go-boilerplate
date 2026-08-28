// Package realtimesecret は、Realtime Delivery の ticket 生値（realtime.SecretGenerator）を OS の暗号論的
// 乱数源から生成する実装を提供します。feature 側の token 境界とは独立に持つのは、sample feature を削除
// しても Realtime Delivery が成り立つためです。
package realtimesecret

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// secretBytes は、1 ticket あたりの乱数バイト数です。256 bit は総当たりで言い当てることが
// 現実的でない水準で、設計正本（ADR-0074）が ticket に要求する幅です。
const secretBytes = 32

// cryptoGenerator は、乱数源から読み出して生値を組み立てます。生成元は New が固定します。
type cryptoGenerator struct {
	source io.Reader
}

// New は、realtime.SecretGenerator の実装を生成します。
func New() realtime.SecretGenerator {
	return &cryptoGenerator{source: rand.Reader}
}

// Generate は、暗号論的に安全な乱数から新しい生値を返します。base64url（パディング無し）なので
// query parameter にそのまま載せられます。必要な長さを読み出せなければエラーを返します —
// 短い値で妥協すると推測できないという保証だけが静かに失われるためです。
func (g *cryptoGenerator) Generate() (string, error) {
	b := make([]byte, secretBytes)
	if _, err := io.ReadFull(g.source, b); err != nil {
		return "", xerrors.Wrap(err, "failed to read random bytes")
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
