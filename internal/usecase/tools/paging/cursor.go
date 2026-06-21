package paging

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// Cursor は、キーセット（cursor）ページネーションのリクエストを表す値オブジェクトです。
//
//	offset 版（Page）がページ番号を起点に LIMIT/OFFSET へ変換するのに対し、
//	Cursor は「直前ページ末尾行のソートキー」を不透明トークンとして受け取り、
//	keyset 比較（例: WHERE (created_at, id) < (:k0, :k1)）で次ページを取得するための情報を保持します。
//
//	keys はソートキーのタプルを文字列化したものです。型の解釈（RFC3339 → time、UUID 文字列 → uuid 等）は
//	Cursor を受け取る呼び出し元（usecase 等）の責務であり、本パッケージは輸送（エンコード/デコード）と件数ポリシーのみを担います。
type Cursor struct {
	limit int
	keys  []string
}

// NewCursor は、不透明カーソル文字列（after）と取得件数（first）から Cursor を生成します。
//
//	after が nil または空文字の場合は先頭ページとして扱い、keys は空になります。
//	after の形式が不正な場合は apperror.ErrInvalidArgument を返します。
//	first[limit] の補完・クランプ規約は offset 版（NewPageFrom1Based）と共通で、
//	0以下または nil の場合は defaultPerPage、maxPerPage を超える場合は maxPerPage を使用します。
//	keyset はページ番号を持たないため、offset 版のような最大ページ数エラーは発生しません。
func NewCursor(after *string, first *int) (*Cursor, error) {
	limit := defaultPerPage
	if first != nil && *first > 0 {
		limit = *first
	}
	if limit > maxPerPage {
		limit = maxPerPage
	}

	keys, err := decodeCursor(after)
	if err != nil {
		return nil, err
	}

	return &Cursor{
		limit: limit,
		keys:  keys,
	}, nil
}

// Limit は、ページの取得上限を返します。
func (c Cursor) Limit() int { return c.limit }

// Limit32 は、ページの取得上限をint32型で返します。
func (c Cursor) Limit32() int32 {
	limit := min(c.limit, maxPerPage)
	//nolint:gosec // G115: maxPerPage(int32範囲内の定数)でクランプ済みのためオーバーフローしません
	return int32(limit)
}

// HasCursor は、カーソルが指定されている（＝先頭ページではない）場合に true を返します。
func (c Cursor) HasCursor() bool { return len(c.keys) > 0 }

// Keys は、直前ページ末尾行のソートキー（タプル）のコピーを返します。
//
//	先頭ページの場合は空スライスを返します。呼び出し元はこの値を keyset 比較の境界として解釈します。
func (c Cursor) Keys() []string {
	if len(c.keys) == 0 {
		return []string{}
	}
	out := make([]string, len(c.keys))
	copy(out, c.keys)
	return out
}

// EncodeCursor は、ソートキーのタプルを不透明なカーソル文字列へエンコードします。
//
//	クエリ層が「現在ページ末尾行」のソートキー（文字列化済み）を渡して次ページ用カーソルを生成します。
//	キーが1つも渡されない場合は、先頭ページを表す空文字を返します。
func EncodeCursor(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	b, err := json.Marshal(keys)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor は、不透明カーソル文字列をソートキーのタプルへ復号します。
//
//	nil または空文字は先頭ページとみなし、nil を返します。
//	base64/JSON の形式不正、または空タプルの場合は apperror.ErrInvalidArgument を返します。
func decodeCursor(after *string) ([]string, error) {
	if after == nil || *after == "" {
		return nil, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(*after)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, fmt.Sprintf("invalid cursor encoding: %v", err))
	}

	var keys []string
	if err := json.Unmarshal(raw, &keys); err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, fmt.Sprintf("invalid cursor payload: %v", err))
	}
	if len(keys) == 0 {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: empty keyset")
	}

	return keys, nil
}
