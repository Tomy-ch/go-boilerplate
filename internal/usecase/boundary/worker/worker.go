//go:generate mockgen -source=$GOFILE -destination=mock/mock_worker.gen.go -package=mock_$GOPACKAGE

package worker

// Worker は、選択起動の単位です。1 worker type = 1 Worker。
// engine は Name で worker を選択し、その Consumer から引いて Handler で処理します。
type Worker interface {
	// Name は、worker type の名前を返します（サブコマンド引数で選択されます）。
	Name() string
	// Consumer は、この worker が消費するキューの Consumer を返します。
	Consumer() Consumer
	// Handler は、業務処理を返します。
	Handler() Handler
	// FailureHandler は、Permanent メッセージの退避先を返します（nil なら engine 既定）。
	FailureHandler() FailureHandler
}
