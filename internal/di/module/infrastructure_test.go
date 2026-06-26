package module

import (
	"testing"
)

func TestInfrastructureModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 各 concern（persistence / clock / httpclient / webapi / outbox_publisher / security）の
	// 個別検証は各 concern の *_test.go に切り出し、ここでは集約された InfrastructureModule 全体が
	// 欠落なく結線されることを確認する。SQL や実 DB 挙動は infra 層のテストに任せる。
	opts := append(commonDeps(), InfrastructureModule())
	validateGraph(t, opts...)
}
