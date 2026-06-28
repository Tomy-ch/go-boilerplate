package authz

import (
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
)

// Resource は、認可対象のリソースを表します。
// 所有権ベースの判定（呼出元 == 所有者）に必要な最小情報を持ちます。
type Resource struct {
	kind    string     // リソース種別（例: "user"）
	ownerID *uuid.UUID // リソース所有者の ID（所有者概念がない場合は nil）
}

// NewResource は、種別と所有者 ID から Resource を生成します。
// ownerID は防御的にコピーして保持するため、呼出元が渡したポインタの後続変更や
// スタック寿命に Resource は影響されません（Authorize が非同期実装でも安全）。
func NewResource(kind string, ownerID *uuid.UUID) *Resource {
	return &Resource{kind: kind, ownerID: ptr.Copy(ownerID)}
}

// Kind は、リソース種別を返します。
func (r *Resource) Kind() string { return r.kind }

// OwnerID は、リソース所有者の ID を返します（nil の場合は所有者概念なし）。
// 内部状態を保護するためコピーを返します。
func (r *Resource) OwnerID() *uuid.UUID { return ptr.Copy(r.ownerID) }
