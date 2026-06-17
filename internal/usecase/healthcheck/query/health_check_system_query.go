//go:generate mockgen -source=$GOFILE -destination=mock/mock_health_check_system_query.gen.go -package=mock_$GOPACKAGE

// Package query は、システムの健全性チェックに関するクエリを提供します。
package query

import (
	"context"
	"time"
)

// DBSystemQuery は、データベースの健全性を確認するシステムクエリのインターフェースです。
type DBSystemQuery interface {
	CheckDBHealth(ctx context.Context) (DBHealth, error)
}

// DBHealth は、データベースの健全性情報を表します。
type DBHealth struct {
	Ready       bool
	RespondedAt time.Time
	Latency     time.Duration
}
