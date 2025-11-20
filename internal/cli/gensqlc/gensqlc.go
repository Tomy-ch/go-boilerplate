// Package gensqlc は、SQLCのコード生成を行うコマンドを提供します。
package gensqlc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	// ▼ チューニング用定数群(値の由来や背景は README に記載)

	// sqlcDBConcurrency:
	//   sqlc生成ジョブ(DBイントロスペクション＋コード出力)の並列数。
	sqlcDBConcurrency = 4

	// maxSQLCConcurrency:
	//   占有してよい並列の上限。
	//   runtime.NumCPU()と組み合わせてDockerのCPUを極端に占有しないようにする。
	maxSQLCConcurrency = 4

	// workDir:
	//   DockerツールコンテナのWORKDIR(=リポジトリルート)を前提にした絶対パス。
	//   exec.Cmd.Dirにも同値を設定して、相対パスの解決ブレを防止。
	workDir = "/app"

	// ▼ ここから下は基本固定

	// permRWRR:
	//   0644(ユーザ: 読み書き / グループ: 読み / その他: 読み)
	permRWRR = 0o644

	// dmlRootDir:
	//   DMLのルート(この配下にdatabase/dml/[repository|query_service]/<category>/...のSQLを置く)。
	dmlRootDir = "database/dml/"

	// sqlcRootDir:
	//   sqlcのルートディレクトリ。
	sqlcRootDir = "database/sqlc/"

	// settingYamlFile:
	//   sqlcの設定YAMLのファイル名。__DATABASE_URL__を置換して使う。
	settingYamlFile = "sqlc.yaml"

	// minSQLCConcurrency:
	//   並列の下限。I/O待ちが多いので1だと非効率なので最低2を確保。
	minSQLCConcurrency = 2
)

// --type(repository|query_service)
var targetType string

// NewCommand は、sqlc generate コマンドを生成します。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-sqlc",
		Short: "DMLディレクトリ(database/dml/<repository/query_service>)のsqlファイルを対象に sqlc generate を実行",
		Long: "database/dml/<type>/ 配下のsqlファイルを対象に、\n" +
			"sqlファイルをdatabase/sqlc/ 配下に一時的にコピーし、\n" +
			"sqlc generate を実行してコード生成を行います。",
		RunE: generateSQLCRun,
	}

	cmd.Flags().StringVar(&targetType, "type", "", "filter TYPE (repository|query_service)")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

// generateSQLCRun は、sqlc generate を実行するコマンドの実行処理を行います。
//
// 手順:
//  1. 設定ロード(DATABASE_URLを取得)
//  2. 設定YAMLの読み込み
//  3. 対象カテゴリの列挙(<type>配下の全サブディレクトリ)
//  4. 各カテゴリのSQLファイルをsqlcRootDirへ並列コピー
//  5. sqlc generate実行
//  6. 一時コピーしたSQLファイルの削除
//
// 並列数はresolveConcurrencyConst()で決定します。
func generateSQLCRun(_ *cobra.Command, _ []string) error {
	logger := logging.NewProductionLogger()

	// 1) 設定ロード（DATABASE_URL を取得）
	cfg, err := config.SetUpConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.NamedError("config", err))
	}
	dbURL := cfg.DatabaseDSN()

	// 2) 設定YAMLの読み込み
	sqlcYamlRaw, err := os.ReadFile(filepath.Join(workDir, sqlcRootDir, settingYamlFile))
	if err != nil {
		logger.Fatal("failed to read settings yaml", zap.NamedError("os.ReadFile", err))
	}

	// 3) カテゴリ列挙
	var categories []string
	if categories, err = listDirs(filepath.Join(workDir, dmlRootDir, targetType)); err != nil {
		logger.Fatal("failed to list directories", zap.String("path", dmlRootDir+targetType), zap.NamedError("os.ReadDir", err))
	}
	if len(categories) == 0 {
		logger.Info("dml directory is empty, skipping sqlc generation", zap.String("path", dmlRootDir+targetType))
		return nil
	}

	// 4) 並列実行。errgroup+セマフォで同時実行数を制限。
	rootCtx := context.Background()
	g, egCtx := errgroup.WithContext(rootCtx)
	s := semaphore.NewWeighted(int64(resolveConcurrencyConst()))
	for _, category := range categories {
		cat := category
		g.Go(func() error {
			if err := s.Acquire(egCtx, 1); err != nil {
				return err
			}
			defer s.Release(1)

			return copySQLFile(logger, dmlRootDir, cat, targetType, sqlcRootDir)
		})
	}
	if err := g.Wait(); err != nil {
		logger.Fatal("failed to copy SQL files", zap.NamedError("copySQLFile", err))
		return err
	}

	// 5) sqlc generate 実行
	if err := runSQLCForCategory(rootCtx, logger, string(sqlcYamlRaw), dbURL, targetType); err != nil {
		logger.Fatal("failed to run sqlc for generate", zap.NamedError("runSQLCForCategory", err))
		return err
	}

	// 6) 一時コピーしたsqlファイルを削除
	return cleanupSQLFiles(logger, filepath.Join(workDir, sqlcRootDir))
}

// listDirs は、指定されたディレクトリ内のサブディレクトリをリストします。
func listDirs(base string) ([]string, error) {
	ents, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	subDirs := make([]string, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			subDirs = append(subDirs, e.Name())
		}
	}
	sort.Strings(subDirs)
	return subDirs, nil
}

// resolveConcurrencyConst は、SQLCの同時実行数を解決します。
//
//   - 上限: min(runtime.NumCPU(), maxSQLCConcurrency)
//   - 下限: minSQLCConcurrency
//
// Docker/CIで全コア占有を避けるため、固定上限を設けています。
func resolveConcurrencyConst() int {
	maxAllowed := runtime.NumCPU()
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

// runSQLCForCategory は、指定されたカテゴリの一時YAMLを作成→sqlc generateを実行します。
func runSQLCForCategory(
	ctx context.Context,
	logger *zap.Logger,
	tpl, dbURL, targetType string,
) error {
	// 1) テンプレ内のプレースホルダを置換
	repl := strings.NewReplacer(
		"__DATABASE_URL__", dbURL,
	).Replace(tpl)

	// 2) YAMLを書き出し
	tmpPath := filepath.Join(workDir, settingYamlFile)
	if err := os.WriteFile(tmpPath, []byte(repl), permRWRR); err != nil {
		return fmt.Errorf("failed to write temporary YAML file: os.WriteFile: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			logger.Warn("failed to remove YAML file", zap.NamedError("os.Remove", err))
		}
	}()

	// 3) YAMLを使ってsqlc実行
	// #nosec G204 -- settingYamlFile is a constant and does not originate from user
	cmd := exec.CommandContext(ctx, "sqlc", "generate", "-f", settingYamlFile)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sqlc generate failed: type: %s, exec.Run: %w", targetType, err)
	}

	logger.Info("sqlc generate completed", zap.String("type", targetType))

	return nil
}

// copySQLFile は、指定されたカテゴリのDMLのSQLファイルを指定したディレクトリにコピーします。
func copySQLFile(
	logger *zap.Logger,
	workDir string,
	category string,
	targetType string,
	sqlcDir string,
) error {
	dmlDir := filepath.Join(workDir, targetType, category)
	return filepath.WalkDir(dmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".sql" {
			return nil
		}

		// 元ファイル名 (拡張子付き) 例: find_by_id.sql
		base := filepath.Base(path)

		// .sql を一旦落とす
		nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))

		// <Category>_<Type>_元のファイル名.sql
		// 例: user_repository_find_by_id.sql
		newName := fmt.Sprintf("%s_%s_%s.sql", category, targetType, nameWithoutExt)
		dstPath := filepath.Join(sqlcDir, newName)

		logger.Debug("copy dml sql to sqlc",
			zap.String("src", path),
			zap.String("dst", dstPath),
		)

		if err := copyFile(logger, path, dstPath); err != nil {
			return fmt.Errorf("copy %s -> %s: %w", path, dstPath, err)
		}

		return nil
	})
}

// copyFile は、srcファイルをdstファイルにコピーします。
func copyFile(logger *zap.Logger, src, dst string) error {
	if err := ensureUnderDir(logger, src, dmlRootDir); err != nil {
		logger.Fatal("invalid src path", zap.String("src", src), zap.NamedError("ensureUnderDir", err))
	}
	if err := ensureUnderDir(logger, dst, sqlcRootDir); err != nil {
		logger.Fatal("invalid dst path", zap.String("dst", dst), zap.NamedError("ensureUnderDir", err))
	}

	// #nosec G304 -- src is verified under a fixed root directory and does not originate from user input
	in, err := os.Open(src)
	if err != nil {
		logger.Fatal("failed to open src file", zap.String("src", src), zap.NamedError("os.Open", err))
	}
	defer func() {
		if cerr := in.Close(); cerr != nil {
			logger.Fatal("failed to close src file", zap.String("src", src), zap.NamedError("in.Close", cerr))
		}
	}()

	// #nosec G304 -- dst is verified under a fixed root directory and does not originate from user input
	out, err := os.Create(dst)
	if err != nil {
		logger.Fatal("failed to create dst file", zap.String("dst", dst), zap.NamedError("os.Create", err))
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			logger.Fatal("failed to close dst file", zap.String("dst", dst), zap.NamedError("out.Close", cerr))
		}
	}()

	if err := out.Sync(); err != nil {
		logger.Fatal("failed to sync dst file", zap.String("dst", dst), zap.NamedError("out.Sync", err))
	}
	return nil
}

// cleanupSQLFiles は、指定したディレクトリ内の.sqlファイルを削除します。
func cleanupSQLFiles(logger *zap.Logger, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		logger.Fatal("failed to read directory", zap.String("dir", dir), zap.NamedError("os.ReadDir", err))
	}

	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) != ".sql" || e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			logger.Fatal("failed to remove file", zap.String("file", filepath.Join(dir, name)), zap.NamedError("os.Remove", err))
		}
	}

	return nil
}

// ensureUnderDir は path が baseDir 配下かを検証します。
func ensureUnderDir(logger *zap.Logger, path, baseDir string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("failed to get absolute path", zap.String("path", path), zap.NamedError("filepath.Abs", err))
		return err
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		logger.Error("failed to get absolute baseDir", zap.String("baseDir", baseDir), zap.NamedError("filepath.Abs", err))
		return err
	}

	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		logger.Error("failed to get relative path", zap.String("baseDir", absBase), zap.String("path", absPath), zap.NamedError("filepath.Rel", err))
		return err
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		logger.Error("path is outside of baseDir", zap.String("path", absPath), zap.String("baseDir", absBase))
		return err
	}
	return nil
}
