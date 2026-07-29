//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package address は、郵便番号 lookup gateway を用いた住所補完ユースケースのサンプルです。
package address

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/address"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Usecase は、郵便番号住所補完ユースケースを表します。
type Usecase interface {
	// LookupByPostalCode は、正規化済み 7 桁の郵便番号から住所候補を返します。
	// 外部 lookup 障害時はエラーを返さず IsFallback: true の空候補で degrade します。
	LookupByPostalCode(ctx context.Context, postalCode string) (*Result, error)
}

// Result は、住所補完結果の出力 DTO です。
type Result struct {
	// Candidates は、住所候補の一覧です。該当なし・degrade 時は空スライスです。
	Candidates []*CandidateView
	// IsFallback は、lookup 機構が機能しなかった（外部不通・不正応答）場合に true です。
	IsFallback bool
}

// CandidateView は、住所候補 1 件の出力 DTO です。
type CandidateView struct {
	// PrefectureID は、都道府県ID です。県名を prefectures マスタで解決できなかった場合は nil です。
	PrefectureID *uuid.UUID
	// PrefectureName は、都道府県名（フル表記）です。
	PrefectureName string
	// City は、市区町村です。
	City string
	// Town は、町域です。
	Town string
}

// usecase は、Usecase の実装です。
type usecase struct {
	gateway     boundary.Gateway
	prefectureQ prefecture.Repository
	tracer      observability.LayerTracer
}

// New は、住所補完ユースケースを生成します。
func New(
	gateway boundary.Gateway,
	prefectureQ prefecture.Repository,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		gateway:     gateway,
		prefectureQ: prefectureQ,
		tracer:      tf.Usecase(),
	}
}

func (u *usecase) LookupByPostalCode(ctx context.Context, postalCode string) (*Result, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	candidates, err := u.gateway.Lookup(ctx, postalCode)
	if err != nil {
		// 外部 lookup 障害は 503 で止めず degrade する（exchangerate の 503 とは正反対に登録を止めない）。
		// 障害自体は gateway の Infra span と httpclient substrate の downstream 失敗メトリクスで観測できるため、
		// ここでは error を握り、IsFallback:true の空候補を返す。
		return &Result{Candidates: []*CandidateView{}, IsFallback: true}, nil
	}

	// 同一郵便番号は同一県名がほとんどのため、県名で解決結果を memo 化し FindByName の呼び出しを最小化する。
	// 県名を prefectures マスタで完全一致解決し ID を埋める。解決不能（NotFound）は部分 degrade として
	// PrefectureID:nil のまま候補を残し、それ以外のエラー（DB 障害等）は degrade せず伝播する。
	resolved := make(map[string]*uuid.UUID, 1)
	views := make([]*CandidateView, 0, len(candidates))
	for _, c := range candidates {
		prefID, ok := resolved[c.PrefectureName]
		if !ok {
			p, ferr := u.prefectureQ.FindByName(ctx, c.PrefectureName)
			switch {
			case ferr == nil:
				prefID = p.ID().ToPtr()
			case xerrors.Is(ferr, apperror.ErrNotFound):
				prefID = nil
			default:
				return nil, ferr
			}
			resolved[c.PrefectureName] = prefID
		}
		views = append(views, &CandidateView{
			PrefectureID:   prefID,
			PrefectureName: c.PrefectureName,
			City:           c.City,
			Town:           c.Town,
		})
	}

	return &Result{Candidates: views, IsFallback: false}, nil
}
