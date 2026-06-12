// Package system は、システム全体で使用される共通のインフラストラクチャを提供します。
package system

import (
	"time"

	"go-boilerplate/internal/usecase/boundary/clock"
)

type systemClock struct{}

// NewClock は、clockを生成します。
func NewClock() clock.Clock {
	return &systemClock{}
}

// Now は、現在の時刻を返します。
func (sc *systemClock) Now() time.Time {
	return time.Now()
}
