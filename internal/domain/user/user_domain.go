// Package user は、ユーザー関連のドメインを提供します。
package user

import (
	"time"

	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/stringkit"
	"boilerplate-go/pkg/uuid"
	"boilerplate-go/pkg/xerrors"
)

type Users []*User

type User struct {
	id           uuid.UUID
	firstName    string
	lastName     string
	passwordHash string
	email        string
	phone        string
	prefectureID uuid.UUID
	city         string
	street       string
	building     *string
	postalCode   string
	createdAt    time.Time
	updatedAt    time.Time
	deletedAt    *time.Time
}

// New は、ユーザーエンティティの検証と生成を行います。
func New(
	id uuid.UUID,
	firstName string,
	lastName string,
	passwordHash string,
	email string,
	phone string,
	prefectureID uuid.UUID,
	city string,
	street string,
	building *string,
	postalCode string,
	createdAt time.Time,
	updatedAt time.Time,
	deletedAt *time.Time,
) (*User, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}

	if !stringkit.InRange(firstName, minLength, maxFirstNameLength) {
		return nil, xerrors.Wrap(ErrInvalidFirstName, stringkit.ErrorMsgInRange(minLength, maxFirstNameLength, firstName))
	}

	if !stringkit.InRange(lastName, minLength, maxLastNameLength) {
		return nil, xerrors.Wrap(ErrInvalidLastName, stringkit.ErrorMsgInRange(minLength, maxLastNameLength, lastName))
	}

	if !stringkit.InRange(passwordHash, minLength, maxPasswordLength) {
		return nil, xerrors.Wrap(ErrInvalidPassword, stringkit.ErrorMsgInRange(minLength, maxPasswordLength, passwordHash))
	}

	if !stringkit.InRange(email, minLength, maxEmailLength) {
		return nil, xerrors.Wrap(ErrInvalidEmail, stringkit.ErrorMsgInRange(minLength, maxEmailLength, email))
	}

	if !stringkit.InRange(phone, minLength, maxPhoneLength) {
		return nil, xerrors.Wrap(ErrInvalidPhone, stringkit.ErrorMsgInRange(minLength, maxPhoneLength, phone))
	}

	if prefectureID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidPrefectureID, "prefectureID is required")
	}

	if !stringkit.InRange(city, minLength, maxCityLength) {
		return nil, xerrors.Wrap(ErrInvalidCity, stringkit.ErrorMsgInRange(minLength, maxCityLength, city))
	}

	if !stringkit.InRange(street, minLength, maxStreetLength) {
		return nil, xerrors.Wrap(ErrInvalidStreet, stringkit.ErrorMsgInRange(minLength, maxStreetLength, street))
	}

	if building != nil && !stringkit.InRange(*building, minLength, maxBuildingLength) {
		return nil, xerrors.Wrap(ErrInvalidBuilding, stringkit.ErrorMsgInRange(minLength, maxBuildingLength, *building))
	}

	if !stringkit.InRange(postalCode, minLength, maxPostalCodeLength) {
		return nil, xerrors.Wrap(ErrInvalidPostalCode, stringkit.ErrorMsgInRange(minLength, maxPostalCodeLength, postalCode))
	}

	if updatedAt.Before(createdAt) {
		return nil, xerrors.Wrap(ErrInvalidUpdatedAt, "updatedAt must be after or equal to createdAt")
	}

	if deletedAt != nil && deletedAt.Before(createdAt) {
		return nil, xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to createdAt")
	}

	if deletedAt != nil && deletedAt.Before(updatedAt) {
		return nil, xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to updatedAt")
	}

	return &User{
		id:           id,
		firstName:    firstName,
		lastName:     lastName,
		passwordHash: passwordHash,
		email:        email,
		phone:        phone,
		prefectureID: prefectureID,
		city:         city,
		street:       street,
		building:     ptr.Copy(building),
		postalCode:   postalCode,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		deletedAt:    ptr.Copy(deletedAt),
	}, nil
}

// ID は、ユーザーのIDを返します。
func (u *User) ID() uuid.UUID { return u.id }

// FirstName は、ユーザーの名前を返します。
func (u *User) FirstName() string { return u.firstName }

// LastName は、ユーザーの名字を返します。
func (u *User) LastName() string { return u.lastName }

// PasswordHash は、ユーザーのパスワードハッシュを返します。
func (u *User) PasswordHash() string { return u.passwordHash }

// Email は、ユーザーのメールアドレスを返します。
func (u *User) Email() string { return u.email }

// Phone は、ユーザーの電話番号を返します。
func (u *User) Phone() string { return u.phone }

// PrefectureID は、ユーザーの都道府県IDを返します。
func (u *User) PrefectureID() uuid.UUID { return u.prefectureID }

// City は、ユーザーの市区町村名を返します。
func (u *User) City() string { return u.city }

// Street は、ユーザーの番地を返します。
func (u *User) Street() string { return u.street }

// Building は、ユーザーの建物名を返します。
func (u *User) Building() *string { return ptr.Copy(u.building) }

// PostalCode は、ユーザーの郵便番号を返します。
func (u *User) PostalCode() string { return u.postalCode }

// DeletedAt は、ユーザーの削除日時を返します。
func (u *User) DeletedAt() *time.Time { return ptr.Copy(u.deletedAt) }

// CreatedAt は、ユーザーの作成日時を返します。
func (u *User) CreatedAt() time.Time { return u.createdAt }

// UpdatedAt は、ユーザーの更新日時を返します。
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// FullName は、ユーザーのフルネームを返します。
func (u *User) FullName() string { return u.firstName + " " + u.lastName }
