// Package inquirymessage は、問い合わせメッセージリポジトリ（inquirymessage.Repository）の
// RDB 実装を提供します。
package inquirymessage

import (
	"context"

	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、inquirymessage.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) inquirymessage.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// Create は、メッセージを 1 件登録します。
// 問い合わせ内の位置が重複した場合は一意制約違反を Conflict として返します。
func (r *repository) Create(ctx context.Context, m *inquirymessage.Message) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	err := db.CreateInquiryMessage(ctx, &gen.CreateInquiryMessageParams{
		ID:              m.ID(),
		InquiryID:       m.InquiryID(),
		AuthorKind:      m.Author().Kind().String(),
		AuthorSubjectID: m.Author().SubjectID(),
		Body:            m.Body(),
		Sequence:        m.Sequence(),
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// ListByInquiry は、問い合わせのメッセージを位置の昇順で 1 ページ分再構築して返します。
func (r *repository) ListByInquiry(
	ctx context.Context,
	inquiryID uuid.UUID,
	params inquirymessage.HistoryParams,
) ([]*inquirymessage.Message, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	pageSize, err := safecast.IntToInt32(params.Limit)
	if err != nil {
		return nil, err
	}

	rows, err := db.ListInquiryMessages(ctx, &gen.ListInquiryMessagesParams{
		InquiryID:     inquiryID,
		AfterSequence: params.AfterSequence,
		UpToSequence:  params.UpToSequence,
		PageSize:      pageSize,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	messages := make([]*inquirymessage.Message, 0, len(rows))
	for _, row := range rows {
		entity, rerr := reconstruct(row.InquiryMessages)
		if rerr != nil {
			return nil, rerr
		}
		messages = append(messages, entity)
	}
	return messages, nil
}

// reconstruct は、行をドメイン集約へ写します。
func reconstruct(row gen.InquiryMessages) (*inquirymessage.Message, error) {
	kind, err := inquirymessage.NewAuthorKind(row.AuthorKind)
	if err != nil {
		return nil, err
	}
	author, err := inquirymessage.NewAuthor(kind, row.AuthorSubjectID)
	if err != nil {
		return nil, err
	}

	return inquirymessage.Reconstruct(row.ID, inquirymessage.Attributes{
		InquiryID: row.InquiryID,
		Author:    author,
		Body:      row.Body,
		Sequence:  row.Sequence,
		CreatedAt: row.CreatedAt,
	})
}
