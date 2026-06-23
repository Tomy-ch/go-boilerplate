//go:generate mockgen -source=$GOFILE -destination=mock/mock_state.gen.go -package=mock_$GOPACKAGE

package worker

// State は、起動対象の worker 名・引数と、engine の実行結果を返す done チャネルを保持します。
type State interface {
	// Set は、起動対象の worker 名・引数と done チャネルを設定します。
	// done はバッファ付き（cap≥1）であること。engine 停止時に結果を 1 度送って close します。
	Set(name string, args []string, done chan error)
	// Snapshot は、現在の起動対象と done チャネルをスナップショットとして取得します。
	Snapshot() (name string, args []string, done chan error)
}
