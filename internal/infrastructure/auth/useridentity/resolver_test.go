package useridentity

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuerMock = "mock"
	// johnUserID は JWT/mock 両 issuer に登録された未削除ユーザー（John Doe）の内部 ID。
	johnUserID = "550e8400-e29b-41d4-a716-446655440000"
)

// seed が投入する JWT identity の issuer は環境の AUTH_ISSUER で、worktree の DB スロットでポートがずれる。
// テスト用 DB を seed するのも値を渡すのも make のため、DB を使う本テストは make test / make test-cached
// 経由で実行する（素の go test は make の渡す値を受け取らない）。

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("resolver のインスタンスが生成される", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)
			tf := observability.NewNoopTracerFactory(t)
			expected := &resolver{
				db:     testDB,
				tracer: tf.Infra(),
			}
			actual := New(testDB, tf)
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_resolver_Resolve(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &resolver{
		tracer: lt,
		db:     testDB,
	}

	newAuthn := func(t *testing.T, issuer, subject string) *authbd.Authn {
		t.Helper()
		authn, err := authbd.New(subject, issuer, nil, nil)
		require.NoError(t, err)
		return authn
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWT issuer と subject に対応する未削除ユーザーの UserID を解決する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				resolved, err := repo.Resolve(ctx, newAuthn(t, config.ResolvedAuthIssuer(t), "user-john-doe"))
				require.NoError(t, err)
				require.True(t, resolved.HasUserID())
				userID, err := resolved.UserID()
				require.NoError(t, err)
				assert.Equal(t, johnUserID, userID.String())
			})
		})

		t.Run("mock issuer と subject でも同じユーザーの UserID を解決する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				resolved, err := repo.Resolve(ctx, newAuthn(t, testIssuerMock, johnUserID))
				require.NoError(t, err)
				require.True(t, resolved.HasUserID())
				userID, err := resolved.UserID()
				require.NoError(t, err)
				assert.Equal(t, johnUserID, userID.String())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("削除済みユーザーの identity は ErrUserUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				resolved, err := repo.Resolve(ctx, newAuthn(t, config.ResolvedAuthIssuer(t), "user-charlie-davis"))
				assert.Nil(t, resolved)
				require.ErrorIs(t, err, authbd.ErrUserUnavailable)
			})
		})

		t.Run("未登録の subject は ErrIdentityNotFound を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				resolved, err := repo.Resolve(ctx, newAuthn(t, config.ResolvedAuthIssuer(t), "user-nonexistent"))
				assert.Nil(t, resolved)
				require.ErrorIs(t, err, authbd.ErrIdentityNotFound)
			})
		})

		t.Run("issuer が一致しない場合は ErrIdentityNotFound を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				resolved, err := repo.Resolve(ctx, newAuthn(t, "https://evil.example.com", "user-john-doe"))
				assert.Nil(t, resolved)
				require.ErrorIs(t, err, authbd.ErrIdentityNotFound)
			})
		})

		t.Run("キャンセル済みコンテキストでは ErrCanceled へ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			resolved, err := repo.Resolve(ctx, newAuthn(t, config.ResolvedAuthIssuer(t), "user-john-doe"))
			assert.Nil(t, resolved)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
