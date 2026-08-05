// Package awsclient は、AWS SDK v2 クライアントに共通する組み立て — 資格情報の解決と、
// API 呼び出し用 / 資格情報解決用に分けた HTTP クライアントの割り当て — を提供します。
// SQS・S3 などの adapter はこの結果の aws.Config からサービスクライアントを生成します。
package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

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
//
// 資格情報は「明示注入か chain か」を設定で選ばせません。SDK 既定の chain は静的資格情報
// （環境変数プロバイダ）を自身の一部として含むため、static / chain の二者択一は SDK の標準形に
// 存在せず、判別子を足すのは標準が決めている所へノブを置くことになるためです。明示注入は
// chain の上書きとして扱います。
//
// 解決可否は起動時に一度だけ確かめます。誤設定のまま起動すると、最初の API 呼び出しまで
// 認証エラーが顕在化しません。
//
// 資格情報の解決には SDK 既定のトランスポートを使い、cfg.HTTPClient（SSRF ガード付き）は
// サービス API の呼び出しにだけ割り当てます。IMDS（169.254.169.254）と ECS の
// task metadata（169.254.170.2）は link-local にあり、ガードは link-local を常に拒否するため、
// 同じクライアントを共有すると EC2 / ECS のロール運用だけが解決不能になります。守る対象が
// 違う — ガードが防ぐのは外部サービスへの egress であって、自身の実行基盤への資格情報の
// 問い合わせではありません。
func Resolve(ctx context.Context, cfg Config) (aws.Config, error) {
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

	if _, err := awsCfg.Credentials.Retrieve(ctx); err != nil {
		return aws.Config{}, xerrors.Wrap(ErrInvalidCredentials, err.Error())
	}

	awsCfg.HTTPClient = cfg.HTTPClient
	return awsCfg, nil
}
