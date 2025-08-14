// Package user は、ユーザーに関連するドメインロジックを提供します。
package user

import (
	"time"

	"boilerplate-go/pkg/uuid"
)

type User struct {
	id             uuid.UUID
	firstName      string
	lastName       string
	email          string
	phone          string
	prefectureID   string
	prefectureName string
	city           string
	street         string
	building       *string
	postalCode     string
	deletedAt      *time.Time
}

func (u *User) ID() uuid.UUID {
	return u.id
}

func (u *User) FirstName() string {
	return u.firstName
}

func (u *User) LastName() string {
	return u.lastName
}

func (u *User) Email() string {
	return u.email
}

func (u *User) Phone() string {
	return u.phone
}

func (u *User) PrefectureID() string {
	return u.prefectureID
}

func (u *User) PrefectureName() string {
	return u.prefectureName
}

func (u *User) City() string {
	return u.city
}

func (u *User) Street() string {
	return u.street
}

func (u *User) Building() *string {
	return u.building
}

func (u *User) PostalCode() string {
	return u.postalCode
}

func (u *User) DeletedAt() *time.Time {
	return u.deletedAt
}
