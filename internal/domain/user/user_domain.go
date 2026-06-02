// Package user は、ユーザー関連のドメインを提供します。
package user

import (
	"time"

	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
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

	if err := validateProfileFields(firstName, lastName, email, phone, prefectureID, city, street, building, postalCode); err != nil {
		return nil, err
	}

	if err := validatePasswordHash(passwordHash); err != nil {
		return nil, err
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

// UpdateProfile は、プロフィール（氏名・連絡先・住所・都道府県ID）と更新日時を一括で置き換えます。
// パスワードは変更しません。各フィールドは New と同じ不変条件で検証します。
func (u *User) UpdateProfile(
	firstName, lastName, email, phone string,
	prefectureID uuid.UUID,
	postalCode, city, street string,
	building *string,
	updatedAt time.Time,
) error {
	if err := validateProfileFields(firstName, lastName, email, phone, prefectureID, city, street, building, postalCode); err != nil {
		return err
	}
	if err := u.ensureUpdatedAt(updatedAt); err != nil {
		return err
	}

	u.firstName = firstName
	u.lastName = lastName
	u.email = email
	u.phone = phone
	u.prefectureID = prefectureID
	u.city = city
	u.street = street
	u.building = ptr.Copy(building)
	u.postalCode = postalCode
	u.updatedAt = updatedAt
	return nil
}

// ChangePassword は、パスワードハッシュと更新日時を置き換えます。
func (u *User) ChangePassword(passwordHash string, updatedAt time.Time) error {
	if err := validatePasswordHash(passwordHash); err != nil {
		return err
	}
	if err := u.ensureUpdatedAt(updatedAt); err != nil {
		return err
	}

	u.passwordHash = passwordHash
	u.updatedAt = updatedAt
	return nil
}

// MarkAsDeleted は、ユーザーを論理削除します（deletedAt を設定）。
// 既に削除済みの場合は ErrAlreadyDeleted を返します。
func (u *User) MarkAsDeleted(deletedAt time.Time) error {
	if u.deletedAt != nil {
		return xerrors.Wrap(ErrAlreadyDeleted, "user is already deleted")
	}
	if deletedAt.Before(u.createdAt) {
		return xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to createdAt")
	}
	if deletedAt.Before(u.updatedAt) {
		return xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to updatedAt")
	}

	u.deletedAt = ptr.Copy(&deletedAt)
	return nil
}

// ensureUpdatedAt は、更新日時が不変条件（createdAt 以降、かつ deletedAt 以前）を満たすか検証します。
func (u *User) ensureUpdatedAt(updatedAt time.Time) error {
	if updatedAt.Before(u.createdAt) {
		return xerrors.Wrap(ErrInvalidUpdatedAt, "updatedAt must be after or equal to createdAt")
	}
	if u.deletedAt != nil && u.deletedAt.Before(updatedAt) {
		return xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to updatedAt")
	}
	return nil
}

// validateProfileFields は、プロフィール系フィールドの不変条件を検証します。
func validateProfileFields(
	firstName, lastName, email, phone string,
	prefectureID uuid.UUID,
	city, street string,
	building *string,
	postalCode string,
) error {
	if !stringkit.InRange(firstName, minLength, maxFirstNameLength) {
		return xerrors.Wrap(ErrInvalidFirstName, stringkit.ErrorMsgInRange(minLength, maxFirstNameLength, firstName))
	}
	if !stringkit.InRange(lastName, minLength, maxLastNameLength) {
		return xerrors.Wrap(ErrInvalidLastName, stringkit.ErrorMsgInRange(minLength, maxLastNameLength, lastName))
	}
	if !stringkit.InRange(email, minLength, maxEmailLength) {
		return xerrors.Wrap(ErrInvalidEmail, stringkit.ErrorMsgInRange(minLength, maxEmailLength, email))
	}
	if !stringkit.InRange(phone, minLength, maxPhoneLength) {
		return xerrors.Wrap(ErrInvalidPhone, stringkit.ErrorMsgInRange(minLength, maxPhoneLength, phone))
	}
	if prefectureID.IsNil() {
		return xerrors.Wrap(ErrInvalidPrefectureID, "prefectureID is required")
	}
	if !stringkit.InRange(city, minLength, maxCityLength) {
		return xerrors.Wrap(ErrInvalidCity, stringkit.ErrorMsgInRange(minLength, maxCityLength, city))
	}
	if !stringkit.InRange(street, minLength, maxStreetLength) {
		return xerrors.Wrap(ErrInvalidStreet, stringkit.ErrorMsgInRange(minLength, maxStreetLength, street))
	}
	if building != nil && !stringkit.InRange(*building, minLength, maxBuildingLength) {
		return xerrors.Wrap(ErrInvalidBuilding, stringkit.ErrorMsgInRange(minLength, maxBuildingLength, *building))
	}
	if !stringkit.InRange(postalCode, minLength, maxPostalCodeLength) {
		return xerrors.Wrap(ErrInvalidPostalCode, stringkit.ErrorMsgInRange(minLength, maxPostalCodeLength, postalCode))
	}
	return nil
}

// validatePasswordHash は、パスワードハッシュの不変条件を検証します。
func validatePasswordHash(passwordHash string) error {
	if !stringkit.InRange(passwordHash, minLength, maxPasswordLength) {
		return xerrors.Wrap(ErrInvalidPasswordHash, stringkit.ErrorMsgInRange(minLength, maxPasswordLength, passwordHash))
	}
	return nil
}
