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
	"go-boilerplate/pkg/xerrors"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

const (
	// ▼ カテゴリ単位のファイル連結ジョブの並列数チューニング定数（値の由来は README に記載）

	// maxSQLCConcurrency は、並列数の上限です。
	maxSQLCConcurrency = 4
	// minSQLCConcurrency は、並列数の下限です。
	minSQLCConcurrency = 2

	genFilePerm = 0o644
)

var (
	// errInvalidTargetType は、許可されていない DML カテゴリタイプが指定された場合のエラーです。
	errInvalidTargetType = xerrors.New("invalid target type")
	// errPathOutsideBaseDir は、生成対象パスがベースディレクトリの外を指した場合のエラーです。
	errPathOutsideBaseDir = xerrors.New("path is outside of baseDir")
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

// Generator は、DML ファイルから sqlc 用の統合 SQL ファイルを生成するオブジェクトです。
type Generator struct {
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

// NewGenerator は、merge-dml 用のジェネレーターインスタンスを生成します。
func NewGenerator(logger logging.Logger, workDir string) *Generator {
	return &Generator{
		logger:          logger,
		callerSkipCount: 1,
		workDir:         workDir,
		dmlRootDir:      "database/dml/",
		genRootDir:      "database/gen/",
		sqlcCfg:         "sqlc.yaml",
		fs:              osFileSystem{},
	}
}

// RunMerge は、DMLファイルをマージして、カテゴリごとに単一ファイルにまとめます。
func RunMerge(ctx context.Context, gen *Generator, targetType string) error {
	logger := gen.logger

	if err := validateTargetType(targetType); err != nil {
		return err
	}

	categories, err := gen.fs.ListSubDirNames(gen.dmlTypeRootAbs(targetType))
	if err != nil {
		logger.CallerSkip(gen.callerSkipCount).Named("mergedml.listDirs").Error(ctx, "failed to list directories",
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}
	if len(categories) == 0 {
		logger.CallerSkip(gen.callerSkipCount).Named("mergedml.NoCategories").Info(
			ctx,
			"no categories found under dml directory, cleanup only",
			logging.String("dml_path", gen.dmlRootDir+targetType),
		)

		// ★ 0件でも stale を消す（=このtypeの生成物を全消しに近い挙動）
		if err := gen.cleanupStaleGeneratedFiles(ctx, nil, targetType); err != nil {
			logger.CallerSkip(gen.callerSkipCount).Named("mergedml.cleanupStaleGeneratedFiles").Error(
				ctx,
				"failed to cleanup stale generated sql files",
				logging.Error(logging.ErrorKey, err),
			)
			return err
		}
		return nil
	}

	eg, egCtx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(resolveConcurrencyConst()))

	for _, category := range categories {
		cat := category
		eg.Go(func() error {
			if err := sem.Acquire(egCtx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			return gen.buildCategorySQLFile(egCtx, cat, targetType)
		})
	}

	if err := eg.Wait(); err != nil {
		gen.logger.CallerSkip(gen.callerSkipCount).Named("mergedml.buildMergedQueries").Error(ctx, "failed to build merged sql files",
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	if err := gen.cleanupStaleGeneratedFiles(ctx, categories, targetType); err != nil {
		gen.logger.CallerSkip(gen.callerSkipCount).Named("mergedml.cleanupStaleGeneratedFiles").Error(
			ctx,
			"failed to cleanup stale generated sql files",
			logging.Error(logging.ErrorKey, err),
		)
		return err
	}

	return nil
}

// validateTargetType は、許可済みの DML カテゴリタイプのみを受け付けます。
func validateTargetType(targetType string) error {
	switch targetType {
	case "repository", "query_service", "system_cqrs", "command_service":
		return nil
	default:
		return xerrors.Wrap(errInvalidTargetType, targetType)
	}
}

// dmlTypeRootAbs は、指定されたタイプのDMLルートディレクトリの絶対パスを返します。
func (g *Generator) dmlTypeRootAbs(targetType string) string {
	return filepath.Join(g.workDir, g.dmlRootDir, targetType)
}

// buildCategorySQLFile は、指定されたカテゴリのSQLファイルを連結して1つのSQLファイルにまとめます。
func (g *Generator) buildCategorySQLFile(ctx context.Context, category, targetType string) error {
	// 入力走査も workDir 起点で統一する。CWD 起点だと CWD != workDir のとき走査結果が 0 件になり、
	// 「入力なし」分岐に入って workDir 配下の生成物を誤って削除してしまうため。
	dmlDir := filepath.Join(g.workDir, g.dmlRootDir, targetType, category)

	files, err := g.fs.FindSQLFiles(dmlDir)
	if err != nil {
		return err
	}

	outName := fmt.Sprintf("%s_%s.gen.sql", category, targetType) // 例: <category>_<type>.gen.sql
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
			ctx,
			"no sql files found, remove generated file if exists",
			logging.String("category", category),
			logging.String("type", targetType),
			logging.String("dst", dstPath),
		)
		return nil
	}

	g.logger.CallerSkip(g.callerSkipCount).Named("mergedml.buildCategorySQLFile").Info(ctx, "build merged sql for sqlc",
		logging.String("category", category),
		logging.String("type", targetType),
		logging.String("dst", dstPath),
		logging.Int("files", len(files)),
	)

	if err := g.ensureUnderDir(dstPath); err != nil {
		return err
	}

	// 成功時のみ一括書き込みし、途中失敗でも部分生成物を残しません。
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
func (g *Generator) ensureUnderDir(path string) error {
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
		return xerrors.Wrap(errPathOutsideBaseDir, fmt.Sprintf("path=%s base=%s", absPath, absBase))
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

// resolveConcurrency は、CPU 数を [minSQLCConcurrency, maxSQLCConcurrency] にクランプして同時実行数を返します。
func resolveConcurrency(numCPU int) int {
	return max(min(numCPU, maxSQLCConcurrency), minSQLCConcurrency)
}

func (g *Generator) cleanupStaleGeneratedFiles(ctx context.Context, categories []string, targetType string) error {
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
		if !strings.HasSuffix(name, suffix) {
			continue
		}

		if _, ok := keep[name]; ok {
			continue
		}

		full := filepath.Join(genAbs, name)

		if err := g.ensureUnderDir(full); err != nil {
			return err
		}

		if err := g.fs.Remove(full); err != nil {
			return err
		}

		g.logger.CallerSkip(g.callerSkipCount).Named("mergedml.cleanupStaleGeneratedFiles").Info(
			ctx,
			"remove stale generated sql",
			logging.String("file", filepath.ToSlash(full)),
			logging.String("type", targetType),
		)
	}

	return nil
}
