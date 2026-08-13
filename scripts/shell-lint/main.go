// Package main はリポジトリ内の `*.sh` を shellcheck で検査する。
//
// composite action の中のシェルは actions-shellcheck が見るが、ファイルとして置かれたシェルは
// どのゲートにも掛かっていなかった。フックやセットアップの入口はそこに居る —— PreToolUse フック、
// SessionStart フック、スキル同梱スクリプト、コンテナの init。いずれも編集のたび、あるいは
// セッションのたびに走るのに、壊れても CI は緑を返していた。
//
// 内容をそのまま渡すので、指摘の行・列は写し戻さずに使える（起動の詳細は scripts/lib/shellcheck）。
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go-boilerplate/pkg/xerrors"
	"go-boilerplate/scripts/lib/shellcheck"
)

const shellSuffix = ".sh"

// 走査から外すディレクトリ名。依存の取得物と VCS の内部で、いずれも我々が書いたものではない。
var skippedDirs = []string{".git", "node_modules", "vendor", "tmp"}

var (
	errShellcheckMissing = xerrors.New("shellcheck が見つかりません")
	errFindings          = xerrors.New("shellcheck が指摘を検出しました")
)

func main() {
	log.SetFlags(0)

	if err := run(context.Background(), os.Getwd, exec.LookPath); err != nil {
		log.Fatalf("❌ %v", err)
	}
}

// run はリポジトリ内のシェルスクリプトを shellcheck に掛け、結果を報告します。
// wd は走査の基点となるディレクトリの取得手段、lookPath は shellcheck の所在確認手段です。
func run(ctx context.Context, wd func() (string, error), lookPath func(string) (string, error)) error {
	if _, err := lookPath(shellcheck.Binary); err != nil {
		return xerrors.Wrap(errShellcheckMissing, err.Error())
	}

	root, err := wd()
	if err != nil {
		return xerrors.Wrap(err, "getwd")
	}

	scripts, err := shellScripts(root)
	if err != nil {
		return err
	}

	var findings []string
	for _, script := range scripts {
		// script は shellScripts が root 配下を walk して得た相対パスだけを取る。
		body, err := os.ReadFile(filepath.Join(root, script)) //nolint:gosec // G304: 走査結果のみ
		if err != nil {
			return xerrors.Wrap(err, "read")
		}

		out, err := shellcheck.Run(ctx, string(body))
		if err != nil {
			return err
		}
		findings = append(findings, prefixFindings(script, out)...)
	}

	if len(findings) > 0 {
		log.Print(strings.Join(findings, "\n"))
		return xerrors.Wrap(errFindings, fmt.Sprintf("%d 件", len(findings)))
	}

	log.Printf("✅ シェルスクリプト %d ファイルを shellcheck で検査しました", len(scripts))

	return nil
}

// shellScripts は root 配下の `*.sh` をリポジトリ相対パスで昇順に返します。
func shellScripts(root string) ([]string, error) {
	var scripts []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if slices.Contains(skippedDirs, entry.Name()) {
				return fs.SkipDir
			}

			return nil
		}
		if !strings.HasSuffix(entry.Name(), shellSuffix) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return xerrors.Wrap(err, "rel")
		}
		scripts = append(scripts, filepath.ToSlash(rel))

		return nil
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "walk")
	}

	slices.Sort(scripts)

	return scripts, nil
}

// prefixFindings は shellcheck の出力を 1 行 1 指摘へ整え、先頭をリポジトリ相対パスへ差し替えます。
// stdin で渡しているため shellcheck 自身は入力を `-` としか呼べず、どのファイルの指摘か言えません。
func prefixFindings(script, out string) []string {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil
	}

	var findings []string
	for line := range strings.SplitSeq(trimmed, "\n") {
		_, rest, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		findings = append(findings, script+":"+rest)
	}

	return findings
}
