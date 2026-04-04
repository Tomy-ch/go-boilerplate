// Package system は、システム全体で使用される共通のインフラストラクチャを提供します。
package system

import (
	"time"

	"boilerplate-go/internal/usecase/boundary/clock"
)

type c struct{}

// NewClock は、clockを生成します。
func NewClock() clock.Clock {
	return &c{}
}

// Now は、現在の時刻を返します。
func (c *c) Now() time.Time {
	return time.Now()
}
