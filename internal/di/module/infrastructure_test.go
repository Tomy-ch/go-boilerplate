package module

import (
	"testing"
)

func TestInfrastructureModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// インフラ層はリポジトリ／クエリサービス追加で増える領域。SQL や実 DB 挙動は infra 層の
	// テストに任せ、ここでは各コンストラクタが DB ハンドル等の依存と正しく結線されることを確認する。
	opts := append(commonDeps(), InfrastructureModule())
	validateGraph(t, opts...)
}
