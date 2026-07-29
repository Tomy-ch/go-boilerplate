//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package objectstorage は、オブジェクトストレージへの中立な保存境界（Storage）と
// vendor 非依存の入力表現を定義します。usecase 層はこの境界にのみ依存し、
// S3 互換 adapter（infrastructure 層）が実装します。bucket / region / etag 等の
// vendor 語彙は境界に露出しません。
package objectstorage

import "context"

// Storage は、オブジェクトを指定キー配下へ保存する中立な境界です。
type Storage interface {
	// Put は、obj をそのキー配下へ保存し、保存されたパス（オブジェクトキー）を返します。
	// 返された Path は上位が永続化する対象であり、表示 URL は上位（フロント）が
	// 配信ベース URL と組み合わせて別途組み立てます。
	// 失敗時は apperror sentinel（ErrUnavailable 等）を返します。
	Put(ctx context.Context, obj PutObject) (Path, error)
}

// PutObject は、保存対象オブジェクトの入力 DTO です。
type PutObject struct {
	// Key は、保存先のオブジェクトキー（例 "products/{uuid}.png"）です。呼び出し側が採番します。
	Key string
	// Body は、保存するバイト列です。長さがそのままオブジェクトサイズになります。
	Body []byte
	// ContentType は、オブジェクトの MIME タイプ（例 "image/png"）です。
	ContentType string
	// CacheControl は、配信時に返す Cache-Control（例 "public, max-age=31536000, immutable"）です。
	// 空文字なら設定しません。キャッシュ可否はキーの不変性に依存するため、呼び出し側が決めます。
	CacheControl string
}

// Path は、保存されたオブジェクトのパス（キー）です。
type Path string
