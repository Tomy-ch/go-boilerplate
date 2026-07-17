package apperror

import (
	"fmt"
	"io"

	"go-boilerplate/pkg/xerrors"
)

// Meta は、エラー発生箇所がレスポンス向けに動的に付与できるメタ情報です。
// [NewMeta] で構築します。空の項目は transport 層(controller)の既定値にフォールバックします。
// apperror センチネルによる分類(HTTP ステータス解決)には一切影響しません。
type Meta struct {
	code    string
	message string
	details []string
}

// MetaError は、Meta を運ぶラッパーエラーです。
// Unwrap により元エラーのセンチネル分類([xerrors.Is] / [IsAppError])を保持します。
type MetaError struct {
	meta Meta
	err  error
}

// NewMeta は、機械可読なエラーコード code と詳細識別子 details から Meta を構築します。
// code が空の場合はステータス由来の既定コードが使われます。details はレスポンスに
// そのまま公開されるため、識別子(例: 不正フィールド名)のみを入れ、理由文や入力値
// そのものを入れてはいけません。details は防御的コピーされます。
// 利用者向け文言は持ちません(文言の正は controller のカタログ。上書きは [Meta.WithMessage])。
func NewMeta(code string, details ...string) Meta {
	return Meta{code: code, details: append([]string(nil), details...)}
}

// WithMessage は、利用者向け文言を上書きした Meta のコピーを返します。
// 文言の正は controller のカタログにあるため、呼び出しは controller 層に限ります
// (domain / usecase では使わない)。
func (m Meta) WithMessage(message string) Meta {
	m.message = message
	return m
}

// Code は機械可読なエラーコードを返します。
func (m Meta) Code() string {
	return m.code
}

// Message は利用者向け文言を返します。
func (m Meta) Message() string {
	return m.message
}

// Details は詳細識別子の防御的コピーを返します。
func (m Meta) Details() []string {
	if m.details == nil {
		return nil
	}
	return append([]string(nil), m.details...)
}

// WithMeta は、err に meta を付与したエラーを返します。err が nil の場合は nil を返します。
func WithMeta(err error, meta Meta) error {
	if err == nil {
		return nil
	}
	return &MetaError{meta: meta, err: err}
}

// WithDetails は、err に details のみを付与したエラーを返します。[WithMeta] の糖衣です。
func WithDetails(err error, details ...string) error {
	return WithMeta(err, NewMeta("", details...))
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

// Meta は付与されたメタ情報を返します。フィールドは非公開のため、取り出した値経由で
// 内部状態を変更することはできません([Meta.Details] は防御的コピーを返します)。
func (e *MetaError) Meta() Meta {
	return e.meta
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
