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

// Profile は、ユーザーのプロフィール属性一式です。New と UpdateProfile が同じ集合を受け取ります。
// Building は未設定を nil で表し、Email と PostalCode は値オブジェクトへの変換時に検証します。
type Profile struct {
	FirstName    string
	LastName     string
	Email        string
	Phone        string
	PrefectureID uuid.UUID
	City         string
	Street       string
	Building     *string
	PostalCode   string
}

// Attributes は、ユーザーの属性一式です。Profile に監査時刻を加えた、生成時に必要な集合を表します。
// DeletedAt は未削除を nil で表します。
type Attributes struct {
	Profile

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// New は、ユーザーエンティティの検証と生成を行います。
// UpdatedAt は CreatedAt 以降、DeletedAt（非 nil 時）は CreatedAt および UpdatedAt 以降である必要があり、違反時はそれぞれ ErrInvalidUpdatedAt / ErrInvalidDeletedAt を返します。
func New(id uuid.UUID, attrs Attributes) (*User, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}

	emailVO, postalCodeVO, err := validateProfileFields(attrs.Profile)
	if err != nil {
		return nil, err
	}

	if attrs.UpdatedAt.Before(attrs.CreatedAt) {
		return nil, xerrors.Wrap(ErrInvalidUpdatedAt, "updatedAt must be after or equal to createdAt")
	}

	if attrs.DeletedAt != nil {
		if err := validateDeletedAt(*attrs.DeletedAt, attrs.CreatedAt, attrs.UpdatedAt); err != nil {
			return nil, err
		}
	}

	return &User{
		id:           id,
		firstName:    attrs.FirstName,
		lastName:     attrs.LastName,
		email:        emailVO,
		phone:        attrs.Phone,
		prefectureID: attrs.PrefectureID,
		city:         attrs.City,
		street:       attrs.Street,
		building:     ptr.Copy(attrs.Building),
		postalCode:   postalCodeVO,
		createdAt:    attrs.CreatedAt,
		updatedAt:    attrs.UpdatedAt,
		deletedAt:    ptr.Copy(attrs.DeletedAt),
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

// IsActive は、ユーザーが在籍しているか（退会していないか）を返します。
// 「在籍している」という業務上の語が指す条件はこの述語が定義であり、永続化層の絞り込み条件が
// 定義になることはありません。
func (u *User) IsActive() bool { return u.deletedAt == nil }

// UpdateProfile は、プロフィール（氏名・連絡先・住所・都道府県ID）と更新日時を一括で置き換えます。
// 各フィールドは New と同じ不変条件で検証します。論理削除済みユーザーには ErrAlreadyDeleted を返します。
func (u *User) UpdateProfile(profile Profile, updatedAt time.Time) error {
	if err := u.ensureNotDeleted(); err != nil {
		return err
	}
	emailVO, postalCodeVO, err := validateProfileFields(profile)
	if err != nil {
		return err
	}
	if err := u.ensureUpdatedAt(updatedAt); err != nil {
		return err
	}

	u.firstName = profile.FirstName
	u.lastName = profile.LastName
	u.email = emailVO
	u.phone = profile.Phone
	u.prefectureID = profile.PrefectureID
	u.city = profile.City
	u.street = profile.Street
	u.building = ptr.Copy(profile.Building)
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
	u.updatedAt = deletedAt
	return nil
}

// ensureNotDeleted は、論理削除済みユーザーへの変更操作を拒否します（削除済みは不変）。
func (u *User) ensureNotDeleted() error {
	if !u.IsActive() {
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
func validateProfileFields(profile Profile) (Email, PostalCode, error) {
	var errs []error
	var fields []string

	if ok, msg := stringkit.ValidateInRange(profile.FirstName, minLength, maxFirstNameLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidFirstName, msg))
		fields = append(fields, FieldFirstName)
	}
	if ok, msg := stringkit.ValidateInRange(profile.LastName, minLength, maxLastNameLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidLastName, msg))
		fields = append(fields, FieldLastName)
	}
	emailVO, emailErr := NewEmail(profile.Email)
	if emailErr != nil {
		errs = append(errs, emailErr)
		fields = append(fields, FieldEmail)
	}
	if ok, msg := stringkit.ValidateInRange(profile.Phone, minLength, maxPhoneLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidPhone, msg))
		fields = append(fields, FieldPhone)
	}
	if profile.PrefectureID.IsNil() {
		errs = append(errs, xerrors.Wrap(ErrInvalidPrefectureID, "prefectureID is required"))
		fields = append(fields, FieldPrefectureID)
	}
	if ok, msg := stringkit.ValidateInRange(profile.City, minLength, maxCityLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidCity, msg))
		fields = append(fields, FieldCity)
	}
	if ok, msg := stringkit.ValidateInRange(profile.Street, minLength, maxStreetLength); !ok {
		errs = append(errs, xerrors.Wrap(ErrInvalidStreet, msg))
		fields = append(fields, FieldStreet)
	}
	if profile.Building != nil {
		if ok, msg := stringkit.ValidateInRange(*profile.Building, minLength, maxBuildingLength); !ok {
			errs = append(errs, xerrors.Wrap(ErrInvalidBuilding, msg))
			fields = append(fields, FieldBuilding)
		}
	}
	postalCodeVO, postalCodeErr := NewPostalCode(profile.PostalCode)
	if postalCodeErr != nil {
		errs = append(errs, postalCodeErr)
		fields = append(fields, FieldPostalCode)
	}

	if len(errs) > 0 {
		return Email{}, PostalCode{}, apperror.WithDetails(xerrors.Join(errs...), fields...)
	}
	return emailVO, postalCodeVO, nil
}
