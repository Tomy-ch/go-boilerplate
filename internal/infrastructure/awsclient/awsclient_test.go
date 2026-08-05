package awsclient_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/awsclient"
	"go-boilerplate/internal/observability"
)

// isolateCredentialChain は、SDK 既定の credential chain が実行マシンの設定を拾わないようにします。
// 共有プロファイル・IMDS・コンテナ資格情報のいずれも見に行かせないことで、chain 側の解決結果を
// テストが完全に決められるようにします。
func isolateCredentialChain(t *testing.T) {
	t.Helper()

	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", t.TempDir()+"/credentials")
	t.Setenv("AWS_CONFIG_FILE", t.TempDir()+"/config")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "")
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", "")
}

//nolint:paralleltest // isolateCredentialChain が t.Setenv を使用するため並列化不可
func TestResolve(t *testing.T) {
	//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
		t.Run("明示注入した静的資格情報で解決する", func(t *testing.T) {
			isolateCredentialChain(t)

			got, err := awsclient.Resolve(t.Context(), awsclient.Config{
				Region:          "ap-northeast-1",
				AccessKeyID:     "dummy-key",
				SecretAccessKey: "dummy-secret",
			})

			require.NoError(t, err)
			assert.Equal(t, "ap-northeast-1", got.Region)
			creds, err := got.Credentials.Retrieve(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "dummy-key", creds.AccessKeyID)
		})

		//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
		t.Run("資格情報が空なら SDK 既定の chain へ委ねる", func(t *testing.T) {
			isolateCredentialChain(t)
			// chain の環境変数プロバイダだけを生かす。明示注入が無くても解決できる経路の代表。
			t.Setenv("AWS_ACCESS_KEY_ID", "chain-key")
			t.Setenv("AWS_SECRET_ACCESS_KEY", "chain-secret")

			got, err := awsclient.Resolve(t.Context(), awsclient.Config{Region: "us-east-1"})

			require.NoError(t, err)
			creds, err := got.Credentials.Retrieve(t.Context())
			require.NoError(t, err)
			assert.Equal(t, "chain-key", creds.AccessKeyID)
		})

		//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
		t.Run("API 呼び出し用の HTTPClient を割り当てる", func(t *testing.T) {
			isolateCredentialChain(t)
			outbound := observability.NewDisabledOutboundHTTPClient(true)

			got, err := awsclient.Resolve(t.Context(), awsclient.Config{
				Region:          "us-east-1",
				AccessKeyID:     "dummy-key",
				SecretAccessKey: "dummy-secret",
				HTTPClient:      outbound,
			})

			require.NoError(t, err)
			assert.Same(t, outbound, got.HTTPClient)
		})
	})

	//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
		t.Run("アクセスキーだけの指定を弾く", func(t *testing.T) {
			isolateCredentialChain(t)

			_, err := awsclient.Resolve(t.Context(), awsclient.Config{
				Region:      "us-east-1",
				AccessKeyID: "dummy-key",
			})

			require.ErrorIs(t, err, awsclient.ErrInvalidCredentials)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
		t.Run("シークレットだけの指定を弾く", func(t *testing.T) {
			isolateCredentialChain(t)

			_, err := awsclient.Resolve(t.Context(), awsclient.Config{
				Region:          "us-east-1",
				SecretAccessKey: "dummy-secret",
			})

			require.ErrorIs(t, err, awsclient.ErrInvalidCredentials)
		})

		//nolint:paralleltest // 親が t.Setenv を使用するため並列化不可
		t.Run("chain が資格情報を解決できなければ起動時に失敗する", func(t *testing.T) {
			isolateCredentialChain(t)

			_, err := awsclient.Resolve(t.Context(), awsclient.Config{Region: "us-east-1"})

			require.ErrorIs(t, err, awsclient.ErrInvalidCredentials)
		})
	})
}
