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

	"go-boilerplate/pkg/xerrors"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

var (
	wordSepRe   = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	qualifierRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*\.`)
)

var (
	// errMissingNameOrType は、name / type のいずれかが未指定の場合のエラーです。
	errMissingNameOrType = xerrors.New("name and type are required")
	// errInvalidName は、名前から識別子を組み立てられなかった場合のエラーです。
	errInvalidName = xerrors.New("invalid name")
	// errInvalidIdentifier は、組み立てた識別子が Go の識別子として不正な場合のエラーです。
	errInvalidIdentifier = xerrors.New("invalid identifier")
	// errQualifierAliasMismatch は、型の修飾子と import alias が食い違う場合のエラーです。
	errQualifierAliasMismatch = xerrors.New("type qualifier does not match alias")
)

//go:embed template_ctx.tpl
var ctxTpl string

//go:embed template_test.tpl
var testTpl string

type Param struct {
	NameLower        string
	NameCamel        string
	Type             string
	ImportPath       string
	ImportAlias      string
	TestSuccessValue string
	TestFailValue    string
}

func GenerateCtxKey(name, typ, importPath, importAlias, outDir, testValue string) error {
	if name == "" || typ == "" {
		return errMissingNameOrType
	}

	outDir = resolveOutDir(outDir)

	lower, err := toIdentifierLower(name)
	if err != nil {
		return err
	}

	camel, err := toExportedName(name)
	if err != nil {
		return err
	}

	alias, err := resolveImportAlias(typ, importPath, importAlias)
	if err != nil {
		return err
	}
	importAlias = alias

	typeExpr := typ

	success, fail := resolveTestValue(typeExpr, lower, testValue)

	p := Param{
		NameLower:        lower,
		NameCamel:        camel,
		Type:             typeExpr,
		ImportPath:       importPath,
		ImportAlias:      importAlias,
		TestSuccessValue: success,
		TestFailValue:    fail,
	}

	if err := os.MkdirAll(outDir, dirPerm); err != nil {
		return xerrors.Wrap(err, "mkdir "+outDir)
	}

	if err := writeFile(filepath.Join(outDir, p.NameLower+"_ctx.gen.go"), ctxTpl, p); err != nil {
		return err
	}

	if err := writeFile(filepath.Join(outDir, p.NameLower+"_ctx_test.go"), testTpl, p); err != nil {
		return err
	}

	return nil
}

// resolveOutDir は、出力先ディレクトリを正規化します。
// 空文字はカレントディレクトリ（"."）を指します。
func resolveOutDir(outDir string) string {
	return filepath.Clean(outDir)
}

func writeFile(path, tpl string, p Param) error {
	t := template.Must(template.New("").Parse(tpl))

	var buf bytes.Buffer
	if err := t.Execute(&buf, p); err != nil {
		return xerrors.Wrap(err, "execute template "+path)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return xerrors.Wrap(err, "format "+path)
	}

	if existing, err := os.ReadFile(path); err == nil { //nolint:gosec // generator writes/reads controlled path
		if bytes.Equal(existing, src) {
			return nil
		}
	}

	if err := os.WriteFile(path, src, filePerm); err != nil {
		return xerrors.Wrap(err, "write "+path)
	}
	return nil
}

// splitWords は、非英数字区切りで語に分割し、空要素を除去して返します。
func splitWords(s string) []string {
	parts := wordSepRe.Split(s, -1)
	words := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		words = append(words, p)
	}
	return words
}

func toExportedName(s string) (string, error) {
	var sb strings.Builder
	for _, p := range splitWords(s) {
		runes := []rune(p)
		sb.WriteString(strings.ToUpper(string(runes[0])) + string(runes[1:]))
	}
	out := sb.String()

	if out == "" {
		return "", xerrors.Wrap(errInvalidName, s)
	}

	if !isValidIdentifier(out) {
		return "", xerrors.Wrap(errInvalidIdentifier, out)
	}

	return out, nil
}

func toIdentifierLower(s string) (string, error) {
	var sb strings.Builder
	for _, p := range splitWords(s) {
		sb.WriteString(strings.ToLower(p))
	}
	out := sb.String()
	if out == "" {
		return "", xerrors.Wrap(errInvalidName, s)
	}
	runes := []rune(out)
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		out = "x" + out
	}
	if !isValidIdentifier(out) {
		return "", xerrors.Wrap(errInvalidIdentifier, out)
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

	if importAlias == "" {
		if qualifier != "" {
			return qualifier, nil
		}
		return sanitizeAlias(lastSegment(importPath)), nil
	}

	if qualifier != "" && qualifier != importAlias {
		return "", xerrors.Wrap(errQualifierAliasMismatch, fmt.Sprintf("qualifier=%s alias=%s", qualifier, importAlias))
	}

	return importAlias, nil
}

func extractQualifier(t string) string {
	// find last occurrence of identifier followed by '.'
	matches := qualifierRe.FindAllString(t, -1)
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
		// override があれば success に採用し、無ければ zero 値（*new(T)）を success/fail 双方に用いる。
		fail := "*new(" + t + ")"
		if override != "" {
			return override, fail
		}
		return fail, fail
	}
}
