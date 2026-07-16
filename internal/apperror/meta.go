package apperror

import (
	"fmt"
	"io"

	"go-boilerplate/pkg/xerrors"
)

// Meta は、エラー発生箇所がレスポンス向けに動的に付与できるメタ情報です。
// 全フィールド任意で、空の項目は transport 層(controller)の既定値にフォールバックします。
// apperror センチネルによる分類(HTTP ステータス解決)には一切影響しません。
type Meta struct {
	// Code は機械可読なエラーコードです。空の場合はステータス由来の既定コードが使われます。
	Code string
	// Message は利用者向け文言です。文言の正は controller のカタログにあるため、
	// domain / usecase では原則空のままにします。
	Message string
	// Details はプロトコル中立な詳細識別子(例: 不正フィールド名)です。
	// レスポンスにそのまま公開されるため、理由文や値そのものを入れてはいけません。
	Details []string
}

// MetaError は、Meta を運ぶラッパーエラーです。
// Unwrap により元エラーのセンチネル分類([xerrors.Is] / [IsAppError])を保持します。
type MetaError struct {
	meta Meta
	err  error
}

// WithMeta は、err に meta を付与したエラーを返します。err が nil の場合は nil を返します。
// Details は防御的コピーするため、呼び出し元は渡したスライスを以後安全に変更できます。
func WithMeta(err error, meta Meta) error {
	if err == nil {
		return nil
	}
	if meta.Details != nil {
		meta.Details = append([]string(nil), meta.Details...)
	}
	return &MetaError{meta: meta, err: err}
}

// WithDetails は、err に details のみを付与したエラーを返します。[WithMeta] の糖衣です。
func WithDetails(err error, details ...string) error {
	return WithMeta(err, Meta{Details: details})
}

// MetaFrom は、err のチェーンから最も外側の [MetaError] の Meta を抽出します。
// 多重に付与されている場合は外側が勝ちます。見つからない場合は ok=false を返します。
func MetaFrom(err error) (Meta, bool) {
	var metaErr *MetaError
	if !xerrors.As(err, &metaErr) {
		return Meta{}, false
	}
	return metaErr.Meta(), true
}

// Error は error インターフェースを実装します。メッセージは元エラーのまま変えません。
func (e *MetaError) Error() string {
	return e.err.Error()
}

// Unwrap は元エラーを返します。シグネチャは errors.Is / errors.As がチェーンを辿るための
// 標準ライブラリ契約であり、改名するとセンチネル分類がこのラッパーを貫通できなくなります。
func (e *MetaError) Unwrap() error {
	return e.err
}

// Meta は付与されたメタ情報を返します。Details は防御的コピーです。
func (e *MetaError) Meta() Meta {
	meta := e.meta
	if meta.Details != nil {
		meta.Details = append([]string(nil), meta.Details...)
	}
	return meta
}

// Format は fmt.Formatter を実装し、元エラーの表現(スタックトレース含む)へ委譲します。
// これがないと %+v(ログ出力)で元エラーのスタックトレースが失われます。
func (e *MetaError) Format(s fmt.State, verb rune) {
	if formatter, ok := e.err.(fmt.Formatter); ok {
		formatter.Format(s, verb)
		return
	}
	//nolint:errcheck,gosec // fmt.State への書き込み失敗はハンドリング不能
	io.WriteString(s, e.Error())
}
