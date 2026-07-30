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

	runKeyRe  = regexp.MustCompile(`(?m)^[ \t]*-?[ \t]*run:`)
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
	errExtractorBroken   = xerrors.New("run: を持つ composite action があるのに 1 ステップも抽出できませんでした")
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
	var (
		steps      []step
		hasRunText bool
	)
	for _, f := range files {
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, nil, xerrors.Wrap(err, "read "+f)
		}
		s, err := parseAction(f, data)
		if err != nil {
			return nil, nil, err
		}
		if len(s) == 0 && runKeyRe.Match(data) && isComposite(data) {
			hasRunText = true
		}
		steps = append(steps, s...)
	}
	if len(steps) == 0 && hasRunText {
		return nil, nil, errExtractorBroken
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

func isComposite(data []byte) bool {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false
	}
	using := mapValue(mapValue(documentRoot(&doc), "runs"), "using")
	return using != nil && using.Value == compositeUsing
}

func parseAction(file string, data []byte) ([]step, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, xerrors.Wrap(err, "parse "+file)
	}
	runs := mapValue(documentRoot(&doc), "runs")
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

func isBlockScalar(run *yaml.Node) bool {
	return run.Style == yaml.LiteralStyle || run.Style == yaml.FoldedStyle
}

func bodyFirstLine(run *yaml.Node) int {
	if isBlockScalar(run) {
		return run.Line + blockScalarKeyLines
	}
	return run.Line
}

func bodyColumnBase(data []byte, run *yaml.Node, firstLine int) int {
	if !isBlockScalar(run) {
		return run.Column - firstColumn
	}
	lines := strings.Split(string(data), "\n")
	if firstLine < firstBodyIndex || firstLine > len(lines) {
		return 0
	}
	body := lines[firstLine-firstBodyIndex]
	return len(body) - len(strings.TrimLeft(body, " \t"))
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

func shellDialect(shell string) string {
	if strings.Contains(shell, exprOpen) {
		return ""
	}
	for field := range strings.FieldsSeq(shell) {
		if name := fieldBase(field); name != envCommand {
			return name
		}
	}
	return ""
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

func exprEnd(expr string) int {
	quoted := false
	for i := range len(expr) {
		switch {
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
