// Package env は、バイナリへ焼き込む設定とマイグレーションを embed.FS として公開します。
package env

import "embed"

// FS は env/.env と database/migrations を埋め込んだ読み取り専用ファイルシステムです。
//
//go:embed env/.env database/migrations
var FS embed.FS
