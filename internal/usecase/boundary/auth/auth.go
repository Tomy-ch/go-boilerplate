// Package auth は認証に関連するインターフェースを提供します。
package auth

import (
	"strings"

	"boilerplate-go/pkg/uuid"
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

func New(
	subject string,
	provider string,
	scopes []string,
	claims map[string]any,
) (*Authn, error) {
	trimmedSubject := strings.TrimSpace(subject)
	if trimmedSubject == "" {
		return nil, ErrUnauthorizedSubjectMissing
	}

	p := &Authn{
		subject:  trimmedSubject,
		provider: provider,
		scopes:   scopes,
		claims:   claims,
	}

	// subject が UUID なら id を埋める（変換できない場合は nil のまま）
	if id, err := uuid.Parse(trimmedSubject); err == nil {
		p.id = &id
	}

	return p, nil
}

// Subject は token の sub を返します。
func (p *Authn) Subject() string {
	return p.subject
}

// HasID は UUIDとして解釈できたかを返します。
func (p *Authn) HasID() bool {
	return p.id != nil
}

// ID は UUID を返します。
// UUID として解釈できなかった場合はエラーを返します。
func (p *Authn) ID() (uuid.UUID, error) {
	if p.id == nil {
		return uuid.UUID{}, ErrInvalidIDMissing
	}
	return *p.id, nil
}

// Provider は認証プロバイダを返します。
func (p *Authn) Provider() string {
	return p.provider
}

// Scopes はスコープ一覧を返します。
func (p *Authn) Scopes() []string {
	return p.scopes
}

// Claims はクレーム一覧を返します。
func (p *Authn) Claims() map[string]any {
	return p.claims
}
