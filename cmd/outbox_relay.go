package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	outboxcli "go-boilerplate/internal/cli/outbox"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
)

// newOutboxRelayCommand は、outbox relay コマンドを生成します（replay サブコマンドを含む）。
func newOutboxRelayCommand() *cobra.Command {
	var channel string

	cmd := &cobra.Command{
		Use:   "outbox-relay",
		Short: "outbox relay を起動します。",
		Long: "outbox-relay コマンドは、outbox テーブルを周期 poll して未 publish メッセージを送る relay を起動し、\n" +
			"SIGTERM まで常駐します。--channel で担当する配送チャネルを 1 つ選び、そのチャネルの行だけを配送します。",
		RunE: func(_ *cobra.Command, _ []string) error { return outboxRelayRun(channel) },
	}
	cmd.Flags().StringVar(&channel, "channel", outboxbndry.ChannelHTTP.String(), "担当する配送チャネル（http / realtime）")
	cmd.AddCommand(newOutboxReplayCommand())
	return cmd
}

// outboxRelayRun は outboxcli.RunRelay への薄い委譲殻です。
func outboxRelayRun(channel string) error {
	ch, err := outboxbndry.ParseChannel(channel)
	if err != nil {
		return err
	}

	cfg, err := config.SetUpConfig()
	if err != nil {
		return err
	}
	appCfg := config.NewApplicationConfig(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app := di.NewOutboxRelayApp(appCfg.ShutdownTimeout(), ch)
	startApp, stopApp := di.NewApplicationServer(app)

	return outboxcli.RunRelay(ctx, appCfg.ShutdownTimeout(), startApp, stopApp)
}

// newOutboxReplayCommand は、dead 行を pending へ戻す replay サブコマンドを生成します。
func newOutboxReplayCommand() *cobra.Command {
	var messageID string

	cmd := &cobra.Command{
		Use:   "replay",
		Short: "dead 状態の outbox 行を pending へ戻します。",
		Long: "replay コマンドは、dead 状態の outbox 行を pending へ戻し再 publish 対象に復帰させます。\n" +
			"--message-id 指定時は当該行のみ、未指定なら全 dead 行が対象です。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			count, err := outboxcli.RunReplayWith(cmd.Context(), messageID, di.RunOutboxReplay)
			if err != nil {
				return err
			}
			cmd.Printf("replayed %d outbox row(s)\n", count)
			return nil
		},
	}
	cmd.Flags().StringVar(&messageID, "message-id", "", "対象の message_id（未指定なら全 dead 行）")
	return cmd
}
