// Package mergedml は、DMLをマージして単一ファイルにまとめる機能を提供します。
package mergedml

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"go-boilerplate/internal/logging"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

const (
	// ▼ チューニング用定数群(値の由来や背景は README に記載)

	// sqlcDBConcurrency:
	//   sqlc生成ジョブ(DBイントロスペクション＋コード出力)の並列数。
	sqlcDBConcurrency = 4

	// maxSQLCConcurrency:
	//   占有してよい並列の上限。
	//   runtime.NumCPU()と組み合わせてDockerのCPUを極端に占有しないようにする。
	maxSQLCConcurrency = 4

	// ▼ ここから下は基本固定

	// minSQLCConcurrency:
	//   並列の下限。I/O待ちが多いので1だと非効率なので最低2を確保。
	minSQLCConcurrency = 2

	// genFilePerm は、生成する SQL ファイルのパーミッションです。
	genFilePerm = 0o644
)

// FileSystem は merge-dml が必要とするファイル操作を抽象化します。
type FileSystem interface {
	ListSubDirNames(base string) ([]string, error)    // base 直下のサブディレクトリ名（昇順）
	ListGenFileNames(genDir string) ([]string, error) // genDir 直下のファイル名（非ディレクトリ）
	FindSQLFiles(dir string) ([]string, error)        // dir 配下の .sql ファイルパス（昇順）
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
}

// osFileSystem は os パッケージを用いた FileSystem の実装です。
type osFileSystem struct{}

type generator struct {
	logger          logging.Logger
	callerSkipCount int

	workDir    string
	dmlRootDir string
	genRootDir string
	sqlcCfg    string

	fs FileSystem
}

func (osFileSystem) ListSubDirNames(base string) ([]string, error) {
	ents, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

func (osFileSystem) ListGenFileNames(genDir string) ([]string, error) {
	ents, err := os.ReadDir(genDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (osFileSystem) FindSQLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".sql" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (osFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // src は固定ルート配下で検証済み・ユーザー入力由来ではない
}

func (osFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (osFileSystem) Remove(name string) error {
	return os.Remove(name)
}

// newGenerator は、merge-dml 用のジェネレーターインスタンスを生成します。
func newGenerator(logger logging.Logger, workDir string) *generator {
	return &generator{
		logger:          logger,
		callerSkipCount: 1,
		workDir:         workDir,
		dmlRootDir:      "database/dml/",
		genRootDir:      "database/gen/",
		sqlcCfg:         "sqlc.yaml",
		fs:              osFileSystem{},
	}
}

// NewCommand は、merge-dml コマンドを生成します。
func NewCommand() *cobra.Command {
	var (
		targetType string
		workDir    string
	)

	cmd := &cobra.Command{
		Use:   "merge-dml",
		Short: "DMLディレクトリ(database/dml/<repository/query_service/command_service>)のsqlファイルを対象にして、<type>ごとにマージします。",
		Long: "指定されたタイプ(repository|query_service|command_service)のDMLディレクトリ内の全サブディレクトリを走査し、\n" +
			"各カテゴリごとにSQLファイルを連結して1つのSQLファイルにまとめます。\n" +
			"生成されるファイルは database/gen/ 配下に <category>_<type>.gen.sql という名前で保存されます。",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mergeDMLRun(cmd.Context(), targetType, workDir)
		},
	}

	cmd.Flags().StringVar(&targetType, "type", "", "filter TYPE (repository|query_service|command_service)")
	_ = cmd.MarkFlagRequired("type")
	cmd.Flags().StringVar(&workDir, "work-dir", "/app", "working directory path")

	return cmd
}

// mergeDMLRun は、ロガーとジェネレーターを実依存で組み立て、マージ処理を runMerge へ委譲する薄い殻です。
func mergeDMLRun(ctx context.Context, targetType, workDir string) error {
	logger, err := logging.NewProductionLogger()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	gen := newGenerator(logger, workDir)
	return runMerge(ctx, gen, targetType)
}

// runMerge は、DMLファイルをマージして、カテゴリごとに単一ファイルにまとめます。
func runMerge(ctx context.Context, gen *generator, targetType string) error {
	logger := gen.logger

	// type 配下のカテゴリ一覧を取得し、カテゴリ単位でマージ対象を決定します。
	categories, err := gen.fs.ListSubDirNames(gen.dmlTypeRootAbs(targetType))
	if err != nil {
		logger.CallerSkip(gen.callerSkipCount).Named("mergedml.listDirs").Error("failed to list directories",
			logging.Error("os.ReadDir", err),
		)
		return err
	}
	if len(categories) == 0 {
		logger.CallerSkip(gen.callerSkipCount).Named("mergedml.NoCategories").Info(
			"no categories found under dml directory, cleanup only",
			logging.String("dml_path", gen.dmlRootDir+targetType),
		)

		// ★ 0件でも stale を消す（=このtypeの生成物を全消しに近い挙動）
		if err := gen.cleanupStaleGeneratedFiles(nil, targetType); err != nil {
			logger.CallerSkip(gen.callerSkipCount).Named("mergedml.cleanupStaleGeneratedFiles").Error(
				"failed to cleanup stale generated sql files",
				logging.Error("cleanup", err),
			)
			return err
		}
		return nil
	}

	// カテゴリごとの生成は独立しているため並列化しつつ、同時実行数は semaphore で抑制します。
	eg, egCtx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(resolveConcurrencyConst()))

	for _, category := range categories {
		cat := category
		eg.Go(func() error {
			if err := sem.Acquire(egCtx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			return gen.buildCategorySQLFile(cat, targetType)
		})
	}

	if err := eg.Wait(); err != nil {
		gen.logger.CallerSkip(gen.callerSkipCount).Named("mergedml.buildMergedQueries").Error("failed to build merged sql files",
			logging.Error("errgroup.Wait", err),
		)
		return err
	}

	// 今回生成されなかった旧ファイルを削除し、database/gen 配下を最新の入力状態に合わせます。
	if err := gen.cleanupStaleGeneratedFiles(categories, targetType); err != nil {
		gen.logger.CallerSkip(gen.callerSkipCount).Named("mergedml.cleanupStaleGeneratedFiles").Error(
			"failed to cleanup stale generated sql files",
			logging.Error("cleanup", err),
		)
		return err
	}

	return nil
}

// dmlTypeRootAbs は、指定されたタイプのDMLルートディレクトリの絶対パスを返します。
func (g *generator) dmlTypeRootAbs(targetType string) string {
	return filepath.Join(g.workDir, g.dmlRootDir, targetType)
}

// buildCategorySQLFile は、指定されたカテゴリのSQLファイルを連結して1つのSQLファイルにまとめます。
func (g *generator) buildCategorySQLFile(category, targetType string) error {
	// 入力走査も workDir 起点で統一する。CWD 起点だと CWD != workDir のとき走査結果が 0 件になり、
	// 「入力なし」分岐に入って workDir 配下の生成物を誤って削除してしまうため。
	dmlDir := filepath.Join(g.workDir, g.dmlRootDir, targetType, category)

	files, err := g.fs.FindSQLFiles(dmlDir)
	if err != nil {
		return err
	}

	outName := fmt.Sprintf("%s_%s.gen.sql", category, targetType) // 例: prefecture_repository.gen.sql
	// 出力先も workDir 起点で統一し、相対/絶対の二系統を排除する。
	dstPath := filepath.Join(g.workDir, g.genRootDir, outName)

	if len(files) == 0 {
		// 「カテゴリは存在するが SQL が空」の場合は、生成物を消すのが最新状態とみなします。
		if err := g.ensureUnderDir(dstPath); err != nil {
			return err
		}

		if err := g.fs.Remove(dstPath); err != nil && !os.IsNotExist(err) {
			return err
		}

		g.logger.CallerSkip(g.callerSkipCount).Named("mergedml.buildCategorySQLFile").Info(
			"no sql files found, remove generated file if exists",
			logging.String("category", category),
			logging.String("type", targetType),
			logging.String("dst", dstPath),
		)
		return nil
	}

	g.logger.CallerSkip(g.callerSkipCount).Named("mergedml.buildCategorySQLFile").Info("build merged sql for sqlc",
		logging.String("category", category),
		logging.String("type", targetType),
		logging.String("dst", dstPath),
		logging.Int("files", len(files)),
	)

	// 出力先が必ず database/gen 配下であることを確認してから書き出します。
	if err := g.ensureUnderDir(dstPath); err != nil {
		return err
	}

	// 連結結果はメモリ上に構築し、成功時のみ一括書き込みします（部分生成物を残さない）。
	var buf bytes.Buffer
	for _, fpath := range files {
		// 由来が分かるように見出しを入れる（sqlcはSQLコメントなら無害）
		_, _ = fmt.Fprintf(&buf, "\n-- === source: %s ===\n", filepath.ToSlash(fpath))

		b, err := g.fs.ReadFile(fpath)
		if err != nil {
			return err
		}
		buf.Write(b)

		// ファイル末尾に改行が無いケースでも連結が壊れないように
		if len(b) > 0 && b[len(b)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}

	return g.fs.WriteFile(dstPath, buf.Bytes(), genFilePerm)
}

// ensureUnderDir は path が baseDir 配下かを検証します。
func (g *generator) ensureUnderDir(path string) error {
	// 誤ったパス解決や path traversal により、想定外の場所を操作しないようにします。
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	absBase, err := filepath.Abs(filepath.Join(g.workDir, g.genRootDir))
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return err
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path is outside of baseDir: path=%s base=%s", absPath, absBase)
	}
	return nil
}

// resolveConcurrencyConst は、SQLCの同時実行数を解決します。
//
//   - 上限: min(runtime.NumCPU(), maxSQLCConcurrency)
//   - 下限: minSQLCConcurrency
//
// Docker/CIで全コア占有を避けるため、固定上限を設けています。
func resolveConcurrencyConst() int {
	return resolveConcurrency(runtime.NumCPU())
}

// resolveConcurrency は、与えられた CPU 数から SQLC の同時実行数を解決します（resolveConcurrencyConst の純粋本体）。
func resolveConcurrency(numCPU int) int {
	maxAllowed := numCPU
	if maxAllowed > maxSQLCConcurrency {
		maxAllowed = maxSQLCConcurrency
	}

	switch {
	case sqlcDBConcurrency > maxAllowed:
		return maxAllowed
	case sqlcDBConcurrency < minSQLCConcurrency:
		return minSQLCConcurrency
	default:
		return sqlcDBConcurrency
	}
}

func (g *generator) cleanupStaleGeneratedFiles(categories []string, targetType string) error {
	// 今回の入力から再生成されるファイル名だけを keep 対象として記録します。
	keep := make(map[string]struct{}, len(categories))
	for _, cat := range categories {
		keep[fmt.Sprintf("%s_%s.gen.sql", cat, targetType)] = struct{}{}
	}

	genAbs := filepath.Join(g.workDir, g.genRootDir) // /app/database/gen
	names, err := g.fs.ListGenFileNames(genAbs)
	if err != nil {
		return err
	}

	suffix := fmt.Sprintf("_%s.gen.sql", targetType)

	for _, name := range names {
		// 今回対象の type と無関係な生成物は触らず、そのまま残します。
		if !strings.HasSuffix(name, suffix) {
			continue
		}

		// 今回も生成されるファイルは最新扱いなので削除しません。
		if _, ok := keep[name]; ok {
			continue
		}

		full := filepath.Join(genAbs, name)

		// 念のため、genRootDir配下であることを確認
		if err := g.ensureUnderDir(full); err != nil {
			return err
		}

		if err := g.fs.Remove(full); err != nil {
			return err
		}

		g.logger.CallerSkip(g.callerSkipCount).Named("mergedml.cleanupStaleGeneratedFiles").Info(
			"remove stale generated sql",
			logging.String("file", filepath.ToSlash(full)),
			logging.String("type", targetType),
		)
	}

	return nil
}
