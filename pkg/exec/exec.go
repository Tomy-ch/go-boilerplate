// Package exec は、外部プロセス実行の薄いラッパーを提供します。
// os/exec への直接依存を一点に集約し、利用側はインターフェース経由で差し替え可能にします。
package exec

import (
	"bytes"
	"context"
	"os"
	osexec "os/exec"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock_exec.gen.go -package=mock_$GOPACKAGE

// Runner は、外部コマンド実行を抽象化するインターフェースです。
type Runner interface {
	// Output は dir をカレントとしてコマンドを実行し、標準出力を返します。
	// env は追加で設定する環境変数（"KEY=VALUE"）で、資格情報を引数に載せず渡す用途に使います。
	// 標準エラーは os.Stderr へ流します。コマンドが非ゼロ終了した場合はエラーを返します。
	Output(ctx context.Context, dir string, env []string, name string, args []string) ([]byte, error)
}

// OS は、os/exec を用いた Runner の実装です。
type OS struct{}

func (OS) Output(ctx context.Context, dir string, env []string, name string, args []string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := osexec.CommandContext(ctx, name, args...) //nolint:gosec // name/args は呼び出し側で固定された信頼値
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}
