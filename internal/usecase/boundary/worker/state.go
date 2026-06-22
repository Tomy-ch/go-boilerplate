//go:generate mockgen -source=$GOFILE -destination=mock/mock_state.gen.go -package=mock_$GOPACKAGE

package worker

// State は、起動対象の worker 名と引数を保持します。
// DI で app を構築した後、cmd から選択された worker 名を engine へ引き渡すために使います
// （job の State と同様の役割。worker は常駐のため完了通知 channel は持ちません）。
type State interface {
	// Set は、起動対象の worker 名と引数を設定します。
	Set(name string, args []string)
	// Snapshot は、現在の起動対象をスナップショットとして取得します。
	Snapshot() (name string, args []string)
}
