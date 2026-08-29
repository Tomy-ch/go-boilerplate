package hook

import (
	"context"

	"go-boilerplate/internal/di/lifecycle"
)

// serve instance のライフサイクルに参加する値の fx group 名。参加者はどれも任意（soft group）で、
// 1 つも無いときの起動・停止は HTTP サーバーだけの従来どおりです。
const (
	// ReadinessGroup は、HTTP を listen する前に依存の到達性を確かめる参加者の group です。
	ReadinessGroup = "serve.readiness"
	// ProvisionerGroup は、instance 固有の外部 resource を用意し、停止時に片付ける参加者の group です。
	ProvisionerGroup = "serve.provisioners"
	// RunnerGroup は、listen 中だけ常駐する背景処理の group です。
	RunnerGroup = "serve.runners"
	// DrainerGroup は、HTTP shutdown より前に確定済みの長寿命レスポンスを閉じ切る参加者の group です。
	DrainerGroup = "serve.drainers"
)

// ReadinessProbe は、HTTP を listen する前に依存の到達性を確かめる参加者です。失敗は起動失敗です
// （Realtime runtime の起動に不可欠な dependency は startup で fail-fast する — docs/design/realtime-delivery.md §2.6）。
type ReadinessProbe struct {
	// Name は、ログに載せる参加者の名前です。
	Name string
	// Probe は、依存に到達できるかを確かめます。
	Probe func(ctx context.Context) error
}

// Provisioner は、instance 固有の外部 resource を起動時に用意し、停止時に片付ける参加者です。
// 起動が途中で失敗したら、用意済みの参加者は逆順に Teardown されます。
type Provisioner struct {
	// Name は、ログに載せる参加者の名前です。
	Name string
	// Provision は、resource を用意します。
	Provision func(ctx context.Context) error
	// Teardown は、resource を片付けます。Provision していなくても呼べる（何もしない）実装にします。
	Teardown func(ctx context.Context) error
}

// Runner は、HTTP を listen している間だけ常駐する背景処理です。契約は lifecycle.SupervisedRunner と同じ
// （detached、停止時に cancel して完了を猶予の範囲で待つ）。
type Runner struct {
	// Name は、ログに載せる参加者の名前です。
	Name string
	// Runner は、常駐処理の本体です。
	Runner lifecycle.SupervisedRunner
}

// Drainer は、HTTP shutdown より前に、確定済みの長寿命レスポンス（SSE 接続）を閉じ切る参加者です。
// 新規接続の拒否もこの中で行います。Drain が返るまで常駐処理の停止と HTTP shutdown は始まりません。
type Drainer struct {
	// Name は、ログに載せる参加者の名前です。
	Name string
	// Drain は、接続を閉じ切るまで待ちます。ctx の猶予を超えたら残りを諦めて返します。
	Drain func(ctx context.Context) error
}
