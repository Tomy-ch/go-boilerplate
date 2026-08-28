// Package realtimeinit は、Realtime Delivery の 3 つの table（EventLog / StreamTicket / InstanceLease）を
// 作る one-shot 初期化のコアロジックを提供します。application の起動時には呼ばれず、`realtime-init`
// コマンド（make realtime-init）から実行します。何度実行しても同じ状態に収束します。
package realtimeinit

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	eventlogdynamo "go-boilerplate/internal/infrastructure/eventlog/dynamodb"
	instanceleasedynamo "go-boilerplate/internal/infrastructure/instancelease/dynamodb"
	streamticketdynamo "go-boilerplate/internal/infrastructure/streamticket/dynamodb"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

// Ensurer は、1 つの table 定義を実在させる関数型です（実体は dynamodbclient.EnsureTable）。
type Ensurer func(ctx context.Context, spec dynamodbclient.TableSpec) error

// Specs は、config が指す名前で 3 table の定義を返します。
func Specs(cfg *config.RealtimeConfig) []dynamodbclient.TableSpec {
	return []dynamodbclient.TableSpec{
		eventlogdynamo.TableSpec(cfg.EventLogTable()),
		streamticketdynamo.TableSpec(cfg.StreamTicketTable()),
		instanceleasedynamo.TableSpec(cfg.InstanceLeaseTable()),
	}
}

// Run は、3 table を順に実在させます。1 つでも失敗したらその table 名を添えて止まります
// （残りを続けても中途半端な状態を増やすだけで、再実行すれば冪等に続きから収束するため）。
func Run(ctx context.Context, cfg *config.RealtimeConfig, ensure Ensurer, logger logging.Logger) error {
	for _, spec := range Specs(cfg) {
		if err := ensure(ctx, spec); err != nil {
			return xerrors.Wrap(err, "realtime-init: "+spec.Name)
		}

		logger.Info(ctx, "table is ready", logging.String("table", spec.Name))
	}

	return nil
}
