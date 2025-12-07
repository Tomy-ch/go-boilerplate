package driver

import (
	"database/sql"
	"testing"

	"boilerplate-go/pkg/ptr"
)

// NewTestInstance は、テスト用のDatabaseDriverインスタンスを生成します。
func NewTestInstance(t *testing.T) DatabaseDriver {
	t.Helper()
	return &dbDriver{DB: ptr.To(sql.DB{})}
}
