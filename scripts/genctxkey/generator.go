package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"
)

const (
	dirPerm  = 0o750
	filePerm = 0o600
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

func GenerateCtxKey(name, typ, outDir string) error {
	if name == "" || typ == "" {
		return fmt.Errorf("name and type are required")
	}

	if outDir == "" {
		outDir = "."
	}
	outDir = filepath.Clean(outDir)

	lower := strings.ToLower(name)

	var importPath string
	var importAlias string
	typeName := typ

	if strings.Contains(typ, "/") {
		// expect format: github.com/foo/bar.Type
		lastDot := strings.LastIndex(typ, ".")
		if lastDot == -1 {
			return fmt.Errorf("invalid type format: %s", typ)
		}

		importPath = typ[:lastDot]
		rawType := typ[lastDot+1:]

		parts := strings.Split(importPath, "/")
		importAlias = parts[len(parts)-1]

		typeName = importAlias + "." + rawType
	}

	success, fail := resolveTestValue(typeName, lower)

	p := Param{
		NameLower:        lower,
		NameFlat:         lower,
		NameCamel:        toCamel(name),
		Type:             typeName,
		ImportPath:       importPath,
		ImportAlias:      importAlias,
		TestSuccessValue: success,
		TestFailValue:    fail,
	}

	if err := os.MkdirAll(outDir, dirPerm); err != nil {
		return err
	}

	// ctx
	if err := writeFile(filepath.Join(outDir, p.NameFlat+"_ctx.gen.go"), ctxTpl, p); err != nil {
		return err
	}

	// test
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

	if existing, err := os.ReadFile(path); err == nil { //nolint:gosec // generator controlled path
		if bytes.Equal(existing, src) {
			return nil
		}
	}

	return os.WriteFile(path, src, filePerm)
}

func toCamel(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func resolveTestValue(t, nameLower string) (string, string) {
	switch t {
	case "string":
		return `"test-` + nameLower + `"`, `""`
	case "int":
		return "123", "0"
	case "bool":
		return "true", "false"
	default:
		return "*new(" + t + ")", "*new(" + t + ")"
	}
}
