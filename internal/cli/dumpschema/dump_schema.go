// Package dumpschema は、DBスキーマのダンプと整形のコアロジックを提供します。
package dumpschema

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/exec"
	"go-boilerplate/pkg/fs"
)

const schemaFilePerm = 0o644 // rw-r--r--

var (
	// dumpCommand は、スキーマダンプに使用するコマンド名を表します。
	dumpCommand = "pg_dump"
	// dumpSubArgs は、pg_dump に渡す固定オプション引数列を表します。
	dumpSubArgs = []string{
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--format=plain",
	}

	// trimPrefixes は、スキーマファイルから除去する行の接頭辞を表します。
	trimPrefixes = []string{
		`\`,
		"-- Dumped from database version",
		"-- Dumped by pg_dump version",
	}
)

// Generator は、スキーマダンプと整形に必要な依存と設定を保持します。
type Generator struct {
	logger          logging.Logger
	callerSkipCount int
	permission      os.FileMode

	workDir       string
	schemaRelPath string

	dumpCommand string
	dumpArgs    []string

	fs     fs.FS
	runner exec.Runner
}

// NewGenerator は、dump-schema 用のジェネレーターインスタンスを生成します。
func NewGenerator(logger logging.Logger, workDir string) *Generator {
	return &Generator{
		logger:          logger,
		callerSkipCount: 1,
		permission:      schemaFilePerm,
		workDir:         workDir,
		schemaRelPath:   "database/gen/schema.gen.sql",
		dumpCommand:     dumpCommand,
		dumpArgs:        dumpSubArgs,
		fs:              fs.OS{},
		runner:          exec.OS{},
	}
}

// RunDump は、DSN 解決・スキーマダンプ・整形のオーケストレーションを行います。
//
// 手順:
//  1. ダンプコマンドを実行してスキーマをダンプ
//  2. スキーマファイル内のメタコマンド行を除去
func RunDump(ctx context.Context, gen *Generator, loadDSN func() (string, error)) error {
	dbURL, err := loadDSN()
	if err != nil {
		return err
	}

	// まず生の schema を出力し、その後 sqlc が扱いやすい形に整形します。
	if err := gen.dumpSchema(ctx, dbURL); err != nil {
		return err
	}
	return gen.sanitizeSchemaInPlace()
}

// dumpSchema は、ダンプコマンドを実行してスキーマのDDLを取得し、schema.gen.sqlとして保存します。
func (g *Generator) dumpSchema(ctx context.Context, dbURL string) error {
	args := append([]string{dbURL}, g.dumpArgs...)

	g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.dumpSchema").Info("start pg_dump schema",
		logging.String("out", g.schemaRelPath),
	)

	out, err := g.runner.Output(ctx, g.workDir, g.dumpCommand, args)
	if err != nil {
		g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.dumpSchema").Warn("pg_dump failed",
			logging.String("out", g.schemaRelPath),
			logging.Error("pg_dump", err),
		)
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	schemaAbs := filepath.Join(g.workDir, g.schemaRelPath)
	if err := g.fs.WriteFile(schemaAbs, out, g.permission); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.dumpSchema").Info("pg_dump schema completed",
		logging.String("out", g.schemaRelPath),
	)

	return nil
}

// sanitizeSchemaInPlace は、schema.sql 内の psqlメタコマンド行を除去します。
func (g *Generator) sanitizeSchemaInPlace() error {
	srcAbs := filepath.Join(g.workDir, g.schemaRelPath)

	b, err := g.fs.ReadFile(srcAbs)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		// pg_dump 由来のメタ情報や psql メタコマンドは sqlc 不要のため除去します。
		trim := strings.TrimSpace(ln)
		for _, prefix := range trimPrefixes {
			if strings.HasPrefix(trim, prefix) {
				trim = ""
				break
			}
		}
		if trim == "" {
			continue
		}
		out = append(out, ln)
	}

	if err := g.fs.WriteFile(srcAbs, []byte(strings.Join(out, "\n")), g.permission); err != nil {
		return fmt.Errorf("write sanitized schema: %w", err)
	}

	g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.sanitizeSchemaInPlace").Info("schema sanitized for sqlc",
		logging.String("schema", g.schemaRelPath),
	)
	return nil
}
