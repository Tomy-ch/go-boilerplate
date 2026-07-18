// Package auth は認証に関連するインターフェースを提供します。
package auth

import (
	"maps"
	"slices"
	"strings"

	"go-boilerplate/pkg/uuid"
)

const (
	// IssuerMock はモック認証の発行者（issuer）を示します。
	IssuerMock = "mock"
)

// Authn は、認証結果を表します。
// 中核は認証主体 Subject・発行者 Issuer・内部ユーザー UserID の三点です。
type Authn struct {
	subject string         // 認証主体（token の sub）
	userID  *uuid.UUID     // subject を UUID として解釈できた場合の内部ユーザー ID（nil 可能）
	issuer  string         // トークン発行者（例: "mock" / IdP の issuer）
	scopes  []string       // 任意
	claims  map[string]any // 任意（監査・UI制御等）
}

// New は、認証主体や付随情報から認証結果 Authn を生成して返します。
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

	a := &Authn{
		subject: trimmedSubject,
		issuer:  issuer,
		scopes:  slices.Clone(scopes),
		claims:  maps.Clone(claims),
	}

	if id, err := uuid.Parse(trimmedSubject); err == nil {
		a.userID = &id
	}

	return a, nil
}

// Subject は token の sub を返します。
func (a *Authn) Subject() string {
	return a.subject
}

// HasUserID は subject を UUID として解釈できたかを返します。
func (a *Authn) HasUserID() bool {
	return a.userID != nil
}

// UserID は内部ユーザー ID（UUID）を返します。
// subject を UUID として解釈できなかった場合はエラーを返します。
func (a *Authn) UserID() (uuid.UUID, error) {
	if a.userID == nil {
		return uuid.UUID{}, ErrSubjectNotUUID
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
