//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import "context"

// OrphanReclaimer は、死んだ instance が残した受信先を識別子から辿って片付ける境界です。
// InstanceSubscription が自分の生存期間に閉じるのに対し、こちらは他の instance の残骸を対象にします。
// 対象が無いことは成功で、同じ識別子に対して繰り返し呼んでも同じ状態に収束します。
// 失敗は apperror sentinel で返します。
type OrphanReclaimer interface {
	// Reclaim は、id の instance の登録を解除して受信先を削除します。どちらも無ければ何もしません。
	Reclaim(ctx context.Context, id InstanceID) error
}
