// Package dumpschema は、DBスキーマをダンプして整形する機能を提供します。
package dumpschema

import (
	"context"
	"fmt"
	"io"
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

// fileSystem は dump-schema が必要とするファイル操作を抽象化します。
// テストでフェイクを注入し、実ファイルシステムに触れずに分岐を検証できるようにします。
type fileSystem interface {
	Create(name string) (io.WriteCloser, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// commandRunner は外部コマンド（pg_dump）の実行を抽象化します。
type commandRunner interface {
	Run(ctx context.Context, dir, name string, args []string, stdout io.Writer) error
}

// osFileSystem は os パッケージを用いた fileSystem の実装です。
type osFileSystem struct{}

// execCommandRunner は os/exec を用いた commandRunner の実装です。
type execCommandRunner struct{}

type generator struct {
	logger          logging.Logger
	callerSkipCount int
	permission      os.FileMode

	workDir       string
	schemaRelPath string

	dumpCommand string
	dumpArgs    []string

	fs     fileSystem
	runner commandRunner
}

func (osFileSystem) Create(name string) (io.WriteCloser, error) {
	return os.Create(name) //nolint:gosec // path は信頼された CLI フラグ由来の固定パス
}

func (osFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // path は信頼された CLI フラグ由来の固定パス
}

func (osFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (execCommandRunner) Run(ctx context.Context, dir, name string, args []string, stdout io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // name/args は信頼された CLI 設定由来
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// newGenerator は、dump-schema 用のジェネレーターインスタンスを生成します。
func newGenerator(logger logging.Logger, workDir string) *generator {
	return &generator{
		logger:          logger,
		callerSkipCount: 1,
		permission:      schemaFilePerm,
		workDir:         workDir,
		schemaRelPath:   "database/gen/schema.gen.sql",
		dumpCommand:     dumpCommand,
		dumpArgs:        dumpSubArgs,
		fs:              osFileSystem{},
		runner:          execCommandRunner{},
	}
}

// NewCommand は、dump-schema コマンドを生成します。
func NewCommand() *cobra.Command {
	// フラグはパッケージグローバルにせずローカルに束縛し、コマンドの並列テスト安全性を保ちます。
	var workDir string

	cmd := &cobra.Command{
		Use:   "dump-schema",
		Short: "databaseに接続してスキーマをダンプして読み込みやすい形に整形します。",
		Long: "ファイルで定義されたdumpコマンドを実行してDBスキーマをダンプし、\n" +
			"メタコマンドの行を除去してsqlcで読み込みやすい形に整形します。\n" +
			"dumpコマンドを変更したい場合は、dumpCommandおよびdumpSubArgs変数を修正してください。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDumpSchema(cmd.Context(), workDir)
		},
	}

	cmd.Flags().StringVar(&workDir, "work-dir", "/app", "working directory path")

	return cmd
}

// runDumpSchema は、設定を読み込み、DBスキーマのダンプと整形を実行する薄い殻です。
//
// 手順:
//  1. ダンプコマンドを実行してスキーマをダンプ
//  2. スキーマファイル内のメタコマンド行を除去
func runDumpSchema(ctx context.Context, workDir string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}

	gen := newGenerator(logger, workDir)

	// アプリ設定から接続先 DSN を組み立て、ダンプ対象 DB を決定します。
	cfg, err := config.SetUpConfig()
	if err != nil {
		logger.CallerSkip(gen.callerSkipCount).Named("dumpschema.SetUpConfig").Error("failed to load config",
			logging.Error("config", err),
		)
		return err
	}

	dbCfg := config.NewDatabaseConfig(cfg)
	dbURL := driver.DSNString(dbCfg)

	// まず生の schema を出力し、その後 sqlc が扱いやすい形に整形します。
	if err = gen.dumpSchema(ctx, dbURL); err != nil {
		return err
	}
	return gen.sanitizeSchemaInPlace()
}

// dumpSchema は、ダンプコマンドを実行してスキーマのDDLを取得し、schema.gen.sqlとして保存します。
func (g *generator) dumpSchema(ctx context.Context, dbURL string) error {
	schemaAbs := filepath.Join(g.workDir, g.schemaRelPath)

	// pg_dump の出力先を先に開いて、標準出力をそのまま schema.gen.sql に流します。
	f, err := g.fs.Create(schemaAbs)
	if err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.dumpSchema").Warn("failed to close schema file",
				logging.String("schema", g.schemaRelPath),
				logging.Error("close", closeErr),
			)
		}
	}()

	args := append([]string{dbURL}, g.dumpArgs...)

	g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.dumpSchema").Info("start pg_dump schema",
		logging.String("out", g.schemaRelPath),
	)

	if err := g.runner.Run(ctx, g.workDir, g.dumpCommand, args, f); err != nil {
		g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.dumpSchema").Warn("pg_dump failed (schema file may be partial)",
			logging.String("out", g.schemaRelPath),
			logging.Error("pg_dump", err),
		)
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	g.logger.CallerSkip(g.callerSkipCount).Named("dumpschema.dumpSchema").Info("pg_dump schema completed",
		logging.String("out", g.schemaRelPath),
	)

	return nil
}

// sanitizeSchemaInPlace は、schema.sql 内の psqlメタコマンド行を除去します。
func (g *generator) sanitizeSchemaInPlace() error {
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
