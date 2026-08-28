//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// TicketTTL は、発行した ticket で新しい接続を始められる期間です（ADR-0074）。
// 確立済みの接続はこれとは別に最大接続時間で区切られます。
const TicketTTL = 5 * time.Minute

// IssueTicketInput は、feature が subject × destination を認可した後に渡す発行要求です。
type IssueTicketInput struct {
	// Subject は、認可された subject です。
	Subject string
	// Destination は、接続を許す stream です。
	Destination rt.StreamID
	// Scope は、feature が定めた権限の範囲です。機構は解釈しません。
	Scope string
	// InitialCursor は、cursor 無しで接続したときの開始位置です。認可の下限ではありません
	// （design/realtime-delivery.md §2.3）。
	InitialCursor rt.Sequence
}

// TicketView は、発行の結果として client に 1 度だけ渡す ticket の生値と期限です。生値はどこにも保存されません。
type TicketView struct {
	Value     string
	ExpiresAt time.Time
}

// TicketIssuer は、ticket を発行します。feature の ticket-issuing usecase が認可の後に呼びます。
type TicketIssuer interface {
	// Issue は、生値を生成し、その hash を bindings と共に保存して、生値と期限を返します。
	Issue(ctx context.Context, in IssueTicketInput) (TicketView, error)
}

// TicketVerifier は、接続時に提示された ticket を検証します。
type TicketVerifier interface {
	// Verify は、生値の hash に対応する ticket が期限内にあり、destination が一致するときその束縛（StreamGrant）を
	// 返します。無い・期限切れ・destination 違いはいずれも ErrTicketInvalid です（理由は区別しない —
	// 区別すると存在の有無を推測する手がかりになるため）。store が読めなければ apperror.ErrUnavailable です。
	Verify(ctx context.Context, value string, destination rt.StreamID) (rt.StreamGrant, error)
}

type ticketService struct {
	store   rt.StreamTicketStore
	secrets rt.SecretGenerator
	clock   clock.Clock
	tracer  observability.LayerTracer
}

// NewTicketIssuer は、TicketIssuer を生成します。
func NewTicketIssuer(store rt.StreamTicketStore, secrets rt.SecretGenerator, clk clock.Clock, tf observability.TracerFactory) TicketIssuer {
	return newTicketService(store, secrets, clk, tf)
}

// NewTicketVerifier は、TicketVerifier を生成します。
func NewTicketVerifier(store rt.StreamTicketStore, clk clock.Clock, tf observability.TracerFactory) TicketVerifier {
	return newTicketService(store, nil, clk, tf)
}

func newTicketService(store rt.StreamTicketStore, secrets rt.SecretGenerator, clk clock.Clock, tf observability.TracerFactory) *ticketService {
	return &ticketService{store: store, secrets: secrets, clock: clk, tracer: tf.Usecase()}
}

func (s *ticketService) Issue(ctx context.Context, in IssueTicketInput) (TicketView, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	value, err := s.secrets.Generate()
	if err != nil {
		return TicketView{}, xerrors.Wrap(err, "issue ticket")
	}

	now := s.clock.Now()
	ticket := rt.StreamTicket{
		Hash:          hashTicket(value),
		Subject:       in.Subject,
		Destination:   in.Destination,
		Scope:         in.Scope,
		InitialCursor: in.InitialCursor,
		IssuedAt:      now,
		ExpiresAt:     now.Add(TicketTTL),
	}
	if err := s.store.Save(ctx, ticket); err != nil {
		return TicketView{}, err
	}

	return TicketView{Value: value, ExpiresAt: ticket.ExpiresAt}, nil
}

func (s *ticketService) Verify(ctx context.Context, value string, destination rt.StreamID) (rt.StreamGrant, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	if value == "" {
		return rt.StreamGrant{}, xerrors.Wrap(ErrTicketInvalid, "ticket is empty")
	}

	ticket, ok, err := s.store.Find(ctx, hashTicket(value), s.clock.Now())
	if err != nil {
		return rt.StreamGrant{}, err
	}

	if !ok {
		return rt.StreamGrant{}, xerrors.Wrap(ErrTicketInvalid, "ticket is unknown or expired")
	}

	if ticket.Destination != destination {
		return rt.StreamGrant{}, xerrors.Wrap(ErrTicketInvalid, "ticket is bound to another destination")
	}

	return rt.StreamGrant{
		Subject: ticket.Subject, Destination: ticket.Destination, Scope: ticket.Scope, InitialCursor: ticket.InitialCursor,
	}, nil
}

// hashTicket は、生値の保存形（SHA-256 の 16 進）を返します。生値は 256 bit の乱数なので salt は要りません。
func hashTicket(value string) rt.TicketHash {
	sum := sha256.Sum256([]byte(value))

	return rt.TicketHash(hex.EncodeToString(sum[:]))
}
