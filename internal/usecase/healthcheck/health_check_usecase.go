//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package healthcheck は、システムの健全性チェックに関するユースケースを提供します。
package healthcheck

import (
	"context"
	"time"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/healthcheck/query"
)

// ヘルスチェックの総合ステータスを表す値。
const (
	// Degraded は、一部の依存が不調だが稼働継続できる状態を表します。
	Degraded = "degraded"
	// Ok は、すべて正常な状態を表します。
	Ok = "ok"
	// Unhealthy は、サービスが正常に応答できない状態を表します。
	Unhealthy = "unhealthy"
)

// Probe は、readiness に加わる依存 1 つ分の到達性検査です。
// 名前は /ready の応答にそのまま載るので、subsystem の名前（realtime 等）を与えます。
//
// 検査の失敗は degraded であって unhealthy ではありません。ここに並ぶのは、
// 落ちていても通常の HTTP 応答は続けられる依存だけです
// （不可欠な依存は起動時の fail-fast が受け持ちます。docs/design/realtime-delivery.md §2.6）。
type Probe struct {
	// Name は、応答に載せる依存の名前です。
	Name string
	// Check は、依存へ到達できるかを確かめます。nil を返せば健全です。
	Check func(ctx context.Context) error
}

// DependencyStatus は、依存 1 つの状態です。
type DependencyStatus struct {
	// Name は、Probe に与えられた名前です。
	Name string
	// Status は、Ok か Degraded のどちらかです。
	Status string
}

// probeTimeout は、依存 1 つの検査に与える上限です。load balancer の probe timeout より
// 十分短く取り、遅い依存が応答時間の側で instance を落とさないようにします。
const probeTimeout = 1 * time.Second

// DTO は、システムの健全性に関するデータ転送用のオブジェクトです。
type DTO struct {
	Status          string
	ApplicationTime time.Time
	DBHealthCheck   query.DBHealth
	// Dependencies は、readiness に加わった依存の状態です。probe が 1 つも無ければ空です。
	Dependencies []DependencyStatus
}

type usecase struct {
	tracer       observability.LayerTracer
	clock        clock.Clock
	dbSystemCqrs query.DBSystemCqrs
	probes       []Probe
}

// Usecase は、システムの健全性チェックに関するユースケースを定義します。
type Usecase interface {
	// CheckHealth は健全性 DTO を返します。異常時は nil を返し、DTO は参照しないこと。
	CheckHealth(ctx context.Context) (*DTO, error)
}

// New は、システムの健全性チェックに関するユースケースを初期化します。
// probes は、落ちていても通常の HTTP 応答を続けられる依存の検査です（空でも構いません）。
func New(dbsq query.DBSystemCqrs, tf observability.TracerFactory, clock clock.Clock, probes []Probe) Usecase {
	return &usecase{
		tracer:       tf.Usecase(),
		clock:        clock,
		dbSystemCqrs: dbsq,
		probes:       probes,
	}
}

func (u *usecase) CheckHealth(ctx context.Context) (*DTO, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	applTime := u.clock.Now()
	dbHealth, err := u.dbSystemCqrs.CheckDBHealth(ctx)
	if err != nil {
		return nil, err
	}

	deps, status := u.checkDependencies(ctx)

	return &DTO{
		Status:          status,
		ApplicationTime: applTime,
		DBHealthCheck:   dbHealth,
		Dependencies:    deps,
	}, nil
}

// checkDependencies は、各 probe を検査して状態の一覧と総合ステータスを返します。
// 1 つでも落ちていれば degraded ですが、エラーにはしません — ここで 503 を返すと、
// Realtime だけが落ちた instance が load balancer から外れ、通常の HTTP まで止まります。
func (u *usecase) checkDependencies(ctx context.Context) ([]DependencyStatus, string) {
	if len(u.probes) == 0 {
		return nil, Ok
	}

	deps := make([]DependencyStatus, 0, len(u.probes))
	status := Ok
	for _, p := range u.probes {
		depStatus := Ok
		if u.checkProbe(ctx, p) != nil {
			depStatus = Degraded
			status = Degraded
		}

		deps = append(deps, DependencyStatus{Name: p.Name, Status: depStatus})
	}

	return deps, status
}

// checkProbe は、1 つの probe を有界時間で検査します。超過は degraded として扱います。
//
// 上限が要るのは、応答しない依存が status ではなく**応答時間**の側から要件を破るためです。
// 縮退を 200 で返しても、load balancer の probe timeout（ALB 既定 5 秒、Kubernetes の
// readinessProbe 既定 1 秒）を超えて黙っていれば instance は結局そこから外れ、
// 「Realtime だけの不調で通常の HTTP を止めない」が成り立ちません。
func (u *usecase) checkProbe(ctx context.Context, p Probe) error {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	return p.Check(ctx)
}
