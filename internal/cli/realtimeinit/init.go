// Package realtimeinit は、Realtime Delivery の 3 つの table（EventLog / StreamTicket / InstanceLease）と
// fan-out の topic を作る one-shot 初期化のコアロジックを提供します。application の起動時には呼ばれず、
// `realtime-init` コマンド（make realtime-init）から実行します。何度実行しても同じ状態に収束します。
// table の中身（キー定義）と topic の作り方は infrastructure 側が持ち、ここは名前の並びと「止める / 続ける」だけを決めます。
package realtimeinit

import (
	"context"
	"strings"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

// ErrTopicARNInvalid は、設定の topic ARN から topic 名を導けないことを示すエラーです。
var ErrTopicARNInvalid = xerrors.Wrap(
	apperror.ErrInvalidArgument,
	"realtime-init: REALTIME_TOPIC is not a topic arn",
)

// ErrTopicARNMismatch は、作成した topic の ARN が設定の ARN と食い違うことを示すエラーです。
// 食い違ったまま起動すると publish 先だけが別の topic を指し、配送が黙って途切れるため、ここで止めます。
var ErrTopicARNMismatch = xerrors.Wrap(
	apperror.ErrInvalidArgument,
	"realtime-init: created topic arn does not match REALTIME_TOPIC",
)

// Ensurer は、名前で指定した table を実在させる関数型です（実体は cmd 側が infrastructure の
// TableSpec に束ねて渡します）。
type Ensurer func(ctx context.Context, table string) error

// TopicEnsurer は、名前で指定した topic を実在させ、その ARN を返す関数型です（実体は cmd 側が SNS クライアントに束ねて渡します）。
type TopicEnsurer func(ctx context.Context, name string) (arn string, err error)

// TableNames は、config が指す 3 table の名前を作成順に返します。
func TableNames(cfg *config.RealtimeConfig) []string {
	return []string{cfg.EventLogTable(), cfg.StreamTicketTable(), cfg.InstanceLeaseTable()}
}

// TopicName は、topic ARN の末尾要素（= topic 名）を返します。ARN が空か、`:` で区切れないか、
// 末尾が空なら ErrTopicARNInvalid です。
func TopicName(arn string) (string, error) {
	i := strings.LastIndex(arn, ":")
	if arn == "" || i < 0 || i == len(arn)-1 {
		return "", xerrors.Wrap(ErrTopicARNInvalid, arn)
	}

	return arn[i+1:], nil
}

// RunTopic は、topicARN（config の REALTIME_TOPIC）が指す topic を実在させます。作成結果の ARN が食い違えば
// ErrTopicARNMismatch で止まります（emulator の account / region と config の ARN が合っていない誤設定をここで見つける）。
func RunTopic(ctx context.Context, topicARN string, ensure TopicEnsurer, logger logging.Logger) error {
	name, err := TopicName(topicARN)
	if err != nil {
		return err
	}

	arn, err := ensure(ctx, name)
	if err != nil {
		return xerrors.Wrap(err, "realtime-init: "+name)
	}

	if arn != topicARN {
		return xerrors.Wrap(ErrTopicARNMismatch, "created "+arn+", configured "+topicARN)
	}

	logger.Info(ctx, "topic is ready", logging.String("topic", arn))

	return nil
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
