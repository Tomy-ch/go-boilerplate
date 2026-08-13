// Package shellcheck は shellcheck の起動と結果の解釈を 1 箇所へ集める。
//
// 呼び出し側は 2 つある。composite action から抽出した run ステップを検査する actions-shellcheck と、
// ファイルとして置かれたシェルを検査する shell-lint である。どちらも渡し方が同じでなければ、同じ
// スクリプトが経路によって違う判定を受けうる。
//
// 検査対象は常に標準入力から渡す。コマンドの引数を定数に保つためで、可変引数は gosec の G204 に当たる。
// 方言は内容の shebang から shellcheck 自身が決めるので、渡し方は結果を変えない。
package shellcheck

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go-boilerplate/pkg/xerrors"
)

const (
	// Binary は起動する実行ファイル名。所在確認も同じ名前で行う。
	Binary = "shellcheck"

	// findingsExitCode は「指摘があった」ことを表す終了コード。失敗ではなく結果である。
	findingsExitCode = 1

	timeout = 30 * time.Second
)

// ErrRun は shellcheck を起動できなかった、または指摘以外の理由で終了したことを表す。
var ErrRun = xerrors.New("shellcheck の実行に失敗しました")

// Run は script を shellcheck に掛け、gcc 形式の出力を返します。
//
// 指摘があるときの終了コード 1 は失敗ではなく結果なので、そのまま出力を返します。区別しないと
// 「指摘があった」と「起動できなかった」が同じ扱いになり、後者が黙って緑になります。
func Run(ctx context.Context, script string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, Binary, "--norc", "--format=gcc", "-")
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}

	var exitErr *exec.ExitError
	if xerrors.As(err, &exitErr) && exitErr.ExitCode() == findingsExitCode {
		return stdout.String(), nil
	}

	return "", xerrors.Wrap(ErrRun, fmt.Sprintf("%v: %s", err, strings.TrimSpace(stderr.String())))
}
