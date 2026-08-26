package event

import (
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/pkg/xerrors"
)

// errNoWireType は、ドメインの事象に対応するワイヤ表現が未定義の場合のエラーです。
var errNoWireType = xerrors.New("no wire type for purchase event")

// wireTypes は、ドメインの事象からワイヤ上の版付き種別への写像です。
//
// 事象の名前（過去形）はドメインが所有し、版と表現の形はこちらが所有します。同じ事象でも
// payload の形が変われば版が上がるため、両者の寿命は一致しません。
var wireTypes = map[domainpurchase.EventType]string{
	domainpurchase.EventCreated:   TypeCreated,
	domainpurchase.EventPaid:      TypePaid,
	domainpurchase.EventCanceled:  TypeCanceled,
	domainpurchase.EventShipped:   TypeShipped,
	domainpurchase.EventDelivered: TypeDelivered,
}

// WireType は、ドメインの事象に対応する版付きの種別文字列を返します。
// 対応が無い事象は、ワイヤ表現を用意し忘れた実装漏れなのでエラーにします。
func WireType(t domainpurchase.EventType) (string, error) {
	w, ok := wireTypes[t]
	if !ok {
		return "", xerrors.Wrap(errNoWireType, t.Name())
	}
	return w, nil
}
