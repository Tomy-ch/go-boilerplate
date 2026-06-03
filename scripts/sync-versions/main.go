// scripts/sync-versions は mise.toml [tools] の go / node / python を SSOT として、
// 以下のファイルへ伝播する Go 製の小さなツールです。
//
//   - go.mod                    `go X.Y.Z` directive
//   - docker/server/Dockerfile  `FROM golang:X.Y.Z-alpine`
//   - docker/tools/Dockerfile   `FROM golang:X.Y.Z-alpine`
//     `FROM node:X.Y-alpine`
//     `FROM python:X.Y.Z-slim`
//
// 設計:
//   - mise.toml は `[tools]` table 配下の `go` / `node` / `python` のみを参照する
//     行ベース parser を自前で持つ（外部依存ゼロ、production binary への影響なし）。
//   - 各 rule の事前 validate を全て通してから書き出すため partial state を残さない。
//   - 期待マッチ数（expectedCount）未満の rule が1つでもあれば、一切書かずに非ゼロ終了。
package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// filePerm は既存設定ファイルと同じ 0o644 で上書きする。
	filePerm fs.FileMode = 0o644
	// fromRegexpGroups は FROM タグ書き換え regex が持つ capture group 数 + 1（全体マッチ含む）。
	fromRegexpGroups = 3
	// minServerGolangStages は docker/server/Dockerfile 内の `FROM golang:` 行の最低期待数。
	minServerGolangStages = 2
)

var (
	miseSectionRe = regexp.MustCompile(`^\[([^\]]+)\]`)
	miseKeyRe     = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)\s*=\s*"([^"]+)"`)
	goModRe       = regexp.MustCompile(`(?m)^go \d+(?:\.\d+){0,2}$`)
	golangFromRe  = regexp.MustCompile(`(FROM\s+golang:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	nodeFromRe    = regexp.MustCompile(`(FROM\s+node:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
	pythonFromRe  = regexp.MustCompile(`(FROM\s+python:)\d+(?:\.\d+){0,2}(-[\w.-]+)`)
)

// runtimeVersions は mise.toml [tools] から抽出した3ランタイムのバージョン文字列。
type runtimeVersions struct {
	Go     string
	Node   string
	Python string
}

// rule はあるファイルの正規表現マッチ箇所を一つの version で置換する単位。
type rule struct {
	label         string
	file          string
	re            *regexp.Regexp
	version       string
	replace       func(match string) string
	expectedCount int
}

// fileState は同一ファイルへの複数 rule を1回の書き出しにまとめるための中間状態。
type fileState struct {
	original string
	current  string
	applied  []string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ getwd: %v", err)
	}

	v, err := parseMiseTOML(filepath.Join(root, "mise.toml"))
	if err != nil {
		log.Fatalf("❌ mise.toml のパースに失敗: %v", err)
	}

	printSource(v)

	rules := buildRules(v)

	if errs := validateRules(rules, root); len(errs) > 0 {
		reportAndExit("Validation errors（書き換えは行いません）", errs)
	}

	states, errs := computeChanges(rules, root)
	if len(errs) > 0 {
		reportAndExit("Match-count errors（書き換えは行いません）", errs)
	}

	if err := writeChanges(states, root); err != nil {
		log.Fatalf("❌ %v", err)
	}
}

// parseMiseTOML は mise.toml の [tools] table 配下の go/node/python キーだけを抽出する
// 用途特化の最小 parser。完全な TOML 仕様には準拠しない。
func parseMiseTOML(path string) (runtimeVersions, error) {
	var v runtimeVersions
	f, err := os.Open(path) //nolint:gosec // path is constructed from cwd + literal filename
	if err != nil {
		return v, fmt.Errorf("open: %w", err)
	}
	defer func() { _ = f.Close() }()

	currentSection := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := miseSectionRe.FindStringSubmatch(line); m != nil {
			currentSection = m[1]
			continue
		}
		if currentSection != "tools" {
			continue
		}
		m := miseKeyRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch m[1] {
		case "go":
			v.Go = m[2]
		case "node":
			v.Node = m[2]
		case "python":
			v.Python = m[2]
		}
	}
	if err := scanner.Err(); err != nil {
		return v, fmt.Errorf("scan: %w", err)
	}
	return v, nil
}

func printSource(v runtimeVersions) {
	log.Println("Source: mise.toml")
	log.Printf("  go     = %s", emptyAs(v.Go, "(unset)"))
	log.Printf("  node   = %s", emptyAs(v.Node, "(unset)"))
	log.Printf("  python = %s", emptyAs(v.Python, "(unset)"))
}

func buildRules(v runtimeVersions) []rule {
	return []rule{
		{
			label:         "go.mod (go directive)",
			file:          "go.mod",
			re:            goModRe,
			version:       v.Go,
			replace:       func(string) string { return "go " + v.Go },
			expectedCount: 1,
		},
		{
			label:         "docker/server/Dockerfile (golang base)",
			file:          "docker/server/Dockerfile",
			re:            golangFromRe,
			version:       v.Go,
			replace:       fromReplacer(golangFromRe, v.Go),
			expectedCount: minServerGolangStages,
		},
		{
			label:         "docker/tools/Dockerfile (golang base)",
			file:          "docker/tools/Dockerfile",
			re:            golangFromRe,
			version:       v.Go,
			replace:       fromReplacer(golangFromRe, v.Go),
			expectedCount: 1,
		},
		{
			label:         "docker/tools/Dockerfile (node base)",
			file:          "docker/tools/Dockerfile",
			re:            nodeFromRe,
			version:       v.Node,
			replace:       fromReplacer(nodeFromRe, v.Node),
			expectedCount: 1,
		},
		{
			label:         "docker/tools/Dockerfile (python base)",
			file:          "docker/tools/Dockerfile",
			re:            pythonFromRe,
			version:       v.Python,
			replace:       fromReplacer(pythonFromRe, v.Python),
			expectedCount: 1,
		},
	}
}

func fromReplacer(re *regexp.Regexp, version string) func(string) string {
	return func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < fromRegexpGroups {
			return match
		}
		return sub[1] + version + sub[2]
	}
}

// validateRules は version 未設定と file 不在を事前検査する。
func validateRules(rules []rule, root string) []string {
	var errs []string
	for _, r := range rules {
		if r.version == "" {
			errs = append(errs, fmt.Sprintf(
				"%s: mise.toml [tools] に対応する key が未設定", r.label))
		}
	}
	for _, r := range rules {
		if _, err := os.Stat(filepath.Join(root, r.file)); err != nil {
			errs = append(errs, fmt.Sprintf(
				"%s: %s が見つかりません", r.label, r.file))
		}
	}
	return errs
}

// computeChanges は実書き出しを行わずに、各 file の最終内容を計算する。
// 期待マッチ数未満の rule があれば該当 rule のエラーを返す。
func computeChanges(rules []rule, root string) (map[string]*fileState, []string) {
	states := map[string]*fileState{}
	var errs []string
	for _, r := range rules {
		st, ok := states[r.file]
		if !ok {
			data, err := os.ReadFile(filepath.Join(root, r.file)) //nolint:gosec // path from cwd + rule.file
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: read %s: %v", r.label, r.file, err))
				continue
			}
			st = &fileState{original: string(data), current: string(data)}
			states[r.file] = st
		}
		matches := r.re.FindAllStringIndex(st.current, -1)
		if len(matches) < r.expectedCount {
			errs = append(errs, fmt.Sprintf(
				"%s: マッチ件数 %d が期待値 %d 未満（regex: %s）",
				r.label, len(matches), r.expectedCount, r.re))
			continue
		}
		st.current = r.re.ReplaceAllStringFunc(st.current, r.replace)
		st.applied = append(st.applied, fmt.Sprintf("%s x%d", r.label, len(matches)))
	}
	return states, errs
}

// writeChanges は事前 validate を通過した状態を file 単位で1回ずつ書き出す。
func writeChanges(states map[string]*fileState, root string) error {
	anyChange := false
	for file, st := range states {
		if st.original == st.current {
			log.Printf("No change: %s", file)
			continue
		}
		if err := os.WriteFile(
			filepath.Join(root, file), []byte(st.current), filePerm,
		); err != nil {
			return fmt.Errorf("write %s: %w", file, err)
		}
		log.Printf("Updated: %s [%s]", file, strings.Join(st.applied, ", "))
		anyChange = true
	}
	if anyChange {
		log.Println("Done.")
	} else {
		log.Println("Done (no changes).")
	}
	return nil
}

func reportAndExit(title string, errs []string) {
	log.Println("")
	log.Printf("❌ %s:", title)
	for _, e := range errs {
		log.Printf("  - %s", e)
	}
	os.Exit(1)
}

func emptyAs(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
