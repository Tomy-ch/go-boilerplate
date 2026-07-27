package product

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/kernel/money"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/patch"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// UpdateProductParams は、商品部分更新の入力パラメータです。
// nil のポインタは未指定（現在値を据え置く）を表し、クリアも許容するフィールドは patch.Field で 3 状態を表します。
type UpdateProductParams struct {
	// Version は、更新対象を読み込んだ時点の楽観ロックのバージョンです。
	Version int
	// Name は、商品名です。nil の場合は現在値を据え置きます。
	Name *string
	// Price は、十進文字列で表した価格です。nil の場合は現在値を据え置きます。
	Price *string
	// Quantity は、在庫数です。nil の場合は現在値を据え置きます。
	Quantity *int
	// CategoryID は、商品カテゴリ ID です。nil の場合は現在値を据え置きます。
	CategoryID *uuid.UUID
	// StatusID は、商品ステータス ID です。nil の場合は現在値を据え置きます。
	StatusID *uuid.UUID
	// Description は、商品説明です。null 指定でクリアします。
	Description patch.Field[string]
	// StockWarningThreshold は、在庫警告閾値です。null 指定でクリアします。
	StockWarningThreshold patch.Field[int]
	// PublishedAt は、公開日時です。null 指定でクリア（未公開へ戻す）します。
	PublishedAt patch.Field[time.Time]
	// ImagePath は、画像パスです。null 指定でクリアします。
	ImagePath patch.Field[string]
}

// UpdateProduct は、admin 認可のうえ、送られたフィールドのみを反映して商品を部分更新します。
// 読み込みから更新までを 1 つのトランザクションで行い、読み込み時点のバージョンを条件に更新することで
// 並行編集による上書き（lost update）を防ぎます。バージョンが一致しない場合は 409 を返します。
// この 409 は tx.Manager が透過的にリトライする一時障害（serialization_failure）とは異なり、
// 同じ内容の再送では解消しません。クライアントは最新を取得し直したうえでやり直す必要があります。
func (u *usecase) UpdateProduct(
	ctx context.Context, authn *auth.Authn, id uuid.UUID, params UpdateProductParams,
) (ProductView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return ProductView{}, apperror.ErrUnauthenticated
	}
	if err := u.authorizer.Authorize(ctx, authn, authz.ActionProductUpdate, authz.NewResource("product", nil)); err != nil {
		return ProductView{}, err
	}

	price, err := parseOptionalPrice(params.Price)
	if err != nil {
		return ProductView{}, err
	}

	var entity *product.Product
	var updatedVersion int
	err = u.txm.Do(ctx, func(ctx context.Context) error {
		loaded, err := u.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		entity = loaded

		if err = entity.EnsureVersion(params.Version); err != nil {
			return err
		}

		statusRef, categoryRef, err := u.resolveUpdatedRefs(ctx, entity, params)
		if err != nil {
			return err
		}

		if err = entity.Update(product.Attributes{
			Name:                  ptr.Deref(params.Name, entity.Name()),
			Description:           params.Description.Resolve(entity.Description()),
			Price:                 ptr.Deref(price, entity.Price()),
			Quantity:              ptr.Deref(params.Quantity, entity.Quantity()),
			StockWarningThreshold: params.StockWarningThreshold.Resolve(entity.StockWarningThreshold()),
			Status:                statusRef,
			Category:              categoryRef,
			PublishedAt:           params.PublishedAt.Resolve(entity.PublishedAt()),
			ImagePath:             params.ImagePath.Resolve(entity.ImagePath()),
		}); err != nil {
			return err
		}

		updatedVersion, err = u.repo.Update(ctx, entity)
		return err
	})
	if err != nil {
		return ProductView{}, err
	}

	view := toProductView(entity)
	// バージョンの採番は永続化時に DB が行うため、エンティティが保持する読み込み時点の値を採番後の値で置き換えます。
	view.Version = updatedVersion

	return view, nil
}

// resolveUpdatedRefs は、部分更新後の商品ステータス参照・カテゴリ参照を解決します。
// 参照はペアで扱い、両方とも未指定の場合のみ現在値を据え置いてマスタ問い合わせを行いません。
// いずれかが指定された場合は、未指定側も現在の ID でマスタと突合し、参照整合を再確認します。
func (u *usecase) resolveUpdatedRefs(
	ctx context.Context, entity *product.Product, params UpdateProductParams,
) (product.StatusRef, product.CategoryRef, error) {
	if params.StatusID == nil && params.CategoryID == nil {
		return entity.Status(), entity.Category(), nil
	}

	statusRef, categoryRef, err := u.resolveRefs(
		ctx,
		ptr.Deref(params.StatusID, entity.Status().ID()),
		ptr.Deref(params.CategoryID, entity.Category().ID()),
	)
	if err != nil {
		return product.StatusRef{}, product.CategoryRef{}, err
	}

	return statusRef, categoryRef, nil
}

// parseOptionalPrice は、指定された場合のみ十進文字列を価格へ解釈します。
// 非数値は入力エラー（400）、負値は業務不変条件違反（422）として扱います。
func parseOptionalPrice(v *string) (*money.Price, error) {
	if v == nil {
		return nil, nil //nolint:nilnil // 未指定は「価格の指定なし」を表すため nil, nil が妥当です
	}

	parsed, err := decimal.Parse(*v)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "price is not a valid decimal: "+err.Error())
	}
	price, err := money.NewPrice(parsed)
	if err != nil {
		return nil, err
	}

	return &price, nil
}
