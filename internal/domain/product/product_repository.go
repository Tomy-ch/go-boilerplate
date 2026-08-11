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

// Counts は、商品の登録件数の集計です。
type Counts struct {
	// Total は未公開を含む登録商品の総数、Published はそのうち公開済みの件数です。
	Total     int64
	Published int64
}

// Repository は、商品の永続化操作を定義するドメインリポジトリインターフェースです。
//
//nolint:interfacebloat // 集約の永続化契約は 1 本に保つ（ADR-0029 (lightweight-cqrs)）。呼び出し側ごとに分割すると同一集約の契約が複数箇所へ散る
type Repository interface {
	// FindPublishedList は、公開済みの商品を keyset ページネーションで取得します。
	// 並び順は公開日時（同時刻は ID）で、params.Ascending により昇順／降順を切り替えます。
	// params.CategoryID / StatusID / Keyword が指定された場合は該当条件で絞り込みます。
	FindPublishedList(ctx context.Context, params ListParams) (Products, error)
	// FindAllLowStock は、在庫が在庫警告閾値以下まで減った商品を、在庫の少ない順（同数は ID の昇順）で
	// 最大 limit 件返します。在庫警告閾値が未設定の商品は警告対象を持たないため含みません。
	// 補充の要否は公開状態に依存しないため、未公開の商品も含めます。
	FindAllLowStock(ctx context.Context, limit int32) (Products, error)
	// FindPublishedByID は、ID から公開中の単一商品を取得します。
	// 未存在・非公開のいずれも NotFound を返します（未ログイン経路への存在秘匿）。
	FindPublishedByID(ctx context.Context, id uuid.UUID) (*Product, error)
	// FindByID は、ID から公開状態を問わない単一商品を取得します。未存在は NotFound を返します。
	// 公開日時の設定そのものを更新対象とするため、FindPublishedByID と異なり未公開商品も返します。
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
	// LockByID は、更新のために ID から公開状態を問わない単一商品を取得します。未存在は NotFound を返します。
	// 同一商品への並行更新は、先行する更新が終わるまで待機したうえで最新の状態を取得します。
	LockByID(ctx context.Context, id uuid.UUID) (*Product, error)
	// LockByIDs は、更新のために ID の集合から公開状態を問わない商品群を、ID 昇順にまとめて取得します
	// （順序を固定する理由は ADR-0033 (ordered-pessimistic-row-locks)）。
	// 不存在の ID はロックできず結果に現れないため、要素数は ids より少なくなり得ます
	// （不存在の検証は呼び出し側の責務です）。
	LockByIDs(ctx context.Context, ids []uuid.UUID) (Products, error)
	// Create は、商品を新規登録します。p が保持する画像も併せて登録します。
	Create(ctx context.Context, p *Product) error
	// Update は、p が保持するバージョンを条件に商品を更新し、採番後のバージョンを返します。
	// 画像は対象に含みません（置換は ReplaceImages が担います）。
	// 読み込み後に他者が更新しておりバージョンが一致しない場合は ErrVersionConflict を返します。
	Update(ctx context.Context, p *Product) (int, error)
	// ReplaceImages は、商品が現在参照している画像を p が保持する画像で置き換えます。
	// 置き換え前の画像は論理削除として残り、現在の参照からは外れます。
	//
	// Update が成功した後、同じトランザクションの中で呼び出す必要があります。順序を入れ替えると、
	// バージョン不一致で拒否される更新の画像だけが先に入れ替わります
	// （この順序が保護になる仕組みは docs/spec/product/usecase.md の UpdateProduct）。
	ReplaceImages(ctx context.Context, p *Product) error
	// UpdateStock は、p が保持するバージョンを条件に在庫数を更新し、採番後のバージョンを返します。
	// 読み込み後に他者が更新しておりバージョンが一致しない場合は ErrVersionConflict を返します。
	UpdateStock(ctx context.Context, p *Product) (int, error)
	// Count は、登録商品の総数と、そのうち公開済みの件数を返します。商品が 1 件もない場合はゼロ値を返します。
	Count(ctx context.Context) (Counts, error)
	// FilterExistingImagePaths は、paths のうち、いずれかの商品が現在の画像として参照しているものを返します。
	// 重複は取り除き、順序は保証しません。paths が空の場合は空を返します。
	// 返らなかったパスは、どの商品からも参照されていないことを意味します。
	// 置き換えで論理削除された画像は現在の参照ではないため、参照元として数えません。
	FilterExistingImagePaths(ctx context.Context, paths []string) ([]string, error)
}
