// Package awsclient は、AWS SDK v2 クライアントに共通する組み立て — 資格情報の解決と、
// API 呼び出し用 / 資格情報解決用に分けた HTTP クライアントの割り当て — を提供します。
// SQS・S3 などの adapter はこの結果の aws.Config からサービスクライアントを生成します。
package awsclient

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// resolveTimeout は、起動時の資格情報の解決に許す上限時間です。
// chain は複数のプロバイダを順に試すため、個々のプロバイダの既定タイムアウトだけでは全体の上限が
// 決まりません。応答しない解決先で起動が止まったままにならないよう、全体へ締切を与えます。
const resolveTimeout = 10 * time.Second

// ErrInvalidCredentials は、資格情報の指定が解決できる形になっていないことを示すエラーです。
var ErrInvalidCredentials = xerrors.Wrap(apperror.ErrInvalidArgument, "invalid aws credentials")

// Config は、AWS サービスクライアントを組み立てるための共通設定です。
type Config struct {
	// Region は、署名に用いるリージョンです。空の場合は SDK 既定の解決（環境変数・共有プロファイル）に委ねます。
	Region string
	// AccessKeyID / SecretAccessKey は、明示注入する静的資格情報です。
	// 両方空なら SDK 既定の credential chain（環境変数・共有プロファイル・web identity・
	// コンテナ / IMDS）へ委ね、両方非空ならその chain を上書きします。
	AccessKeyID     string
	SecretAccessKey string
	// HTTPClient は、サービス API の呼び出しに使う HTTP クライアントです。
	// SSRF ガード付きの実装を DI が注入します。nil を渡すと SDK 既定のトランスポートになり、
	// ガードを素通りします。
	HTTPClient aws.HTTPClient
}

// Resolve は、資格情報を解決した aws.Config を返します。
// 明示注入は chain の上書きとして扱います。解決可否は起動時に resolveTimeout 以内で一度だけ
// 確かめ、誤設定を起動時に検出します。cfg.HTTPClient（SSRF ガード付き）はサービス API の呼び出しに
// だけ割り当て、資格情報の解決（IMDS / ECS task metadata / STS / SSO を含む）は SDK 既定の
// トランスポートで行います。これらの設計判断の理由は README.md を参照してください。
func Resolve(ctx context.Context, cfg Config) (aws.Config, error) {
	if err := ctx.Err(); err != nil {
		return aws.Config{}, xerrors.Wrap(ErrInvalidCredentials, err.Error())
	}

	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}

	switch {
	case cfg.AccessKeyID != "" && cfg.SecretAccessKey != "":
		opts = append(opts, awsconfig.WithCredentialsProvider(
			awscreds.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	case cfg.AccessKeyID != "" || cfg.SecretAccessKey != "":
		// 片方だけの指定は chain 委譲とも明示注入とも読めない。SDK は access key ID さえあれば
		// 署名を作ってしまい、認証エラーが呼び出しまで出ないため、ここで弾く。
		return aws.Config{}, xerrors.Wrap(ErrInvalidCredentials,
			"access key id and secret access key must be set together, or both left empty to use the default chain")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, xerrors.Wrap(ErrInvalidCredentials, err.Error())
	}

	retrieveCtx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()
	if _, err := awsCfg.Credentials.Retrieve(retrieveCtx); err != nil {
		return aws.Config{}, xerrors.Wrap(ErrInvalidCredentials, err.Error())
	}

	awsCfg.HTTPClient = cfg.HTTPClient
	return awsCfg, nil
}
