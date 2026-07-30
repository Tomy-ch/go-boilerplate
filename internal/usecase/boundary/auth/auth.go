// Package auth は認証に関連するインターフェースを提供します。
package auth

import (
	"maps"
	"slices"
	"strings"

	"go-boilerplate/pkg/uuid"
)

// IssuerMock はモック認証の発行者（issuer）を示します。
const IssuerMock = "mock"

// Authn は、認証結果を表します。
// 中核は認証主体 Subject・発行者 Issuer・内部ユーザー UserID の三点です。
// UserID は WithUserID で設定されるまで未解決（nil）で、解決済みの UserID はゼロ値ではないことが保証されます。
type Authn struct {
	subject string         // 認証主体（token の sub）
	userID  *uuid.UUID     // 内部ユーザー ID。未解決なら nil。
	issuer  string         // トークン発行者（例: "mock" / IdP の issuer）
	scopes  []string       // 任意
	claims  map[string]any // 任意（監査・UI制御等）
}

// New は、認証主体や付随情報から認証結果 Authn を生成して返します。
// subject が空の場合は ErrUnauthenticatedSubjectMissing を返します。返す Authn の UserID は未解決（nil）です。
func New(
	subject string,
	issuer string,
	scopes []string,
	claims map[string]any,
) (*Authn, error) {
	trimmedSubject := strings.TrimSpace(subject)
	if trimmedSubject == "" {
		return nil, ErrUnauthenticatedSubjectMissing
	}

	return &Authn{
		subject: trimmedSubject,
		issuer:  issuer,
		scopes:  slices.Clone(scopes),
		claims:  maps.Clone(claims),
	}, nil
}

// WithUserID は、内部ユーザー ID を解決した複製を返します（元の Authn は変更しません）。
// userID がゼロ値の場合は ErrUserIDZero を返します。
func (a *Authn) WithUserID(userID uuid.UUID) (*Authn, error) {
	if userID.IsNil() {
		return nil, ErrUserIDZero
	}

	cloned := *a
	cloned.userID = &userID
	return &cloned, nil
}

// Subject は token の sub を返します。
func (a *Authn) Subject() string {
	return a.subject
}

// HasUserID は内部ユーザー ID が解決済みかを返します。
func (a *Authn) HasUserID() bool {
	return a.userID != nil
}

// UserID は内部ユーザー ID（UUID）を返します。
// 未解決の場合は ErrUserIDUnresolved を返します。
func (a *Authn) UserID() (uuid.UUID, error) {
	if a.userID == nil {
		return uuid.UUID{}, ErrUserIDUnresolved
	}
	return *a.userID, nil
}

// Issuer はトークン発行者を返します。
func (a *Authn) Issuer() string {
	return a.issuer
}

// Scopes はスコープ一覧を返します。
func (a *Authn) Scopes() []string {
	return slices.Clone(a.scopes)
}

// Claims はクレーム一覧を返します。
func (a *Authn) Claims() map[string]any {
	return maps.Clone(a.claims)
}
