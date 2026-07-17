//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package query は、システムの健全性チェックに関するクエリを提供します。
package query

import (
	"context"
	"time"
)

// DBSystemCqrs は、データベースの健全性を確認するシステムクエリのインターフェースです。
type DBSystemCqrs interface {
	CheckDBHealth(ctx context.Context) (DBHealth, error)
}

// DBHealth は、データベースの健全性情報を表します。
type DBHealth struct {
	Ready       bool
	RespondedAt time.Time
	Latency     time.Duration
}
