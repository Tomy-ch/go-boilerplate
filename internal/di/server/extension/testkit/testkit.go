// Package testkit は、di server extension のモジュールテスト用の共有ヘルパーを提供します。
package testkit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// RequireProvidesOne は、与えたモジュール群が指定 group に要素をちょうど1件 provide することを検証する。
func RequireProvidesOne[T any](t *testing.T, group string, opts ...fx.Option) {
	t.Helper()

	var got []T
	app := fx.New(append(opts,
		fx.Populate(fx.Annotate(&got, fx.ParamTags(`group:"`+group+`"`))),
		fx.NopLogger,
	)...)

	require.NoError(t, app.Start(context.Background()))
	defer func() { require.NoError(t, app.Stop(context.Background())) }()

	require.Len(t, got, 1)
}
