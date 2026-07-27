package module

import (
	"testing"
)

func TestUsecaseModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// ユースケース層は機能追加で増える領域。各ユースケースの振る舞いは usecase 層のテストに任せ、
	// ここではコンストラクタがリポジトリ等の依存と正しく結線されることを確認する。
	opts := append(commonDeps(), InfrastructureModule(), UsecaseModule())
	validateGraph(t, opts...)
}

func TestUsecaseModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

// sample-api:begin
func Test_provideProductUsecase(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

// sample-api:end
