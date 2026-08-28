// Package realtimeinit は、Realtime Delivery の 3 つの table（EventLog / StreamTicket / InstanceLease）を
// 作る one-shot 初期化のコアロジックを提供します。application の起動時には呼ばれず、`realtime-init`
// コマンド（make realtime-init）から実行します。何度実行しても同じ状態に収束します。
// table の中身（キー定義）は infrastructure 側が持ち、ここは名前の並びと「止める / 続ける」だけを決めます。
package realtimeinit

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

// Ensurer は、名前で指定した table を実在させる関数型です（実体は cmd 側が infrastructure の
// TableSpec に束ねて渡します）。
type Ensurer func(ctx context.Context, table string) error

// TableNames は、config が指す 3 table の名前を作成順に返します。
func TableNames(cfg *config.RealtimeConfig) []string {
	return []string{cfg.EventLogTable(), cfg.StreamTicketTable(), cfg.InstanceLeaseTable()}
}

// Run は、3 table を順に実在させます。1 つでも失敗したらその table 名を添えて止まります
// （残りを続けても中途半端な状態を増やすだけで、再実行すれば冪等に続きから収束するため）。
func Run(ctx context.Context, cfg *config.RealtimeConfig, ensure Ensurer, logger logging.Logger) error {
	for _, table := range TableNames(cfg) {
		if err := ensure(ctx, table); err != nil {
			return xerrors.Wrap(err, "realtime-init: "+table)
		}

		logger.Info(ctx, "table is ready", logging.String("table", table))
	}

	return nil
}
