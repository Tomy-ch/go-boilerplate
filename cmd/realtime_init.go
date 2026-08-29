package main

import (
	"context"

	"github.com/spf13/cobra"

	clirealtimeinit "go-boilerplate/internal/cli/realtimeinit"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	eventlogdynamo "go-boilerplate/internal/infrastructure/eventlog/dynamodb"
	instanceleasedynamo "go-boilerplate/internal/infrastructure/instancelease/dynamodb"
	realtimeinfra "go-boilerplate/internal/infrastructure/realtime"
	streamticketdynamo "go-boilerplate/internal/infrastructure/streamticket/dynamodb"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

// errUnknownRealtimeTable は、CLI コアが要求した table 名に対応する定義が無い場合のエラーです。
var errUnknownRealtimeTable = xerrors.New("realtime-init: no table definition for")

// newRealtimeInitCommand は、Realtime Delivery の table を作る one-shot コマンドを生成します。
func newRealtimeInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "realtime-init",
		Short: "Realtime Delivery の table（EventLog / StreamTicket / InstanceLease）を作成します（冪等）。",
		Long: `Realtime Delivery が使う DynamoDB 互換 store の table を作成します。

REALTIME_* と ENDPOINT_REALTIME の設定が指す store に対して、無い table は作り、ACTIVE を待ち、
TTL が未設定なら設定します。既にある table は変えません。application の起動時には実行されないため、
新しい環境や REALTIME_TABLE_SUFFIX を変えた後に一度実行してください。`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRealtimeInit(cmd.Context())
		},
	}
}

// runRealtimeInit は、設定を読み込み、DynamoDB クライアントを組み立てて初期化を実行します。
// CLI は operator が設定した endpoint にだけ繋ぐため、SSRF ガード付きの HTTP クライアントは使いません。
func runRealtimeInit(ctx context.Context) error {
	cfg, err := newCLIConfig("")
	if err != nil {
		return err
	}

	rtCfg := config.NewRealtimeConfig(cfg)
	client, err := dynamodbclient.New(ctx, dynamodbclient.Config{
		Endpoint:        config.NewEndpointConfig(cfg).Realtime(),
		Region:          rtCfg.Region(),
		AccessKeyID:     rtCfg.AccessKeyID(),
		SecretAccessKey: rtCfg.SecretAccessKey(),
	})
	if err != nil {
		return xerrors.Wrap(err, "failed to build dynamodb client")
	}

	// table の定義（キー / index / TTL）は各 adapter package が持つ。
	specs := map[string]dynamodbclient.TableSpec{
		rtCfg.EventLogTable():      eventlogdynamo.TableSpec(rtCfg.EventLogTable()),
		rtCfg.StreamTicketTable():  streamticketdynamo.TableSpec(rtCfg.StreamTicketTable()),
		rtCfg.InstanceLeaseTable(): instanceleasedynamo.TableSpec(rtCfg.InstanceLeaseTable()),
	}
	ensure := func(ctx context.Context, table string) error {
		spec, ok := specs[table]
		if !ok {
			return xerrors.Wrap(errUnknownRealtimeTable, table)
		}

		return dynamodbclient.EnsureTable(ctx, client, spec)
	}

	logger := logging.NewJSONLogger(logging.LevelInfo(), logging.LevelError(), nil)
	if err := clirealtimeinit.Run(ctx, rtCfg, ensure, logger); err != nil {
		return err
	}

	clients, err := realtimeinfra.NewClients(ctx, realtimeinfra.ClientConfig{
		Endpoint:        config.NewEndpointConfig(cfg).RealtimePubSub(),
		Region:          rtCfg.Region(),
		AccessKeyID:     rtCfg.AccessKeyID(),
		SecretAccessKey: rtCfg.SecretAccessKey(),
	})
	if err != nil {
		return xerrors.Wrap(err, "failed to build sns client")
	}

	ensureTopic := func(ctx context.Context, name string) (string, error) {
		return realtimeinfra.EnsureTopic(ctx, clients, name)
	}

	return clirealtimeinit.RunTopic(ctx, rtCfg.Topic(), ensureTopic, logger)
}
