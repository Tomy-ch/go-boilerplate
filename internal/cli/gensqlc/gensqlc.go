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
	"boilerplate-go/internal/controller/middleware/logging"

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

	// dmlDir:
	//   DMLのルート(この配下にdatabase/dml/[repository|query_service]/<category>/...のSQLを置く)。
	dmlDir = "database/dml/"

	// sqlcDir:
	//   sqlcテンプレートや生成時の一時YAMLを置く場所。
	sqlcDir = "database/sqlc/"

	// templateYamlFile:
	//   テンプレートYAMLのファイル名。__DATABASE_URL__/__TYPE__/__CATEGORY__を置換して使う。
	templateYamlFile = "sqlc.template.yaml"

	// minSQLCConcurrency:
	//   並列の下限。I/O待ちが多いので1だと非効率なので最低2を確保。
	minSQLCConcurrency = 2
)

var (
	// --type(repository|query_service)
	targetType string
	// --category(例: user, product)。未指定なら配下の全カテゴリを対象。
	targetCategory string
)

// NewCommand は、sqlc generate コマンドを生成します。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-sqlc",
		Short: "TYPE/CATEGORY（repository/query_service）ごとにテンプレYAMLを置換して sqlc generate を実行",
		Long: "database/dml/<type>/<category>/ 配下のsqlファイルを対象に、\n" +
			"テンプレYAMLを用いて生成設定ファイルを作成し、sqlcのコードを生成します。",
		RunE: generateSQLCRun,
	}

	cmd.Flags().StringVar(&targetType, "type", "", "filter TYPE (repository|query_service)")
	cmd.Flags().StringVar(&targetCategory, "category", "", "filter CATEGORY (e.g. product)")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

// generateSQLCRun は、sqlc generate を実行するコマンドの実行処理を行います。
//
// 手順:
//  1. 設定ロード(DATABASE_URLを取得)
//  2. 対象カテゴリの列挙(--category 未指定時は<dmlDir>/<type>配下のサブディレクトリ)
//  3. テンプレYAMLを読み込み、プレースホルダ置換
//  4. 一時YAMLを作成して【sqlc generate -f】で実行(完了後削除)
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

	// 2) カテゴリ列挙
	var categories []string
	if targetCategory != "" {
		// --category が指定されている場合、そのカテゴリのみを対象とする
		categories = []string{targetCategory}
	} else {
		// --category が未指定の場合、全てのサブディレクトリを対象とする
		if categories, err = listDirs(filepath.Join(workDir, dmlDir, targetType)); err != nil {
			logger.Fatal("failed to list directories", zap.String("path", dmlDir+targetType), zap.NamedError("os.ReadDir", err))
		}
		if len(categories) == 0 {
			logger.Info("no categories found for type", zap.String("type", targetType))
			return nil
		}
	}

	// 3) テンプレートYAMLを読み込み
	tplRaw, err := os.ReadFile(filepath.Join(workDir, sqlcDir, templateYamlFile))
	if err != nil {
		logger.Fatal("template read error", zap.NamedError("os.ReadFile", err))
	}

	// 4) 並列実行。errgroup+セマフォで同時実行数を制限。
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	s := semaphore.NewWeighted(int64(resolveConcurrencyConst()))
	for _, category := range categories {
		category := category
		g.Go(func() error {
			if err = s.Acquire(ctx, 1); err != nil {
				return err
			}
			defer s.Release(1)

			// カテゴリ単位でsqlcを実行
			return runSQLCForCategory(ctx, logger, string(tplRaw), dbURL, targetType, category)
		})
	}
	if err := g.Wait(); err != nil {
		logger.Error("failed to run sqlc for generate", zap.NamedError("runSQLCForCategory", err))
		return err
	}
	return nil
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
	tpl, dbURL, targetType, category string,
) error {
	// 1) テンプレ内のプレースホルダを置換（DB URL / TYPE / CATEGORY）
	repl := strings.NewReplacer(
		"__DATABASE_URL__", dbURL,
		"__TYPE__", targetType,
		"__CATEGORY__", category,
	).Replace(tpl)

	// 2) 一時YAMLを書き出し
	tmpYAML := fmt.Sprintf("sqlc.%s.%s.yaml", targetType, category)
	tmpPath := filepath.Join(workDir, tmpYAML)
	if err := os.WriteFile(tmpPath, []byte(repl), permRWRR); err != nil {
		return fmt.Errorf("failed to write temporary YAML file: os.WriteFile: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			logger.Warn("failed to remove temporary YAML file", zap.NamedError("os.Remove", err))
		}
	}()

	logger.Info("sqlc generate start", zap.String("type", targetType), zap.String("category", category), zap.String("yaml", tmpYAML))

	// 3) 一時YAMLを使ってsqlc実行
	// #nosec G204 -- genYAML is generated internally and not from user input
	cmd := exec.CommandContext(ctx, "sqlc", "generate", "-f", tmpYAML)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sqlc generate failed: type: %s, category: %s, exec.Run: %w", targetType, category, err)
	}

	logger.Info("sqlc generate completed", zap.String("type", targetType), zap.String("category", category))

	return nil
}
