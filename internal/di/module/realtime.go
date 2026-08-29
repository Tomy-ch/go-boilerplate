package module

import (
	"context"

	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.uber.org/fx"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	oapiauth "go-boilerplate/internal/controller/httpstack/oapi/auth"
	ctrlrealtime "go-boilerplate/internal/controller/realtime"
	"go-boilerplate/internal/controller/stream"
	streamauth "go-boilerplate/internal/controller/stream/auth"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/server/hook"
	"go-boilerplate/internal/infrastructure/dynamodbclient"
	"go-boilerplate/internal/infrastructure/eventlog"
	"go-boilerplate/internal/infrastructure/instancelease"
	realtimeinfra "go-boilerplate/internal/infrastructure/realtime"
	"go-boilerplate/internal/infrastructure/realtimesecret"
	"go-boilerplate/internal/infrastructure/streamticket"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// realtimeParticipantName は、serve lifecycle のログに載せる Realtime Delivery の参加者名です。
	realtimeParticipantName = "realtime"
	// startupProbeStreamID は、起動時に EventLog へ到達できるかを確かめる読み取りに使う stream です。
	// 存在しない stream の Latest は「無い」を返すだけで、store に届かなければエラーになります。
	startupProbeStreamID rt.StreamID = "_readiness"
)

// ErrRealtimeTopicNotConfigured は、fan-out を配線したのに REALTIME_TOPIC が空であることを示すエラーです。
var ErrRealtimeTopicNotConfigured = xerrors.Wrap(
	apperror.ErrInvalidArgument,
	"REALTIME_TOPIC must be set when the realtime fan-out is wired",
)

// realtimeFanout は、fan-out の publish 側と受信側が共有する SNS / SQS クライアントと topic です。
type realtimeFanout struct {
	clients  realtimeinfra.Clients
	topicARN string
}

// realtimeModule は、Realtime Delivery の store・機構側 usecase・fan-out・SSE の stream handler・serve lifecycle の
// 参加者を提供する fx.Module です。InfrastructureModule() には束ねず serve profile にだけ配線します
// （内訳は internal/di/module/README.md「Design Policy」、配線条件は docs/design/realtime-delivery.md §3.1）。
// Waker / Revoker は connection registry（controller/stream）が provide します。
func realtimeModule() fx.Option {
	return fx.Module("realtime",
		fx.Provide(
			provideRealtimeClient,
			provideEventLogStore,
			provideStreamTicketStore,
			provideInstanceLeaseStore,
			provideRealtimeSecretGenerator,
			provideCursorValidator,
			provideTicketIssuer,
			provideTicketVerifier,
			fx.Annotate(provideStreamTicketScheme, fx.ResultTags(`group:"`+oapiauth.SchemeGroup+`"`)),
			provideRealtimeFanout,
			provideRevocationNotifier,
			provideAccessRevoker,
			provideInstanceID,
			provideInstanceQueueAttributes,
			provideInstanceSubscription,
			provideLeaseKeeper,
			provideRealtimeConsumer,
			provideRealtimeHeartbeat,
			fx.Annotate(provideRealtimeStartupProbe, fx.ResultTags(`group:"`+hook.StartupGroup+`"`)),
			fx.Annotate(provideRealtimeProvisioner, fx.ResultTags(`group:"`+hook.ProvisionerGroup+`"`)),
			fx.Annotate(provideRealtimeConsumerRunner, fx.ResultTags(`group:"`+hook.RunnerGroup+`"`)),
			fx.Annotate(provideRealtimeHeartbeatRunner, fx.ResultTags(`group:"`+hook.RunnerGroup+`"`)),
		),
		fx.Invoke(
			stream.BindHandler,
		),
	)
}

// provideRealtimeFanout は、fan-out のクライアントを組み立てます。topic の ARN が空なら起動を失敗させます
// （publish 先の無い fan-out を黙って起動させない。ENDPOINT_OUTBOX と同じ扱い）。
func provideRealtimeFanout(
	cfg *config.RealtimeConfig,
	epCfg *config.EndpointConfig,
	outbound *observability.OutboundHTTPClient,
) (realtimeFanout, error) {
	return newRealtimeFanout(cfg.Topic(), realtimeinfra.ClientConfig{
		Endpoint:        epCfg.RealtimePubSub(),
		Region:          cfg.Region(),
		AccessKeyID:     cfg.AccessKeyID(),
		SecretAccessKey: cfg.SecretAccessKey(),
		HTTPClient:      outbound,
	})
}

func newRealtimeFanout(topicARN string, cc realtimeinfra.ClientConfig) (realtimeFanout, error) {
	if topicARN == "" {
		return realtimeFanout{}, ErrRealtimeTopicNotConfigured
	}

	clients, err := realtimeinfra.NewClients(context.Background(), cc)
	if err != nil {
		return realtimeFanout{}, err
	}

	return realtimeFanout{clients: clients, topicARN: topicARN}, nil
}

func provideRevocationNotifier(f realtimeFanout, tf observability.TracerFactory) rt.RevocationNotifier {
	return realtimeinfra.NewRevocationNotifier(f.clients, f.topicARN, tf)
}

func provideAccessRevoker(
	tickets rt.StreamTicketStore,
	notifier rt.RevocationNotifier,
	tf observability.TracerFactory,
) ucrealtime.AccessRevoker {
	return ucrealtime.NewAccessRevoker(tickets, notifier, tf)
}

// provideInstanceID は、この process の serve instance の識別子を起動ごとに採番します。
// 再起動した instance は別の識別子になり、前世代が残した resource は orphan として回収されます（hostname を
// 使うと前世代の lease と衝突し、生きている instance の resource が回収され得る）。
func provideInstanceID() (rt.InstanceID, error) {
	id, err := uuid.New()
	if err != nil {
		return "", xerrors.Wrap(err, "generate realtime instance id")
	}

	return rt.InstanceID(id.String()), nil
}

// provideInstanceQueueAttributes は、instance queue に設定する属性の組み立て手を環境で選びます。
// emulator（GoAWS）が受け付けない属性と、どの環境がどちらの実装を使うかは
// internal/infrastructure/realtime/README.md「Emulator compatibility」を参照。
func provideInstanceQueueAttributes(
	appCfg *config.ApplicationConfig,
	cfg *config.RealtimeConfig,
) (realtimeinfra.AttributesBuilder, error) {
	switch env := appCfg.Env(); env {
	case config.EnvLocal, config.EnvCI, config.EnvTest, config.EnvDast:
		return realtimeinfra.NewEmulatorQueueAttributes(), nil
	case config.EnvDevelopment, config.EnvStaging, config.EnvProduction:
		return realtimeinfra.NewQueueAttributes(realtimeinfra.QueueAttributesInput{TopicARN: cfg.Topic(), DLQARN: cfg.DLQ()}), nil
	default:
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "no instance queue attribute policy for env "+env)
	}
}

func provideInstanceSubscription(
	f realtimeFanout, cfg *config.RealtimeConfig, attrs realtimeinfra.AttributesBuilder, tf observability.TracerFactory,
) rt.InstanceSubscription {
	return realtimeinfra.NewInstanceSubscription(
		f.clients, realtimeinfra.SubscriptionTarget{TopicARN: f.topicARN, QueuePrefix: cfg.QueuePrefix()}, attrs, tf,
	)
}

func provideLeaseKeeper(
	store rt.InstanceLeaseStore,
	clk clock.Clock,
	tf observability.TracerFactory,
) ucrealtime.LeaseKeeper {
	return ucrealtime.NewLeaseKeeper(store, clk, tf)
}

func provideRealtimeConsumer(
	sub rt.InstanceSubscription,
	wakeups ctrlrealtime.Waker,
	revocations ctrlrealtime.Revoker,
	sleeper clock.Sleeper,
	log logging.Logger,
	tf observability.TracerFactory,
) *ctrlrealtime.Engine {
	return ctrlrealtime.NewEngine(sub, wakeups, revocations, sleeper, log, tf, ctrlrealtime.Settings{})
}

func provideRealtimeHeartbeat(
	keeper ucrealtime.LeaseKeeper,
	id rt.InstanceID,
	sleeper clock.Sleeper,
	log logging.Logger,
	tf observability.TracerFactory,
) *ctrlrealtime.Heartbeat {
	return ctrlrealtime.NewHeartbeat(keeper, id, sleeper, log, tf)
}

// provideRealtimeStartupProbe は、HTTP を listen する前に EventLog へ到達できることを確かめる
// [hook.StartupProbe] を返します。
func provideRealtimeStartupProbe(log rt.EventLogStore) hook.StartupProbe {
	return hook.StartupProbe{Name: realtimeParticipantName, Probe: func(ctx context.Context) error {
		_, _, err := log.Latest(ctx, startupProbeStreamID)

		return err
	}}
}

// provideRealtimeProvisioner は、lease の記録と instance の受信先を 1 つの参加者に合成し、
// 起動は lease → 受信先、片付けは 受信先 → lease の順に固定します
// （fx の value group は順序を保証しない。順序そのものの根拠は docs/design/realtime-delivery.md §2.5）。
func provideRealtimeProvisioner(
	sub rt.InstanceSubscription,
	keeper ucrealtime.LeaseKeeper,
	id rt.InstanceID,
) hook.Provisioner {
	return hook.Provisioner{
		Name: realtimeParticipantName,
		Provision: func(ctx context.Context) error {
			if err := keeper.Beat(ctx, id); err != nil {
				return err
			}

			if err := sub.Provision(ctx, id); err != nil {
				if terr := sub.Teardown(ctx); terr != nil {
					return xerrors.Join(err, terr) // 受信先が残ったので lease は残す
				}

				return xerrors.Join(err, keeper.Release(ctx, id))
			}

			return nil
		},
		Teardown: func(ctx context.Context) error {
			if err := sub.Teardown(ctx); err != nil {
				return err // 受信先が残ったので lease は残す
			}

			return keeper.Release(ctx, id)
		},
	}
}

func provideRealtimeConsumerRunner(engine *ctrlrealtime.Engine) hook.Runner {
	return hook.Runner{Name: realtimeParticipantName + "-consumer", Runner: lifecycle.SupervisedRunner{
		Body: func(ctx context.Context) { _ = engine.Run(ctx) },
	}}
}

func provideRealtimeHeartbeatRunner(heartbeat *ctrlrealtime.Heartbeat) hook.Runner {
	return hook.Runner{Name: realtimeParticipantName + "-heartbeat", Runner: lifecycle.SupervisedRunner{
		Body: func(ctx context.Context) { _ = heartbeat.Run(ctx) },
	}}
}

// provideRealtimeClient は、Realtime Delivery の store が共有する DynamoDB クライアントを組み立てます。
// 資格情報を解決できない場合はエラーを返し、app.Start を失敗させます。
func provideRealtimeClient(
	cfg *config.RealtimeConfig,
	epCfg *config.EndpointConfig,
	outbound *observability.OutboundHTTPClient,
) (*awsdynamodb.Client, error) {
	return dynamodbclient.New(context.Background(), dynamodbclient.Config{
		Endpoint:        epCfg.Realtime(),
		Region:          cfg.Region(),
		AccessKeyID:     cfg.AccessKeyID(),
		SecretAccessKey: cfg.SecretAccessKey(),
		HTTPClient:      outbound,
	})
}

func provideEventLogStore(
	c *awsdynamodb.Client,
	cfg *config.RealtimeConfig,
	tf observability.TracerFactory,
) rt.EventLogStore {
	return eventlog.New(c, cfg.EventLogTable(), tf)
}

func provideStreamTicketStore(
	c *awsdynamodb.Client,
	cfg *config.RealtimeConfig,
	tf observability.TracerFactory,
) rt.StreamTicketStore {
	return streamticket.New(c, cfg.StreamTicketTable(), tf)
}

func provideInstanceLeaseStore(
	c *awsdynamodb.Client,
	cfg *config.RealtimeConfig,
	tf observability.TracerFactory,
) rt.InstanceLeaseStore {
	return instancelease.New(c, cfg.InstanceLeaseTable(), tf)
}

func provideRealtimeSecretGenerator() rt.SecretGenerator {
	return realtimesecret.New()
}

func provideCursorValidator(
	log rt.EventLogStore,
	clk clock.Clock,
	tf observability.TracerFactory,
) ucrealtime.CursorValidator {
	return ucrealtime.NewCursorValidator(log, clk, tf)
}

func provideTicketIssuer(
	store rt.StreamTicketStore, secrets rt.SecretGenerator, clk clock.Clock, tf observability.TracerFactory,
) ucrealtime.TicketIssuer {
	return ucrealtime.NewTicketIssuer(store, secrets, clk, tf)
}

func provideTicketVerifier(
	store rt.StreamTicketStore,
	clk clock.Clock,
	tf observability.TracerFactory,
) ucrealtime.TicketVerifier {
	return ucrealtime.NewTicketVerifier(store, clk, tf)
}

// provideStreamTicketScheme は、query の stream ticket を検証する認証器を oapi/auth の scheme group へ出します。
func provideStreamTicketScheme(verifier ucrealtime.TicketVerifier) oapiauth.SchemeAuthenticator {
	return streamauth.New(verifier)
}
