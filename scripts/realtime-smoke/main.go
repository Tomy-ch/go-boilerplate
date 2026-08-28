// Package main は、DynamoDB Local と GoAWS に AWS SDK Go v2 で native 接続できるかを確かめる smoke です。
// Realtime Delivery（docs/design/realtime-delivery.md）が production で使う呼び出し — DynamoDB の
// conditional put / ConsistentRead query / TTL、SNS topic → N 個の SQS queue への fan-out
// （RawMessageDelivery）、queue policy — をそのまま emulator に投げ、互換 / 非互換 / 未対応 / 検証不能 を
// 表にします。表の非互換と未対応だけが local compatibility implementation の範囲になります。
//
// 共有インフラ上で複数の checkout から同時に走り得るため、作る resource は実行ごとの乱数 ID を含む
// 名前にし、終了時に削除します（-keep で残せます）。
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"go-boilerplate/pkg/xerrors"
)

const (
	defaultDynamoDBEndpoint = "http://localhost:8000"
	defaultGoAWSEndpoint    = "http://localhost:4100"
	defaultRegion           = "us-east-1"
	defaultSubscribers      = 3
	defaultTimeout          = 120 * time.Second
	defaultReadyTimeout     = 30 * time.Second
	readyPollInterval       = 500 * time.Millisecond

	formatMarkdown = "markdown"
	formatText     = "text"

	// smokeCredential は、emulator へ渡す静的資格情報です。どちらの emulator も認証しませんが、
	// SDK は署名のために非空の資格情報を要求します。
	smokeCredential = "smoke"
)

var (
	errSubscribers = xerrors.New("-subscribers は 1 以上を指定してください")
	errFormat      = xerrors.New("-format は markdown か text を指定してください")
	errNotReady    = xerrors.New("endpoint が ready にならず、検査を 1 件も実行していません")
)

// options は、コマンドラインで決まる実行条件です。
type options struct {
	dynamoDBEndpoint string
	goAWSEndpoint    string
	region           string
	subscribers      int
	timeout          time.Duration
	readyTimeout     time.Duration
	format           string
	keep             bool
	strict           bool
}

// dialFunc は、readiness 判定に使う TCP 接続手段です。テストで差し替えます。
type dialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// clients は、smoke が叩く 3 つのサービスクライアントです。
type clients struct {
	dynamoDB *dynamodb.Client
	sns      *sns.Client
	sqs      *sqs.Client
}

func main() {
	log.SetFlags(0)

	code, err := run(context.Background(), os.Args[1:], os.Stdout, rand.Reader, (&net.Dialer{}).DialContext)
	if err != nil {
		log.Printf("❌ %v", err)
	}

	os.Exit(code)
}

// run は、flag を解釈し、readiness を待ち、検査を実行して表を書き出し、終了コードを返します。
// 乱数源と TCP 接続手段は差し替えられるよう引数で受けます。
func run(ctx context.Context, args []string, out io.Writer, random io.Reader, dial dialFunc) (int, error) {
	opts, err := parseOptions(args)
	if err != nil {
		if xerrors.Is(err, flag.ErrHelp) {
			return 0, nil
		}

		return 1, err
	}

	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()

	if err := waitReady(ctx, dial, opts.readyTimeout, opts.dynamoDBEndpoint, opts.goAWSEndpoint); err != nil {
		return 1, err
	}

	runID, err := newRunID(random)
	if err != nil {
		return 1, err
	}

	clients, err := newClients(ctx, opts)
	if err != nil {
		return 1, err
	}

	rec := &recorder{}
	n := names{runID: runID}
	log.Printf("ℹ️ run id: %s（table %s / topic %s）", runID, n.table(), n.topic())

	runDynamoDB(ctx, clients.dynamoDB, n.table(), opts.keep, rec)
	runPubSub(ctx, clients.sns, clients.sqs, n, opts.subscribers, opts.keep, rec)

	if err := write(out, opts.format, rec.results); err != nil {
		return 1, err
	}

	return exitCode(rec.results, opts.strict)
}

func parseOptions(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("realtime-smoke", flag.ContinueOnError)
	fs.StringVar(&opts.dynamoDBEndpoint, "dynamodb-endpoint", defaultDynamoDBEndpoint, "DynamoDB Local の endpoint")
	fs.StringVar(&opts.goAWSEndpoint, "goaws-endpoint", defaultGoAWSEndpoint, "GoAWS（SNS / SQS）の endpoint")
	fs.StringVar(&opts.region, "region", defaultRegion, "署名に使う region")
	fs.IntVar(&opts.subscribers, "subscribers", defaultSubscribers, "fan-out 先の SQS queue 数")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "実行全体の上限時間")
	fs.DurationVar(&opts.readyTimeout, "ready-timeout", defaultReadyTimeout, "endpoint の ready を待つ上限時間")
	fs.StringVar(&opts.format, "format", formatMarkdown, "出力形式（markdown / text）")
	fs.BoolVar(&opts.keep, "keep", false, "作成した resource を削除せず残す")
	fs.BoolVar(&opts.strict, "strict", false, "非互換 / 未対応 があれば非ゼロで終了する")

	if err := fs.Parse(args); err != nil {
		if xerrors.Is(err, flag.ErrHelp) {
			return opts, err
		}

		return opts, xerrors.Wrap(err, "parse flags")
	}

	if opts.subscribers < 1 {
		return opts, errSubscribers
	}

	if opts.format != formatMarkdown && opts.format != formatText {
		return opts, errFormat
	}

	return opts, nil
}

// waitReady は、各 endpoint へ TCP 接続できるまで待ちます。compose の healthcheck に依存しないのは、
// `infra-up --wait` が healthcheck の無いサービスを running の時点で ready と見なすためです。
func waitReady(ctx context.Context, dial dialFunc, timeout time.Duration, endpoints ...string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, ep := range endpoints {
		addr, err := hostPort(ep)
		if err != nil {
			return err
		}

		if err := waitTCP(ctx, dial, addr); err != nil {
			return xerrors.Join(errNotReady, xerrors.Wrap(err, ep))
		}
	}

	return nil
}

func waitTCP(ctx context.Context, dial dialFunc, addr string) error {
	for {
		conn, err := dial(ctx, "tcp", addr)
		if err == nil {
			return conn.Close()
		}

		select {
		case <-ctx.Done():
			return xerrors.Wrap(err, "last dial error")
		case <-time.After(readyPollInterval):
		}
	}
}

// hostPort は、endpoint URL から接続先 host:port を返します。port 省略時は scheme の既定を使います。
func hostPort(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", xerrors.Wrap(err, "parse endpoint")
	}

	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}

	return net.JoinHostPort(u.Hostname(), port), nil
}

// newClients は、静的資格情報と endpoint 上書きでクライアントを組み立てます。application の
// awsclient と同じ SDK 経路（LoadDefaultConfig + BaseEndpoint）を通し、smoke の結論が
// application の接続形に対して成り立つようにします。
func newClients(ctx context.Context, opts options) (clients, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(opts.region),
		awsconfig.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider(smokeCredential, smokeCredential, "")),
	)
	if err != nil {
		return clients{}, xerrors.Wrap(err, "load aws config")
	}

	return clients{
		dynamoDB: dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(opts.dynamoDBEndpoint)
		}),
		sns: sns.NewFromConfig(cfg, func(o *sns.Options) {
			o.BaseEndpoint = aws.String(opts.goAWSEndpoint)
		}),
		sqs: sqs.NewFromConfig(cfg, func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(opts.goAWSEndpoint)
		}),
	}, nil
}

func write(out io.Writer, format string, results []Result) error {
	if format == formatText {
		return writeText(out, results)
	}

	return writeMarkdown(out, results)
}
