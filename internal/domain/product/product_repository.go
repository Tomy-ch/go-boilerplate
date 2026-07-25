//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package product

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// ListParams は、公開商品一覧取得の絞り込み・並び順・keyset 境界を表すクエリ条件です。
// keyset 境界は (AfterPublishedAt, AfterID) の組で表し、先頭ページは両方 nil、継続ページは両方が
// 直前ページ末尾行の値になります（不透明カーソルの符号化・復号は usecase 層の責務です）。
type ListParams struct {
	// Limit は、取得件数の上限です。
	Limit int32
	// Ascending は、公開日時の昇順で取得する場合に true、降順の場合に false です。
	Ascending bool
	// CategoryID は、商品カテゴリ ID による絞り込みです。nil の場合は絞り込みません。
	CategoryID *uuid.UUID
	// StatusID は、商品ステータス ID による絞り込みです。nil の場合は絞り込みません。
	StatusID *uuid.UUID
	// Keyword は、商品名・説明への部分一致検索キーワードです。nil の場合は絞り込みません。
	Keyword *string
	// AfterPublishedAt は、keyset 境界となる公開日時です。先頭ページでは nil です。
	AfterPublishedAt *time.Time
	// AfterID は、keyset 境界となる商品 ID です。先頭ページでは nil です。
	AfterID *uuid.UUID
}

// Repository は、商品の永続化操作を定義するドメインリポジトリインターフェースです。
type Repository interface {
	// FindPublishedList は、公開済み（published_at が非 NULL）の商品を keyset ページネーションで取得します。
	// 並び順は (published_at, id) で、params.Ascending により昇順／降順を切り替えます。
	// params.CategoryID / StatusID / Keyword が指定された場合は該当条件で絞り込みます。
	FindPublishedList(ctx context.Context, params ListParams) (Products, error)
	// FindPublishedByID は、ID から公開中（published_at が非 NULL）の単一商品を取得します。
	// 未存在・非公開のいずれも NotFound を返します（未ログイン経路への存在秘匿）。
	FindPublishedByID(ctx context.Context, id uuid.UUID) (*Product, error)
	// Create は、商品を新規登録します。
	Create(ctx context.Context, p *Product) error
}
