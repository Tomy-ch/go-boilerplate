// Package main は composite action（.github/actions/**）の runs.steps[].run を shellcheck で検査し、
// 指摘を action 定義ファイルの行・列へ写し戻す。
//
// actionlint は .github/workflows しか走査せず、action 定義を直接渡すと workflow として解釈して
// 構文エラーで落ちるため、composite action の中のシェルはどのゲートにも掛からない。
//
// 抽出したステップ数は、YAML をそのままデコードして数えた件数とファイル単位で突き合わせる。
// 経路が分かれるため、片方が壊れれば件数差として現れ、検査範囲が黙って縮んだまま緑になることを防ぐ。
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go-boilerplate/pkg/xerrors"

	"gopkg.in/yaml.v3"
)

const (
	actionsDir     = ".github/actions"
	compositeUsing = "composite"
	mergeKey       = "<<"
	envCommand     = "env"

	shellcheckBin     = "shellcheck"
	findingsExitCode  = 1
	shellcheckTimeout = 30 * time.Second

	shebangLines        = 1
	blockScalarKeyLines = 1
	firstBodyIndex      = 1
	firstColumn         = 1

	exprOpen        = "${{"
	exprClose       = "}}"
	exprPlaceholder = "GH_EXPR"
)

var (
	actionFileNames = []string{"action.yml", "action.yaml"}

	findingRe = regexp.MustCompile(`^[^:]*:(\d+):(\d+):(.*)$`)

	shebangs = map[string]string{
		"bash": "#!/usr/bin/env bash",
		"sh":   "#!/bin/sh",
	}
)

var (
	errNoShell           = xerrors.New("composite の run ステップに shell 指定がありません")
	errShellcheck        = xerrors.New("shellcheck の実行に失敗しました")
	errUnterminatedExpr  = xerrors.New("閉じていない ${{ があります")
	errStepCountMismatch = xerrors.New("run ステップの抽出数が YAML のデコード結果と一致しません")
	errStepsNotSequence  = xerrors.New("runs.steps がリストとして読めません")
	errFoldedRun         = xerrors.New("run にブロック折り畳み（>）は使えません。リテラル（|）で書いてください")
	errShellcheckMissing = xerrors.New("shellcheck が PATH にありません（mise install shellcheck）")
)

type step struct {
	file      string
	shell     string
	script    string
	firstLine int
	colBase   int
}

type result struct {
	checked  int
	skipped  []string
	findings []string
}

func main() {
	log.SetFlags(0)
	if _, err := exec.LookPath(shellcheckBin); err != nil {
		log.Fatalf("❌ %v: %v", errShellcheckMissing, err)
	}
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ getwd: %v", err)
	}
	files, steps, err := collectSteps(os.DirFS(root))
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	res, err := check(context.Background(), steps)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	for _, s := range res.skipped {
		log.Printf("  ⏭️ %s", s)
	}
	for _, f := range res.findings {
		log.Print(f)
	}
	if len(res.findings) > 0 {
		log.Fatalf("❌ composite action の run に %d 件の指摘があります（検査 %d ステップ）", len(res.findings), res.checked)
	}
	log.Printf("✅ composite action %d ファイルの run を %d ステップ検査しました（対象外 shell: %d ステップ）",
		len(files), res.checked, len(res.skipped))
}

func collectSteps(fsys fs.FS) ([]string, []step, error) {
	files, err := actionFiles(fsys)
	if err != nil {
		return nil, nil, err
	}
	var steps []step
	for _, f := range files {
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, nil, xerrors.Wrap(err, "read "+f)
		}
		s, err := parseAction(f, data)
		if err != nil {
			return nil, nil, err
		}
		steps = append(steps, s...)
	}
	return files, steps, nil
}

func actionFiles(fsys fs.FS) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, actionsDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if xerrors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll
			}
			return err
		}
		if !entry.IsDir() && isActionFile(entry.Name()) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "walk "+actionsDir)
	}
	sort.Strings(files)
	return files, nil
}

func isActionFile(name string) bool {
	return slices.Contains(actionFileNames, name)
}

func parseAction(file string, data []byte) ([]step, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, xerrors.Wrap(err, "parse "+file)
	}
	want, err := countRunSteps(file, data)
	if err != nil {
		return nil, err
	}
	steps, err := extractSteps(file, data, &doc)
	if err != nil {
		return nil, err
	}
	if len(steps) != want {
		return nil, xerrors.Wrap(errStepCountMismatch, fmt.Sprintf("%s: 抽出 %d / 期待 %d", file, len(steps), want))
	}
	return steps, nil
}

// countRunSteps はデコード結果から run ステップ数を数える。件数を using の値と無関係に数えるのは、
// 綴りを取り違えた action（using: Composite など）を「対象外」に寄せると、run を持つのに 1 件も
// 検査されない状態が緑で通るため。
func countRunSteps(file string, data []byte) (int, error) {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0, xerrors.Wrap(err, "decode "+file)
	}
	steps := fieldValue(fieldValue(doc, "runs"), "steps")
	if steps == nil {
		return 0, nil
	}
	// リストとして読めない形を「対象外」に寄せると、検査範囲が黙って縮んだまま緑になる。
	list, ok := steps.([]any)
	if !ok {
		return 0, xerrors.Wrap(errStepsNotSequence, file)
	}
	count := 0
	for _, item := range list {
		if _, ok := fieldValue(item, "run").(string); ok {
			count++
		}
	}
	return count, nil
}

func fieldValue(node any, key string) any {
	fields, ok := node.(map[string]any)
	if !ok {
		return nil
	}
	return fields[key]
}

func extractSteps(file string, data []byte, doc *yaml.Node) ([]step, error) {
	runs := mapValue(documentRoot(doc), "runs")
	if using := mapValue(runs, "using"); using == nil || using.Value != compositeUsing {
		return nil, nil
	}
	stepsNode := mapValue(runs, "steps")
	if stepsNode == nil {
		return nil, nil
	}
	var steps []step
	for _, node := range stepsNode.Content {
		run := mapValue(node, "run")
		if run == nil {
			continue
		}
		// ブロック折り畳み（>）は隣接する非空行を空白へ畳むため、本文の行と action 定義ファイルの行が
		// 1 対 1 で対応しない。指摘の位置を写し戻せないうえ、畳まれた行がソースに無い構文を作って
		// 誤検知も生むため受け付けない。
		if run.Style == yaml.FoldedStyle {
			return nil, xerrors.Wrap(errFoldedRun, fmt.Sprintf("%s:%d", file, run.Line))
		}
		shell := mapValue(node, "shell")
		if shell == nil {
			return nil, xerrors.Wrap(errNoShell, fmt.Sprintf("%s:%d", file, run.Line))
		}
		firstLine := bodyFirstLine(run)
		steps = append(steps, step{
			file:      file,
			shell:     shell.Value,
			script:    run.Value,
			firstLine: firstLine,
			colBase:   bodyColumnBase(data, run, firstLine),
		})
	}
	return steps, nil
}

func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	return doc.Content[0]
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var merged *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case key:
			return resolveAlias(node.Content[i+1])
		case mergeKey:
			merged = node.Content[i+1]
		}
	}
	return mergeValue(merged, key)
}

func mergeValue(merged *yaml.Node, key string) *yaml.Node {
	merged = resolveAlias(merged)
	if merged == nil {
		return nil
	}
	if merged.Kind != yaml.SequenceNode {
		return mapValue(merged, key)
	}
	for _, node := range merged.Content {
		if v := mapValue(node, key); v != nil {
			return v
		}
	}
	return nil
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

func bodyFirstLine(run *yaml.Node) int {
	if run.Style == yaml.LiteralStyle {
		return run.Line + blockScalarKeyLines
	}
	return run.Line
}

func bodyColumnBase(data []byte, run *yaml.Node, firstLine int) int {
	switch run.Style {
	case yaml.LiteralStyle:
		// ブロックスカラーの本文はインデントを剥がした形で得られるため、剥がされた幅を足し戻す。
		return blockIndentWidth(data, firstLine)
	case yaml.SingleQuotedStyle, yaml.DoubleQuotedStyle:
		// 引用符付きスカラーは範囲の先頭が開き引用符を指すため、その 1 文字分を読み飛ばす。
		return run.Column
	default:
		return run.Column - firstColumn
	}
}

// blockIndentWidth はブロックスカラーのインデント幅を返す。幅を決めるのは最初の非空行（YAML の規則）で、
// 本文が空行で始まる場合にその行の幅 0 を採ると、そのステップの全指摘の列がずれる。
func blockIndentWidth(data []byte, firstLine int) int {
	lines := strings.Split(string(data), "\n")
	if firstLine < firstBodyIndex || firstLine > len(lines) {
		return 0
	}
	for _, body := range lines[firstLine-firstBodyIndex:] {
		trimmed := strings.TrimLeft(body, " \t")
		if trimmed == "" {
			continue
		}
		return len(body) - len(trimmed)
	}
	return 0
}

func check(ctx context.Context, steps []step) (result, error) {
	var res result
	for _, s := range steps {
		shebang, ok := shebangs[shellDialect(s.shell)]
		if !ok {
			res.skipped = append(res.skipped,
				fmt.Sprintf("%s:%d: shell=%q は shellcheck の対象外のため検査しません", s.file, s.firstLine, s.shell))
			continue
		}
		script, err := maskExpressions(s.script)
		if err != nil {
			return result{}, xerrors.Wrap(err, fmt.Sprintf("%s:%d", s.file, s.firstLine))
		}
		out, err := runShellcheck(ctx, shebang, script)
		if err != nil {
			return result{}, err
		}
		res.checked++
		res.findings = append(res.findings, remapFindings(s, out)...)
	}
	return res, nil
}

// shellDialect は shell 指定から shellcheck へ渡す方言名を取り出す。env 経由の指定（shell: env FOO=bar bash）
// では KEY=VALUE 形の変数代入がインタプリタ名の手前に並ぶため、それらを読み飛ばして最初のコマンドを採る。
func shellDialect(shell string) string {
	if strings.Contains(shell, exprOpen) {
		return ""
	}
	for field := range strings.FieldsSeq(shell) {
		if fieldBase(field) == envCommand || isAssignment(field) {
			continue
		}
		return fieldBase(field)
	}
	return ""
}

func isAssignment(field string) bool {
	name, _, ok := strings.Cut(field, "=")
	if !ok || name == "" {
		return false
	}
	for i, r := range name {
		if r == '_' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r)) {
			continue
		}
		return false
	}
	return true
}

func fieldBase(cmd string) string {
	if i := strings.LastIndex(cmd, "/"); i >= 0 {
		return cmd[i+1:]
	}
	return cmd
}

func maskExpressions(script string) (string, error) {
	var masked strings.Builder
	for {
		open := strings.Index(script, exprOpen)
		if open < 0 {
			masked.WriteString(script)
			return masked.String(), nil
		}
		masked.WriteString(script[:open])
		rest := script[open+len(exprOpen):]
		end := exprEnd(rest)
		if end < 0 {
			return "", errUnterminatedExpr
		}
		masked.WriteString(exprPlaceholder)
		masked.WriteString(strings.Repeat("\n", strings.Count(rest[:end], "\n")))
		script = rest[end+len(exprClose):]
	}
}

// exprEnd は式の開始直後から見た閉じ位置を返す。閉じが無ければ -1。
//
// 式の中の ' は文字列リテラルの境界なので、その内側の }} で式を打ち切らないよう引用符を追う。
// ただし ' が奇数個だと引用符の状態が戻らず、自分の }} を読み飛ばして後続の別の式まで
// 巻き込む。GitHub Expressions の式は入れ子にならないため、閉じを見つける前に次の ${{ が
// 来ることは引用符の対応が壊れている証拠であり、そこで未閉じとして扱う。
// 巻き込んだ区間は maskExpressions が空行へ潰すので、fail-open にするとその間のシェルが
// 検査対象から丸ごと消える。
func exprEnd(expr string) int {
	quoted := false
	for i := range len(expr) {
		switch {
		case strings.HasPrefix(expr[i:], exprOpen):
			return -1
		case expr[i] == '\'':
			quoted = !quoted
		case !quoted && strings.HasPrefix(expr[i:], exprClose):
			return i
		}
	}
	return -1
}

func runShellcheck(ctx context.Context, shebang, script string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, shellcheckTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, shellcheckBin, "--norc", "--format=gcc", "-")
	cmd.Stdin = strings.NewReader(shebang + "\n" + script)
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
	return "", xerrors.Wrap(errShellcheck, fmt.Sprintf("%v: %s", err, strings.TrimSpace(stderr.String())))
}

func remapFindings(s step, out string) []string {
	lineBase := s.firstLine - shebangLines - firstBodyIndex
	var findings []string
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		m := findingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		row, rowErr := strconv.Atoi(m[1])
		col, colErr := strconv.Atoi(m[2])
		if rowErr != nil || colErr != nil {
			continue
		}
		findings = append(findings, fmt.Sprintf("  %s:%d:%d:%s", s.file, lineBase+row, col+s.colBase, m[3]))
	}
	return findings
}
