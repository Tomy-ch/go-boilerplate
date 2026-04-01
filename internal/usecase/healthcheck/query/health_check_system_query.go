//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package query は、システムの健全性チェックに関するクエリを提供します。
package query

import (
	"context"
	"time"
)

type DBSystemQuery interface {
	CheckDBHealth(ctx context.Context) (DBHealth, error)
}

// DBHealth は、データベースの健全性情報を表します。
type DBHealth struct {
	Ready       bool
	ResponsedAt time.Time
	Latency     time.Duration
}
