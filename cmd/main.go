// Package main はアプリケーションのエントリポイントです。
// このパッケージは、Cobraを使用してコマンドラインインターフェースを提供します。
package main

import (
	"os"

	"go-boilerplate/internal/system"

	"github.com/spf13/cobra"
)

func main() {
	const applicationName = "go-boilerplate"

	c := &cobra.Command{
		Use:   applicationName,
		Short: applicationName + "アプリケーションのCLIエントリポイントです。",
		Long: applicationName + "は、開発・マイグレーション・コード生成などを行うためのコマンドラインツールです。\n" +
			"用途に応じて、\"serve\", \"migrate-up\", \"seed\"などのサブコマンドを使用します。",
		Version: system.Version + " (rev: " + system.Revision + ", built at: " + system.BuildDate + ")",
		// エラー／usage 出力は main 側へ一本化する（cobra 既定の Error 出力との二重表示を抑止）。
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	registerCommands(c)
	if err := c.Execute(); err != nil {
		// serve に限らず任意のサブコマンド失敗で到達するため、汎用文言にする。
		c.PrintErrln("コマンドの実行に失敗しました。エラー内容を確認してください。: ", err)
		os.Exit(1)
	}
}
