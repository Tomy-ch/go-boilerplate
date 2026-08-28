//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

// SecretGenerator は、ticket の生値（256 bit の推測不能な不透明値）を生成する境界です。
// 乱数を usecase から隔離するための seam で、feature 側の token 境界とは独立に持ちます
// （sample feature を削除しても Realtime Delivery が成り立つため）。
type SecretGenerator interface {
	// Generate は、新しい生値を返します。呼ぶたびに異なり、生成済みの値から次を推測できません。
	// 乱数源が使えなければエラーを返します。
	Generate() (string, error)
}
