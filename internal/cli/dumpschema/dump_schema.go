// Package dumpschema は、DBスキーマのダンプと整形のコアロジックを提供します。
package dumpschema

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/exec"
	"go-boilerplate/pkg/fs"
	"go-boilerplate/pkg/xerrors"
)

const schemaFilePerm = 0o644 // rw-r--r--

// dumpSchemaLoggerName は dumpSchema のログ発生元名です（<package>.<method> 規約）。
const dumpSchemaLoggerName = "dumpschema.dumpSchema"

var (
	dumpCommand = "pg_dump"
	dumpSubArgs = []string{
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--format=plain",
	}

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

// RunDump は、DSN 解決・スキーマダンプ・整形を行い、schema.gen.sql を sqlc が読み込める形で書き出します。
// loadDSN は (パスワード非含有 DSN, パスワード) を返します。
func (g *Generator) RunDump(ctx context.Context, loadDSN func() (string, string, error)) error {
	dbURL, password, err := loadDSN()
	if err != nil {
		return err
	}
	return g.dumpSchema(ctx, dbURL, password)
}

// dumpSchema は、ダンプコマンドを実行してスキーマのDDLを取得し、sqlc 向けに整形して schema.gen.sql として保存します。
func (g *Generator) dumpSchema(ctx context.Context, dbURL, password string) error {
	args := append([]string{dbURL}, g.dumpArgs...)
	env := []string{"PGPASSWORD=" + password}

	g.logger.CallerSkip(g.callerSkipCount).Named(dumpSchemaLoggerName).Info(ctx, "start pg_dump schema",
		logging.String("out", g.schemaRelPath),
	)

	out, err := g.runner.Output(ctx, g.workDir, env, g.dumpCommand, args)
	if err != nil {
		g.logger.CallerSkip(g.callerSkipCount).Named(dumpSchemaLoggerName).Warn(ctx, "pg_dump failed",
			logging.String("out", g.schemaRelPath),
			logging.Error(logging.ErrorKey, err),
		)
		return xerrors.Wrap(err, "pg_dump failed")
	}

	schemaAbs := filepath.Join(g.workDir, g.schemaRelPath)
	if err := g.fs.WriteFile(schemaAbs, sanitizeSchema(out), g.permission); err != nil {
		return xerrors.Wrap(err, "failed to write schema file")
	}

	g.logger.CallerSkip(g.callerSkipCount).Named(dumpSchemaLoggerName).Info(ctx, "pg_dump schema completed",
		logging.String("out", g.schemaRelPath),
	)

	return nil
}

// sanitizeSchema は、pg_dump 出力を sqlc 向けに整形して返します。
// trimPrefixes 一致のメタ行に加え、空行（空白のみ・元から空）も除去します（空行除去まで含むのは意図的）。
func sanitizeSchema(raw []byte) []byte {
	lines := strings.Split(string(raw), "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
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

	return []byte(strings.Join(out, "\n"))
}
