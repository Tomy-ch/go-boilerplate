//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package token は、不透明なトークン文字列の生成を抽象化するバウンダリインターフェースを提供します。
package token

// Generator は、推測できないトークン文字列を生成するためのインターフェースです。
type Generator interface {
	// Generate は、新しいトークン文字列を返します。
	// 呼ぶたびに異なる値を返し、生成済みの値から次の値を推測できないことを保証します。
	// 乱数源が利用できない場合はエラーを返します。
	Generate() (string, error)
}
