// Package auth は認証に関連するインターフェースを提供します。
package auth

import (
	"maps"
	"slices"
	"strings"

	"go-boilerplate/pkg/uuid"
)

const (
	// ProviderMock はモック認証プロバイダを示します。
	ProviderMock = "mock"
)

// Authn は、認証結果を表します。
type Authn struct {
	subject  string         // 認証主体（例: userID）
	id       *uuid.UUID     // subject を UUID として解釈できた場合の ID（nil 可能）
	provider string         // "mock" / "auth-server" / "google" 等
	scopes   []string       // 任意
	claims   map[string]any // 任意（監査・UI制御等）
}

// New は、認証主体や付随情報から認証結果 Authn を生成して返します。
func New(
	subject string,
	provider string,
	scopes []string,
	claims map[string]any,
) (*Authn, error) {
	trimmedSubject := strings.TrimSpace(subject)
	if trimmedSubject == "" {
		return nil, ErrUnauthenticatedSubjectMissing
	}

	a := &Authn{
		subject:  trimmedSubject,
		provider: provider,
		scopes:   slices.Clone(scopes),
		claims:   maps.Clone(claims),
	}

	if id, err := uuid.Parse(trimmedSubject); err == nil {
		a.id = &id
	}

	return a, nil
}

// Subject は token の sub を返します。
func (a *Authn) Subject() string {
	return a.subject
}

// HasID は UUIDとして解釈できたかを返します。
func (a *Authn) HasID() bool {
	return a.id != nil
}

// ID は UUID を返します。
// UUID として解釈できなかった場合はエラーを返します。
func (a *Authn) ID() (uuid.UUID, error) {
	if a.id == nil {
		return uuid.UUID{}, ErrSubjectNotUUID
	}
	return *a.id, nil
}

// Provider は認証プロバイダを返します。
func (a *Authn) Provider() string {
	return a.provider
}

// Scopes はスコープ一覧を返します。
func (a *Authn) Scopes() []string {
	return slices.Clone(a.scopes)
}

// Claims はクレーム一覧を返します。
func (a *Authn) Claims() map[string]any {
	return maps.Clone(a.claims)
}
