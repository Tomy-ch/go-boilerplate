//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package user

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
	"go-boilerplate/pkg/xerrors"
)

const (
	// withdrawalArchiveKeyPrefix は、退会証跡オブジェクトキーの接頭辞です。
	withdrawalArchiveKeyPrefix = "withdrawals/"
	// withdrawalArchiveKeySuffix は、退会証跡オブジェクトキーの拡張子です。証跡はイベント payload そのものです。
	withdrawalArchiveKeySuffix = ".json"
	// withdrawalArchiveContentType は、退会証跡オブジェクトの Content-Type です。
	withdrawalArchiveContentType = "application/json"
)

// ArchiveWithdrawalParams は、退会証跡の保存入力です。
type ArchiveWithdrawalParams struct {
	// UserID は、退会したユーザーの ID です。保存先キーの決定に使います。
	UserID string
	// Payload は、保存する退会イベントの本文です。加工せずそのまま保存します。
	Payload []byte
}

// ArchiveUsecase は、退会の証跡をオブジェクトストレージへ保存するユースケースを定義します。
type ArchiveUsecase interface {
	// ArchiveWithdrawal は、退会証跡を UserID から決まるキーへ保存し、保存先のパスを返します。
	// キーが入力だけで決まり本文も加工しないため、同じ入力での再実行は同じ結果に収束します
	// （at-least-once 配信で複数回実行されうるため）。
	// UserID または Payload が空の場合は apperror.ErrValidation を返します。
	ArchiveWithdrawal(ctx context.Context, params ArchiveWithdrawalParams) (string, error)
}

// archiveUsecase は、退会証跡の保存ユースケースを提供します。
type archiveUsecase struct {
	tracer  observability.LayerTracer
	storage objectstorage.Storage
}

// NewArchive は、退会証跡の保存ユースケースを初期化します。
func NewArchive(tf observability.TracerFactory, storage objectstorage.Storage) ArchiveUsecase {
	return &archiveUsecase{
		tracer:  tf.Usecase(),
		storage: storage,
	}
}

func (u *archiveUsecase) ArchiveWithdrawal(ctx context.Context, params ArchiveWithdrawalParams) (string, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if params.UserID == "" {
		return "", xerrors.Wrap(apperror.ErrValidation, "user id must not be empty")
	}
	if len(params.Payload) == 0 {
		return "", xerrors.Wrap(apperror.ErrValidation, "payload must not be empty")
	}

	path, err := u.storage.Put(ctx, objectstorage.PutObject{
		Key:         withdrawalArchiveKeyPrefix + params.UserID + withdrawalArchiveKeySuffix,
		Body:        params.Payload,
		ContentType: withdrawalArchiveContentType,
		// 同一キーを上書きしうるため配信キャッシュを指定しません。商品画像がキー不変を根拠に
		// immutable を付けているのと対になります。
		CacheControl: "",
	})
	if err != nil {
		return "", err
	}

	return string(path), nil
}
