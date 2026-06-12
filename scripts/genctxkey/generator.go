package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	_ "embed"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

//go:embed template_ctx.tpl
var ctxTpl string

//go:embed template_test.tpl
var testTpl string

type Param struct {
	NameLower        string
	NameFlat         string
	NameCamel        string
	Type             string
	ImportPath       string
	ImportAlias      string
	TestSuccessValue string
	TestFailValue    string
}

func GenerateCtxKey(name, typ, importPath, importAlias, outDir, testValue string) error {
	if name == "" || typ == "" {
		return fmt.Errorf("name and type are required")
	}

	if outDir == "" {
		outDir = "."
	}
	outDir = filepath.Clean(outDir)

	camel, err := toExportedName(name)
	if err != nil {
		return err
	}

	lower, err := toIdentifierLower(name)
	if err != nil {
		return err
	}

	// import handling with qualifier alignment
	alias, err := resolveImportAlias(typ, importPath, importAlias)
	if err != nil {
		return err
	}
	importAlias = alias

	typeExpr := typ

	success, fail := resolveTestValue(typeExpr, lower, testValue)

	p := Param{
		NameLower:        lower,
		NameFlat:         lower,
		NameCamel:        camel,
		Type:             typeExpr,
		ImportPath:       importPath,
		ImportAlias:      importAlias,
		TestSuccessValue: success,
		TestFailValue:    fail,
	}

	if err := os.MkdirAll(outDir, dirPerm); err != nil {
		return err
	}

	if err := writeFile(filepath.Join(outDir, p.NameFlat+"_ctx.gen.go"), ctxTpl, p); err != nil {
		return err
	}

	if err := writeFile(filepath.Join(outDir, p.NameFlat+"_ctx_test.go"), testTpl, p); err != nil {
		return err
	}

	return nil
}

func writeFile(path, tpl string, p Param) error {
	t := template.Must(template.New("").Parse(tpl))

	var buf bytes.Buffer
	if err := t.Execute(&buf, p); err != nil {
		return err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	if existing, err := os.ReadFile(path); err == nil { //nolint:gosec // generator writes/reads controlled path
		if bytes.Equal(existing, src) {
			return nil
		}
	}

	return os.WriteFile(path, src, filePerm)
}

func toExportedName(s string) (string, error) {
	parts := regexp.MustCompile(`[^\p{L}\p{N}]+`).Split(s, -1)

	var out string
	for _, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		out += strings.ToUpper(string(runes[0])) + string(runes[1:])
	}

	if out == "" {
		return "", fmt.Errorf("invalid name: %s", s)
	}

	if !isValidIdentifier(out) {
		return "", fmt.Errorf("invalid identifier: %s", out)
	}

	return out, nil
}

func toIdentifierLower(s string) (string, error) {
	// split on non-alnum, join, and lower
	parts := regexp.MustCompile(`[^\p{L}\p{N}]+`).Split(s, -1)
	var out string
	for _, p := range parts {
		if p == "" {
			continue
		}
		out += strings.ToLower(p)
	}
	if out == "" {
		return "", fmt.Errorf("invalid name: %s", s)
	}
	// ensure starts with a letter or '_'
	runes := []rune(out)
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		out = "x" + out
	}
	if !isValidIdentifier(out) {
		return "", fmt.Errorf("invalid identifier: %s", out)
	}
	return out, nil
}

func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}
	return true
}

func sanitizeAlias(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")

	if !isValidIdentifier(s) {
		return "pkg"
	}

	if token.Lookup(s).IsKeyword() {
		return "pkg"
	}

	return s
}

func resolveImportAlias(typ, importPath, importAlias string) (string, error) {
	if importPath == "" {
		return importAlias, nil
	}

	qualifier := extractQualifier(typ)

	// No alias provided: prefer qualifier, fallback to last segment
	if importAlias == "" {
		if qualifier != "" {
			return qualifier, nil
		}
		return sanitizeAlias(lastSegment(importPath)), nil
	}

	// Alias provided: validate against qualifier when present
	if qualifier != "" && qualifier != importAlias {
		return "", fmt.Errorf("type qualifier (%s) does not match alias (%s)", qualifier, importAlias)
	}

	return importAlias, nil
}

func extractQualifier(t string) string {
	// find last occurrence of identifier followed by '.'
	re := regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*\.`)
	matches := re.FindAllString(t, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	return strings.TrimSuffix(last, ".")
}

func lastSegment(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func resolveTestValue(t, nameLower, override string) (string, string) {
	switch t {
	case "string":
		return `"test-` + nameLower + `"`, `""`
	case "int":
		return "123", "0"
	case "bool":
		return "true", "false"
	default:
		// 任意型は意味ある success 値を自動合成できない。
		// 呼び出し側が -test-value を渡せば success に採用し、未指定なら従来通り zero 値にフォールバックする。
		fail := "*new(" + t + ")"
		if override != "" {
			return override, fail
		}
		return fail, fail
	}
}
