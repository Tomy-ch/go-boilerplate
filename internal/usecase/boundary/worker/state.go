//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package worker

// State は、起動対象の worker 名・引数と、engine の実行結果を返す done チャネルを保持します。
type State interface {
	// Set は、起動対象の worker 名・引数と done チャネルを設定します。
	// done はバッファ付き（cap≥1）であること。送信・close は受け手（Snapshot 側）が所有し、
	// engine の実行結果を 1 度だけ送って close します。
	Set(name string, args []string, done chan error)
	// Snapshot は、現在の起動対象と done チャネルをスナップショットとして取得します。
	Snapshot() (name string, args []string, done chan error)
}
