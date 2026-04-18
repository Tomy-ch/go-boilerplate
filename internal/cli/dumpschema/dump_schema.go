// Package dumpschema は、DBスキーマをダンプして整形する機能を提供します。
package dumpschema

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"

	"github.com/spf13/cobra"
)

const schemaFilePerm = 0o644 // rw-r--r--

var (
	// dumpCommand は、スキーマダンプに使用するコマンド名を表します。
	dumpCommand = "pg_dump"
	// dumpDatabaseCommand は、スキーマダンプのためのコマンドフォーマット文字列を表します。
	dumpSubArgs = []string{
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--format=plain",
	}
	// workDir は、作業ディレクトリのパスを表します。
	workDir string

	// trimPrefixes は、スキーマファイルから除去する行の接頭辞を表します。
	trimPrefixes = []string{
		`\`,
		"-- Dumped from database version",
		"-- Dumped by pg_dump version",
	}
)

type generator struct {
	logger          logging.Logger
	callerSkipCount int
	permission      os.FileMode

	workDir       string
	schemaRelPath string

	dumpCommand string
	dumpArgs    []string
}

// newGenerator は、gensqlc用のジェネレーターインスタンスを生成します。
func newGenerator(logger logging.Logger) *generator {
	return &generator{
		logger:          logger,
		callerSkipCount: 1,
		permission:      schemaFilePerm,
		workDir:         workDir,
		schemaRelPath:   "database/gen/schema.gen.sql",
		dumpCommand:     dumpCommand,
		dumpArgs:        dumpSubArgs,
	}
}

// NewCommand は、sqlc generate コマンドを生成します。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump-schema",
		Short: "databaseに接続してスキーマをダンプして読み込みやすい形に整形します。",
		Long: "ファイルで定義されたdumpコマンドを実行してDBスキーマをダンプし、\n" +
			"メタコマンドの行を除去してsqlcで読み込みやすい形に整形します。\n" +
			"dumpコマンドを変更したい場合は、dumpCommandおよびdumpArgs変数を修正してください。",
		RunE: generateDumpSchema,
	}

	cmd.Flags().StringVar(&workDir, "work-dir", "/app", "working directory path")

	return cmd
}

// generateDumpSchema は、DBスキーマをダンプし整形します。
//
// 手順:
//  1. ダンプコマンド を実行してスキーマをダンプ
//  2. スキーマファイル内のメタコマンド行を除去
func generateDumpSchema(_ *cobra.Command, _ []string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	gen := newGenerator(logger)

	// アプリ設定から接続先 DSN を組み立て、ダンプ対象 DB を決定します。
	cfg, err := config.SetUpConfig()
	if err != nil {
		logger.CallerSkip(gen.callerSkipCount).Named("gensqlc.SetUpConfig").Error("failed to load config",
			logging.Error("config", err),
		)
		return err
	}

	dbCfg := config.NewDatabaseConfig(cfg)
	dbURL := driver.DSNString(dbCfg)

	ctx := context.Background()
	// まず生の schema を出力し、その後 sqlc が扱いやすい形に整形します。
	if err = gen.dumpSchema(ctx, dbURL); err != nil {
		return err
	}
	if err := gen.sanitizeSchemaInPlace(); err != nil {
		return err
	}

	return nil
}

// dumpSchema は、ダンプコマンドを実行してスキーマのDDLを取得し、schema.gen.sqlとして保存します。
func (g *generator) dumpSchema(ctx context.Context, dbURL string) error {
	schemaAbs := filepath.Join(g.workDir, g.schemaRelPath)

	// pg_dump の出力先を先に開いて、標準出力をそのまま schema.gen.sql に流します。
	f, err := os.Create(schemaAbs) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			g.logger.CallerSkip(g.callerSkipCount).Named("gensqlc.dumpSchema").Warn("failed to close schema file",
				logging.String("schema", g.schemaRelPath),
				logging.Error("close", closeErr),
			)
		}
	}()

	args := append([]string{dbURL}, g.dumpArgs...)

	// dump オプションは generator が保持しており、実行時は DB URL だけを先頭に差し込みます。
	cmd := exec.CommandContext(ctx, g.dumpCommand, args...) // #nosec G204
	cmd.Dir = g.workDir
	cmd.Stdout = f
	cmd.Stderr = os.Stderr

	g.logger.CallerSkip(g.callerSkipCount).Named("gensqlc.dumpSchema").Info("start pg_dump schema",
		logging.String("out", g.schemaRelPath),
	)

	if err := cmd.Run(); err != nil {
		g.logger.CallerSkip(g.callerSkipCount).Named("gensqlc.dumpSchema").Warn("pg_dump failed (schema file may be partial)",
			logging.String("out", g.schemaRelPath),
			logging.Error("pg_dump", err),
		)
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	g.logger.CallerSkip(g.callerSkipCount).Named("gensqlc.dumpSchema").Info("pg_dump schema completed",
		logging.String("out", g.schemaRelPath),
	)

	return nil
}

// sanitizeSchemaInPlace は、schema.sql 内の psqlメタコマンド行を除去します。
func (g *generator) sanitizeSchemaInPlace() error {
	srcAbs := filepath.Join(g.workDir, g.schemaRelPath)

	b, err := os.ReadFile(srcAbs) // #nosec G304
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

	//nolint:gosec // safe: path comes from trusted CLI input
	if err := os.WriteFile(srcAbs, []byte(strings.Join(out, "\n")), g.permission); err != nil {
		return fmt.Errorf("write sanitized schema: %w", err)
	}

	g.logger.CallerSkip(g.callerSkipCount).Named("gensqlc.sanitizeSchemaInPlace").Info("schema sanitized for sqlc",
		logging.String("schema", g.schemaRelPath),
	)
	return nil
}
