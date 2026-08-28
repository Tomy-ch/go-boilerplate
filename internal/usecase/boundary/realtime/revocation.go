//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import "context"

// RevocationNotifier は、subject の destination への権利が取り下げられたことを、接続を抱える全 serve instance へ伝える境界です。
// 受け取った instance は該当する接続を STOP で閉じます（ADR-0074）。失敗は apperror sentinel で返します。
type RevocationNotifier interface {
	// NotifyRevoked は、subject × destination の失効を全 instance へ通知します。
	NotifyRevoked(ctx context.Context, subject string, destination StreamID) error
}
