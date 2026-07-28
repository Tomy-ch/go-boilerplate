package address_test

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	domainprefecture "go-boilerplate/internal/domain/prefecture"
	mock_prefecture "go-boilerplate/internal/domain/prefecture/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/address"
	boundary "go-boilerplate/internal/usecase/boundary/address"
	mock_address "go-boilerplate/internal/usecase/boundary/address/mock"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newPrefecture(t *testing.T, name string, code int) *domainprefecture.Prefecture {
	t.Helper()
	id, err := uuid.New()
	require.NoError(t, err)
	p, err := domainprefecture.New(id, name, code)
	require.NoError(t, err)
	return p
}

func Test_usecase_LookupByPostalCode(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("県名を解決しprefecture_idを埋めた候補を返す_同一県名はFindByName1回に集約する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_address.NewMockGateway(ctrl)
			repo := mock_prefecture.NewMockRepository(ctrl)
			tokyo := newPrefecture(t, "東京都", 13)

			gw.EXPECT().Lookup(gomock.Any(), "1000001").Return([]*boundary.Candidate{
				{PrefectureName: "東京都", City: "千代田区", Town: "千代田"},
				{PrefectureName: "東京都", City: "千代田区", Town: "大手町"},
			}, nil)
			// 同一県名は重複排除され FindByName は 1 回だけ呼ばれる。
			repo.EXPECT().FindByName(gomock.Any(), "東京都").Return(tokyo, nil).Times(1)

			uc := address.New(gw, repo, observability.NewNoopTracerFactory(t))
			result, err := uc.LookupByPostalCode(context.Background(), "1000001")

			require.NoError(t, err)
			assert.False(t, result.IsFallback)
			require.Len(t, result.Candidates, 2)
			require.NotNil(t, result.Candidates[0].PrefectureID)
			assert.True(t, result.Candidates[0].PrefectureID.Equal(tokyo.ID()))
			assert.Equal(t, "大手町", result.Candidates[1].Town)
		})

		t.Run("異なる県名はそれぞれFindByNameで解決し各候補に対応するIDを割り当てる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_address.NewMockGateway(ctrl)
			repo := mock_prefecture.NewMockRepository(ctrl)
			tokyo := newPrefecture(t, "東京都", 13)
			osaka := newPrefecture(t, "大阪府", 27)

			gw.EXPECT().Lookup(gomock.Any(), "1000001").Return([]*boundary.Candidate{
				{PrefectureName: "東京都", City: "千代田区", Town: "千代田"},
				{PrefectureName: "大阪府", City: "大阪市北区", Town: "梅田"},
			}, nil)
			repo.EXPECT().FindByName(gomock.Any(), "東京都").Return(tokyo, nil).Times(1)
			repo.EXPECT().FindByName(gomock.Any(), "大阪府").Return(osaka, nil).Times(1)

			uc := address.New(gw, repo, observability.NewNoopTracerFactory(t))
			result, err := uc.LookupByPostalCode(context.Background(), "1000001")

			require.NoError(t, err)
			require.Len(t, result.Candidates, 2)
			require.NotNil(t, result.Candidates[0].PrefectureID)
			require.NotNil(t, result.Candidates[1].PrefectureID)
			assert.True(t, result.Candidates[0].PrefectureID.Equal(tokyo.ID()))
			assert.True(t, result.Candidates[1].PrefectureID.Equal(osaka.ID()))
		})

		t.Run("正常応答で候補0件のときは空候補_IsFallback_falseを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_address.NewMockGateway(ctrl)
			repo := mock_prefecture.NewMockRepository(ctrl)

			gw.EXPECT().Lookup(gomock.Any(), "0000000").Return([]*boundary.Candidate{}, nil)

			uc := address.New(gw, repo, observability.NewNoopTracerFactory(t))
			result, err := uc.LookupByPostalCode(context.Background(), "0000000")

			require.NoError(t, err)
			assert.False(t, result.IsFallback)
			assert.Empty(t, result.Candidates)
		})

		t.Run("県名解決不能_NotFoundのときは候補を残しprefecture_idをnilにする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_address.NewMockGateway(ctrl)
			repo := mock_prefecture.NewMockRepository(ctrl)

			gw.EXPECT().Lookup(gomock.Any(), "1000001").Return([]*boundary.Candidate{
				{PrefectureName: "存在しない県", City: "架空市", Town: "架空町"},
			}, nil)
			repo.EXPECT().FindByName(gomock.Any(), "存在しない県").Return(nil, apperror.ErrNotFound)

			uc := address.New(gw, repo, observability.NewNoopTracerFactory(t))
			result, err := uc.LookupByPostalCode(context.Background(), "1000001")

			require.NoError(t, err)
			assert.False(t, result.IsFallback)
			require.Len(t, result.Candidates, 1)
			assert.Nil(t, result.Candidates[0].PrefectureID)
			assert.Equal(t, "架空市", result.Candidates[0].City)
		})

		t.Run("degrade_外部lookup障害時は空候補_IsFallback_trueを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_address.NewMockGateway(ctrl)
			repo := mock_prefecture.NewMockRepository(ctrl)

			gw.EXPECT().Lookup(gomock.Any(), "1000001").Return(nil, apperror.ErrUnavailable)

			uc := address.New(gw, repo, observability.NewNoopTracerFactory(t))
			result, err := uc.LookupByPostalCode(context.Background(), "1000001")

			require.NoError(t, err)
			assert.True(t, result.IsFallback)
			assert.Empty(t, result.Candidates)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("県名解決がNotFound以外のエラーのときはdegradeせず伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_address.NewMockGateway(ctrl)
			repo := mock_prefecture.NewMockRepository(ctrl)
			dbErr := xerrors.New("db connection failed")

			gw.EXPECT().Lookup(gomock.Any(), "1000001").Return([]*boundary.Candidate{
				{PrefectureName: "東京都", City: "千代田区", Town: "千代田"},
			}, nil)
			repo.EXPECT().FindByName(gomock.Any(), "東京都").Return(nil, dbErr)

			uc := address.New(gw, repo, observability.NewNoopTracerFactory(t))
			result, err := uc.LookupByPostalCode(context.Background(), "1000001")

			require.ErrorIs(t, err, dbErr)
			assert.Nil(t, result)
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引数のgatewayと都道府県リポジトリを結線したユースケースを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_address.NewMockGateway(ctrl)
			repo := mock_prefecture.NewMockRepository(ctrl)

			pref := newPrefecture(t, "東京都", 13)
			gw.EXPECT().Lookup(gomock.Any(), "1000001").
				Return([]*boundary.Candidate{{PrefectureName: "東京都", City: "千代田区", Town: "千代田"}}, nil)
			repo.EXPECT().FindByName(gomock.Any(), "東京都").Return(pref, nil)

			uc := address.New(gw, repo, observability.NewNoopTracerFactory(t))
			require.NotNil(t, uc)

			result, err := uc.LookupByPostalCode(context.Background(), "1000001")
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Candidates, 1)
			require.NotNil(t, result.Candidates[0].PrefectureID)
			assert.Equal(t, pref.ID(), *result.Candidates[0].PrefectureID)
			assert.Equal(t, "千代田区", result.Candidates[0].City)
		})
	})
}
