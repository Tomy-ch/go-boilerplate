//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package objectstorage は、オブジェクトストレージへの中立な保存境界（Storage）と
// vendor 非依存の入出力表現を定義します。usecase 層はこの境界にのみ依存し、
// S3 互換 adapter（infrastructure 層）が実装します。bucket / region / etag /
// continuation token 等の vendor 語彙は境界に露出しません。
package objectstorage

import (
	"context"
	"time"
)

// Storage は、オブジェクトを保存・列挙・削除する中立な境界です。
// 各メソッドは失敗時に apperror sentinel（ErrUnavailable 等）を返します。
type Storage interface {
	// Put は、obj をそのキー配下へ保存し、保存されたパス（オブジェクトキー）を返します。
	// 返された Path は上位が永続化する対象であり、表示 URL は上位（フロント）が
	// 配信ベース URL と組み合わせて別途組み立てます。
	Put(ctx context.Context, obj PutObject) (Path, error)
	// List は、query に一致するオブジェクトを 1 ページ分列挙します。
	// 続きがある場合は ListResult.NextCursor が非空になり、それを次の ListQuery.Cursor に渡すと続きを取得できます。
	List(ctx context.Context, query ListQuery) (ListResult, error)
	// Delete は、keys のオブジェクトをまとめて削除します。keys が空の場合は何もしません。
	// 存在しないキーはエラーにならず、同じ keys で再実行しても結果は変わりません。
	Delete(ctx context.Context, keys []string) error
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

// ListQuery は、オブジェクト列挙の条件です。
type ListQuery struct {
	// Prefix は、列挙対象を絞り込むキーの接頭辞（例 "products/"）です。空なら全件が対象になります。
	Prefix string
	// Cursor は、続きから列挙するための境界です。空なら先頭から列挙します。
	// 直前の ListResult.NextCursor をそのまま渡す想定で、中身は adapter 依存の不透明な文字列です。
	Cursor string
	// Limit は、1 ページで列挙する件数の上限です。0 以下なら adapter の既定値になります。
	Limit int32
}

// ListResult は、1 ページ分の列挙結果です。
type ListResult struct {
	// Objects は、列挙されたオブジェクトです。並び順は保証しません。
	Objects []Object
	// NextCursor は、続きを取得するための境界です。空なら最終ページです。
	NextCursor string
}

// Object は、列挙されたオブジェクトの中立表現です。
type Object struct {
	// Key は、オブジェクトキー（例 "products/{uuid}.png"）です。
	Key string
	// ModifiedAt は、オブジェクトの最終更新時刻です。
	ModifiedAt time.Time
}

// Path は、保存されたオブジェクトのパス（キー）です。
type Path string
