// Package user は、ユーザー関連のドメインを提供します。
package user

import (
	"time"

	"boilerplate-go/internal/domain/user/prefecture"
	"boilerplate-go/pkg/stringkit"
	"boilerplate-go/pkg/uuid"
	"boilerplate-go/pkg/xerrors"
)

type Entities []Entity

type Entity struct {
	id             uuid.UUID
	firstName      string
	lastName       string
	password       string
	email          string
	phone          string
	prefectureID   uuid.UUID
	prefectureName string
	city           string
	street         string
	building       *string
	postalCode     string
	deletedAt      *time.Time
}

// New は、ユーザーエンティティの検証と生成を行います。
func New(
	idStr string,
	firstName string,
	lastName string,
	password string,
	email string,
	phone string,
	prefectureIDStr string,
	prefectureName string,
	city string,
	street string,
	building *string,
	postalCode string,
	deletedAt *time.Time,
) (*Entity, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, xerrors.Wrap(ErrInvalidID, err.Error())
	}

	if !stringkit.InRange(firstName, minLength, maxFirstNameLength) {
		return nil, xerrors.Wrap(ErrInvalidFirstName, stringkit.ErrorMsgInRange(minLength, maxFirstNameLength, firstName))
	}

	if !stringkit.InRange(lastName, minLength, maxLastNameLength) {
		return nil, xerrors.Wrap(ErrInvalidLastName, stringkit.ErrorMsgInRange(minLength, maxLastNameLength, lastName))
	}

	if !stringkit.InRange(prefectureName, prefecture.MinLength, prefecture.MaxPrefectureNameLength) {
		return nil, xerrors.Wrap(
			ErrInvalidPrefectureName, stringkit.ErrorMsgInRange(prefecture.MinLength, prefecture.MaxPrefectureNameLength, prefectureName),
		)
	}

	if !stringkit.InRange(password, minLength, maxPasswordLength) {
		return nil, xerrors.Wrap(ErrInvalidPassword, stringkit.ErrorMsgInRange(minLength, maxPasswordLength, password))
	}

	if !stringkit.InRange(email, minLength, maxEmailLength) {
		return nil, xerrors.Wrap(ErrInvalidEmail, stringkit.ErrorMsgInRange(minLength, maxEmailLength, email))
	}

	if !stringkit.InRange(phone, minLength, maxPhoneLength) {
		return nil, xerrors.Wrap(ErrInvalidPhone, stringkit.ErrorMsgInRange(minLength, maxPhoneLength, phone))
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

	prefectureID, err := uuid.Parse(prefectureIDStr)
	if err != nil {
		return nil, ErrInvalidPrefectureID
	}

	if deletedAt != nil && !(deletedAt.Before(time.Now())) {
		return nil, xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be in the past or equal to now")
	}

	return &Entity{
		id:             id,
		firstName:      firstName,
		lastName:       lastName,
		password:       password,
		email:          email,
		phone:          phone,
		prefectureID:   prefectureID,
		prefectureName: prefectureName,
		city:           city,
		street:         street,
		building:       building,
		postalCode:     postalCode,
		deletedAt:      deletedAt,
	}, nil
}

// ID は、ユーザーのIDを返します。
func (u *Entity) ID() uuid.UUID { return u.id }

// FirstName は、ユーザーの名前を返します。
func (u *Entity) FirstName() string { return u.firstName }

// LastName は、ユーザーの名字を返します。
func (u *Entity) LastName() string { return u.lastName }

// Password は、ユーザーのパスワードを返します。
func (u *Entity) Password() string { return u.password }

// Email は、ユーザーのメールアドレスを返します。
func (u *Entity) Email() string { return u.email }

// Phone は、ユーザーの電話番号を返します。
func (u *Entity) Phone() string { return u.phone }

// PrefectureID は、ユーザーの都道府県IDを返します。
func (u *Entity) PrefectureID() uuid.UUID { return u.prefectureID }

// PrefectureName は、ユーザーの都道府県名を返します。
func (u *Entity) PrefectureName() string { return u.prefectureName }

// City は、ユーザーの市区町村名を返します。
func (u *Entity) City() string { return u.city }

// Street は、ユーザーの番地を返します。
func (u *Entity) Street() string { return u.street }

// Building は、ユーザーの建物名を返します。
func (u *Entity) Building() *string { return u.building }

// PostalCode は、ユーザーの郵便番号を返します。
func (u *Entity) PostalCode() string { return u.postalCode }

// DeletedAt は、ユーザーの削除日時を返します。
func (u *Entity) DeletedAt() *time.Time { return u.deletedAt }

// FullName は、ユーザーのフルネームを返します。
func (u *Entity) FullName() string { return u.firstName + " " + u.lastName }
