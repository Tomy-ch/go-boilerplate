package inquiry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	domaininquiry "go-boilerplate/internal/domain/inquiry"
	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/usecase/boundary/authz"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

// newTestCursor は、指定の位置を指す不透明カーソルを組み立てます。
func newTestCursor(t *testing.T, updatedAt time.Time, id uuid.UUID, first int) *paging.Cursor {
	t.Helper()
	encoded := paging.EncodeCursor(updatedAt.Format(time.RFC3339Nano), id.String())
	cursor, err := paging.NewCursor(&encoded, ptr.To(first))
	require.NoError(t, err)
	return cursor
}

// newFirstPageCursor は、先頭ページを指すカーソルを組み立てます。
func newFirstPageCursor(t *testing.T, first int) *paging.Cursor {
	t.Helper()
	cursor, err := paging.NewCursor(nil, ptr.To(first))
	require.NoError(t, err)
	return cursor
}

func Test_usecase_ListInquiries(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭ページを更新日時の新しい順で返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), authz.ActionInquiryList, gomock.Any()).Return(nil)
			d.repo.EXPECT().ListForOperator(gomock.Any(), gomock.Any()).
				Return([]*domaininquiry.Inquiry{i}, nil)

			view, err := u.ListInquiries(context.Background(), newTestAuthn(t), ListInquiriesParams{
				Cursor: newFirstPageCursor(t, 10),
			})

			require.NoError(t, err)
			require.Len(t, view.Items, 1)
			assert.Equal(t, i.ID(), view.Items[0].ID)
			assert.Nil(t, view.NextCursor)
		})

		t.Run("上限を超えたら切り詰めて次ページのカーソルを返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			first := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			second := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user2"))

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.repo.EXPECT().ListForOperator(gomock.Any(), gomock.Any()).
				Return([]*domaininquiry.Inquiry{first, second}, nil)

			view, err := u.ListInquiries(context.Background(), newTestAuthn(t), ListInquiriesParams{
				Cursor: newFirstPageCursor(t, 1),
			})

			require.NoError(t, err)
			assert.Len(t, view.Items, 1)
			require.NotNil(t, view.NextCursor)
			assert.NotEmpty(t, *view.NextCursor)
		})

		t.Run("カーソルを渡すと境界を解いてリポジトリへ伝える", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			boundaryID := uuidtestkit.NewTestFromSalt(t, "boundary")

			var captured domaininquiry.ListParams
			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.repo.EXPECT().ListForOperator(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params domaininquiry.ListParams) ([]*domaininquiry.Inquiry, error) {
					captured = params
					return nil, nil
				},
			)

			_, err := u.ListInquiries(context.Background(), newTestAuthn(t), ListInquiriesParams{
				Cursor: newTestCursor(t, baseTime, boundaryID, 10),
			})

			require.NoError(t, err)
			require.NotNil(t, captured.Cursor)
			assert.Equal(t, boundaryID, captured.Cursor.ID)
			assert.True(t, baseTime.Equal(captured.Cursor.UpdatedAt))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者でなければ読まない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(apperror.ErrPermissionDenied)

			_, err := u.ListInquiries(context.Background(), newTestAuthn(t), ListInquiriesParams{
				Cursor: newFirstPageCursor(t, 10),
			})

			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("読み出しの失敗をそのまま返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			wantErr := xerrors.New("list failed")

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.repo.EXPECT().ListForOperator(gomock.Any(), gomock.Any()).Return(nil, wantErr)

			_, err := u.ListInquiries(context.Background(), newTestAuthn(t), ListInquiriesParams{
				Cursor: newFirstPageCursor(t, 10),
			})

			require.ErrorIs(t, err, wantErr)
		})
	})
}

func Test_usecase_GetInquiryHistory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定した問い合わせの履歴を返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), authz.ActionInquiryReadAll, gomock.Any()).Return(nil)
			d.repo.EXPECT().FindByID(gomock.Any(), i.ID()).Return(i, nil)
			d.sequences.EXPECT().Current(gomock.Any(), gomock.Any()).Return(rt.Sequence(1), true, nil)
			d.messages.EXPECT().ListByInquiry(gomock.Any(), i.ID(), gomock.Any()).
				Return([]*domainmessage.Message{newTestMessage(t, i.ID(), domainmessage.AuthorKindOperator, 1)}, nil)

			view, err := u.GetInquiryHistory(context.Background(), newTestAuthn(t), OperatorHistoryParams{
				InquiryID: i.ID(),
			})

			require.NoError(t, err)
			assert.Len(t, view.Messages, 1)
			assert.Equal(t, int64(1), view.StreamCursor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しない問い合わせはNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrNotFound)

			_, err := u.GetInquiryHistory(context.Background(), newTestAuthn(t), OperatorHistoryParams{
				InquiryID: uuidtestkit.NewTestFromSalt(t, "missing"),
			})

			require.ErrorIs(t, err, apperror.ErrNotFound)
		})

		t.Run("管理者でなければ読まない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(apperror.ErrPermissionDenied)

			_, err := u.GetInquiryHistory(context.Background(), newTestAuthn(t), OperatorHistoryParams{})

			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})
	})
}

func Test_usecase_Reply(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("送り手を回答者として追加する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))
			operatorID := uuidtestkit.NewTestFromSalt(t, "operator")

			var created *domainmessage.Message
			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), authz.ActionInquiryReply, gomock.Any()).Return(nil)
			d.repo.EXPECT().FindByID(gomock.Any(), i.ID()).Return(i, nil)
			d.sequences.EXPECT().Allocate(gomock.Any(), gomock.Any()).Return(rt.Sequence(2), nil).Times(2)
			d.messages.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, m *domainmessage.Message) error { created = m; return nil },
			)
			d.repo.EXPECT().Touch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.messages.EXPECT().ListByInquiry(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]*domainmessage.Message{newTestMessage(t, i.ID(), domainmessage.AuthorKindOperator, 2)}, nil)
			d.emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil).Times(2)

			view, err := u.Reply(context.Background(), newTestAuthn(t), ReplyParams{
				InquiryID: i.ID(), OperatorID: operatorID, Body: "確認します",
			})

			require.NoError(t, err)
			assert.Equal(t, "operator", view.AuthorKind)
			require.NotNil(t, created)
			assert.Equal(t, operatorID, created.Author().SubjectID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者でなければ追加しない", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(apperror.ErrPermissionDenied)

			_, err := u.Reply(context.Background(), newTestAuthn(t), ReplyParams{Body: "確認します"})

			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("回答者が未設定なら送り手を組めずに失敗する", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			_, err := u.Reply(context.Background(), newTestAuthn(t), ReplyParams{Body: "確認します"})

			require.ErrorIs(t, err, domainmessage.ErrInvalidAuthorSubject)
		})
	})
}

func Test_usecase_authorize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("所有者を持たないリソースとして問い合わせる", func(t *testing.T) {
			t.Parallel()
			u, d := newTestUsecase(t)

			var captured *authz.Resource
			d.authz.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ any, _ authz.Action, resource *authz.Resource) error {
					captured = resource
					return nil
				},
			)

			require.NoError(t, u.authorize(context.Background(), newTestAuthn(t), authz.ActionInquiryList))
			require.NotNil(t, captured)
			assert.Nil(t, captured.OwnerID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体が無ければ未認証を返す", func(t *testing.T) {
			t.Parallel()
			u, _ := newTestUsecase(t)

			err := u.authorize(context.Background(), nil, authz.ActionInquiryList)

			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})
	})
}

func Test_decodeInquiryCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キー列を境界へ解く", func(t *testing.T) {
			t.Parallel()
			id := uuidtestkit.NewTestFromSalt(t, "boundary")

			got, err := decodeInquiryCursor(newTestCursor(t, baseTime, id, 10))

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, id, got.ID)
			assert.True(t, baseTime.Equal(got.UpdatedAt))
		})

		t.Run("先頭ページでは境界なしを返す", func(t *testing.T) {
			t.Parallel()

			got, err := decodeInquiryCursor(newFirstPageCursor(t, 10))

			require.NoError(t, err)
			assert.Nil(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーの個数が合わなければ不正引数を返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor("only-one-key")
			cursor, err := paging.NewCursor(&encoded, ptr.To(10))
			require.NoError(t, err)

			_, err = decodeInquiryCursor(cursor)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("更新日時が解釈できなければ不正引数を返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor("not-a-time", uuidtestkit.NewTestFromSalt(t, "boundary").String())
			cursor, err := paging.NewCursor(&encoded, ptr.To(10))
			require.NoError(t, err)

			_, err = decodeInquiryCursor(cursor)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("IDが解釈できなければ不正引数を返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(baseTime.Format(time.RFC3339Nano), "not-a-uuid")
			cursor, err := paging.NewCursor(&encoded, ptr.To(10))
			require.NoError(t, err)

			_, err = decodeInquiryCursor(cursor)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_encodeInquiryCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("末尾の更新日時とIDから解き直せるカーソルを作る", func(t *testing.T) {
			t.Parallel()
			i := newTestInquiry(t, uuidtestkit.NewTestFromSalt(t, "user"))

			encoded := encodeInquiryCursor(i)

			cursor, err := paging.NewCursor(&encoded, ptr.To(10))
			require.NoError(t, err)
			decoded, err := decodeInquiryCursor(cursor)
			require.NoError(t, err)
			require.NotNil(t, decoded)
			assert.Equal(t, i.ID(), decoded.ID)
			assert.True(t, i.UpdatedAt().Equal(decoded.UpdatedAt))
		})
	})
}
