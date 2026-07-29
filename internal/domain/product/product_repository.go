//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package product

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// ListParams は、公開商品一覧取得の絞り込み・並び順・keyset 境界を表すクエリ条件です。
// keyset 境界は (AfterPublishedAt, AfterID) の組で表し、先頭ページは両方 nil、継続ページは両方が
// 直前ページ末尾の値になります（不透明カーソルの符号化・復号は usecase 層の責務です）。
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
	// FindPublishedList は、公開済みの商品を keyset ページネーションで取得します。
	// 並び順は公開日時（同時刻は ID）で、params.Ascending により昇順／降順を切り替えます。
	// params.CategoryID / StatusID / Keyword が指定された場合は該当条件で絞り込みます。
	FindPublishedList(ctx context.Context, params ListParams) (Products, error)
	// FindPublishedByID は、ID から公開中の単一商品を取得します。
	// 未存在・非公開のいずれも NotFound を返します（未ログイン経路への存在秘匿）。
	FindPublishedByID(ctx context.Context, id uuid.UUID) (*Product, error)
	// FindByID は、ID から公開状態を問わない単一商品を取得します。未存在は NotFound を返します。
	// 公開日時の設定そのものを更新対象とするため、FindPublishedByID と異なり未公開商品も返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
	// LockByID は、更新のために ID から公開状態を問わない単一商品を取得します。未存在は NotFound を返します。
	// 同一商品への並行更新は、先行する更新が終わるまで待機したうえで最新の状態を取得します。
	LockByID(ctx context.Context, id uuid.UUID) (*Product, error)
	// Create は、商品を新規登録します。
	Create(ctx context.Context, p *Product) error
	// Update は、p が保持するバージョンを条件に商品を更新し、採番後のバージョンを返します。
	// 読み込み後に他者が更新しておりバージョンが一致しない場合は ErrVersionConflict を返します。
	Update(ctx context.Context, p *Product) (int, error)
	// UpdateStock は、p が保持するバージョンを条件に在庫数を更新し、採番後のバージョンを返します。
	// 読み込み後に他者が更新しておりバージョンが一致しない場合は ErrVersionConflict を返します。
	UpdateStock(ctx context.Context, p *Product) (int, error)
}
