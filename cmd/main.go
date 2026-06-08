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
			"用途に応じて、\"serve\", \"migrate-up\",  \"seed\"などのサブコマンドを使用します。",
		Version: system.Version + " (rev: " + system.Revision + ", built at: " + system.BuildDate + ")",
	}
	registerCommands(c)
	if err := c.Execute(); err != nil {
		c.PrintErrln("起動に失敗しました。エラー内容を確認してください。: ", err)
		os.Exit(1)
	}

	os.Exit(0)
}
