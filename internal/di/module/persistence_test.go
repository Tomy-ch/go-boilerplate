package module

import (
	"testing"
)

func Test_persistenceModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 永続化系（repository / query_service / system_cqrs）はリポジトリ追加で増える領域。
	// SQL や実 DB 挙動は infra 層のテストに任せ、ここでは各コンストラクタが DB ハンドル等の
	// 依存と正しく結線されることを確認する。
	opts := append(commonDeps(), persistenceModule())
	validateGraph(t, opts...)
}

func Test_persistenceModule(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
