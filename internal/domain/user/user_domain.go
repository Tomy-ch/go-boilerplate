// Package user は、ユーザードメインを定義します。User エンティティ（論理削除・プロフィール更新）・FeedCursor 値オブジェクト・Repository インターフェースを提供します。
package user

import (
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Users は、User エンティティのスライス型です。
type Users []*User

// User は、ユーザーを表すドメインエンティティです。
type User struct {
	id           uuid.UUID
	firstName    string
	lastName     string
	email        Email
	phone        string
	prefectureID uuid.UUID
	city         string
	street       string
	building     *string
	postalCode   PostalCode
	createdAt    time.Time
	updatedAt    time.Time
	deletedAt    *time.Time
}

// New は、ユーザーエンティティの検証と生成を行います。
// updatedAt は createdAt 以降、deletedAt（非 nil 時）は createdAt および updatedAt 以降である必要があり、違反時はそれぞれ ErrInvalidUpdatedAt / ErrInvalidDeletedAt を返します。
func New(
	id uuid.UUID,
	firstName string,
	lastName string,
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

	emailVO, postalCodeVO, err := validateProfileFields(firstName, lastName, email, phone, prefectureID, city, street, building, postalCode)
	if err != nil {
		return nil, err
	}

	if updatedAt.Before(createdAt) {
		return nil, xerrors.Wrap(ErrInvalidUpdatedAt, "updatedAt must be after or equal to createdAt")
	}

	if deletedAt != nil {
		if err := validateDeletedAt(*deletedAt, createdAt, updatedAt); err != nil {
			return nil, err
		}
	}

	return &User{
		id:           id,
		firstName:    firstName,
		lastName:     lastName,
		email:        emailVO,
		phone:        phone,
		prefectureID: prefectureID,
		city:         city,
		street:       street,
		building:     ptr.Copy(building),
		postalCode:   postalCodeVO,
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

// Email は、ユーザーのメールアドレスを返します。
func (u *User) Email() string { return u.email.Value() }

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
func (u *User) PostalCode() string { return u.postalCode.Value() }

// DeletedAt は、ユーザーの削除日時を返します。
func (u *User) DeletedAt() *time.Time { return ptr.Copy(u.deletedAt) }

// CreatedAt は、ユーザーの作成日時を返します。
func (u *User) CreatedAt() time.Time { return u.createdAt }

// UpdatedAt は、ユーザーの更新日時を返します。
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// FullName は、ユーザーのフルネームを返します。
func (u *User) FullName() string { return u.firstName + " " + u.lastName }

// UpdateProfile は、プロフィール（氏名・連絡先・住所・都道府県ID）と更新日時を一括で置き換えます。
// 各フィールドは New と同じ不変条件で検証します。論理削除済みユーザーには ErrAlreadyDeleted を返します。
func (u *User) UpdateProfile(
	firstName, lastName, email, phone string,
	prefectureID uuid.UUID,
	city, street string,
	building *string,
	postalCode string,
	updatedAt time.Time,
) error {
	if err := u.ensureNotDeleted(); err != nil {
		return err
	}
	emailVO, postalCodeVO, err := validateProfileFields(firstName, lastName, email, phone, prefectureID, city, street, building, postalCode)
	if err != nil {
		return err
	}
	if err := u.ensureUpdatedAt(updatedAt); err != nil {
		return err
	}

	u.firstName = firstName
	u.lastName = lastName
	u.email = emailVO
	u.phone = phone
	u.prefectureID = prefectureID
	u.city = city
	u.street = street
	u.building = ptr.Copy(building)
	u.postalCode = postalCodeVO
	u.updatedAt = updatedAt
	return nil
}

// MarkAsDeleted は、ユーザーを論理削除します（deletedAt を設定）。論理削除は更新操作でもあるため updatedAt も deletedAt の値に更新されます。既に削除済みの場合は ErrAlreadyDeleted を返します。
func (u *User) MarkAsDeleted(deletedAt time.Time) error {
	if err := u.ensureNotDeleted(); err != nil {
		return err
	}
	if err := validateDeletedAt(deletedAt, u.createdAt, u.updatedAt); err != nil {
		return err
	}

	u.deletedAt = &deletedAt
	// 論理削除も更新操作のため、updatedAt を削除時刻（usecase が clock から取得した現在時刻）に追従させる。
	u.updatedAt = deletedAt
	return nil
}

// ensureNotDeleted は、論理削除済みユーザーへの変更操作を拒否します（削除済みは不変）。
func (u *User) ensureNotDeleted() error {
	if u.deletedAt != nil {
		return ErrAlreadyDeleted
	}
	return nil
}

// ensureUpdatedAt は、更新日時が createdAt 以降かつ現在の updatedAt 以降（単調非減少）であることを検証します。
func (u *User) ensureUpdatedAt(updatedAt time.Time) error {
	if updatedAt.Before(u.createdAt) {
		return xerrors.Wrap(ErrInvalidUpdatedAt, "updatedAt must be after or equal to createdAt")
	}
	if updatedAt.Before(u.updatedAt) {
		return xerrors.Wrap(ErrInvalidUpdatedAt, "updatedAt must be after or equal to current updatedAt")
	}
	return nil
}

// validateDeletedAt は、削除日時が createdAt / updatedAt 以降であることを検証します。
func validateDeletedAt(deletedAt, createdAt, updatedAt time.Time) error {
	if deletedAt.Before(createdAt) {
		return xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to createdAt")
	}
	if deletedAt.Before(updatedAt) {
		return xerrors.Wrap(ErrInvalidDeletedAt, "deletedAt must be after or equal to updatedAt")
	}
	return nil
}

// validateProfileFields は、プロフィール系フィールドの不変条件をすべて検証し、失敗を収集します。
// 失敗があった場合は各フィールドの検証エラーを結合し、不正フィールドの識別子を
// apperror.Meta の Details として付与して返します（理由文はエラーメッセージ側にのみ残ります）。
// email / postalCode は値オブジェクトの factory で検証し、成功時は構築済みの VO を返します。
func validateProfileFields(
	firstName, lastName, email, phone string,
	prefectureID uuid.UUID,
	city, street string,
	building *string,
	postalCode string,
) (Email, PostalCode, error) {
	var errs []error
	var fields []string

	if ok, msg := stringkit.ValidateInRange(firstName, minLength, maxFirstNameLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidFirstName, msg))
		fields = append(fields, FieldFirstName)
	}
	if ok, msg := stringkit.ValidateInRange(lastName, minLength, maxLastNameLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidLastName, msg))
		fields = append(fields, FieldLastName)
	}
	emailVO, emailErr := NewEmail(email)
	if emailErr != nil {
		errs = append(errs, emailErr)
		fields = append(fields, FieldEmail)
	}
	if ok, msg := stringkit.ValidateInRange(phone, minLength, maxPhoneLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidPhone, msg))
		fields = append(fields, FieldPhone)
	}
	if prefectureID.IsNil() {
		errs = append(errs, xerrors.Wrap(ErrInvalidPrefectureID, "prefectureID is required"))
		fields = append(fields, FieldPrefectureID)
	}
	if ok, msg := stringkit.ValidateInRange(city, minLength, maxCityLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidCity, msg))
		fields = append(fields, FieldCity)
	}
	if ok, msg := stringkit.ValidateInRange(street, minLength, maxStreetLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidStreet, msg))
		fields = append(fields, FieldStreet)
	}
	if building != nil {
		if ok, msg := stringkit.ValidateInRange(*building, minLength, maxBuildingLength); !ok {
			errs = append(errs, xerrors.Wrap(ErrInvalidBuilding, msg))
			fields = append(fields, FieldBuilding)
		}
	}
	postalCodeVO, postalCodeErr := NewPostalCode(postalCode)
	if postalCodeErr != nil {
		errs = append(errs, postalCodeErr)
		fields = append(fields, FieldPostalCode)
	}

	if len(errs) > 0 {
		return Email{}, PostalCode{}, apperror.WithDetails(xerrors.Join(errs...), fields...)
	}
	return emailVO, postalCodeVO, nil
}
