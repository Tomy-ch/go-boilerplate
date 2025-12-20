// Package conv は、database/sqlのnullable型をポインタと変換するためのユーティリティを提供します。
package conv

import (
	"database/sql"
	"time"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/pkg/uuid"
	"boilerplate-go/pkg/xerrors"

	googleUUID "github.com/google/uuid"
)

// UUIDPtrFromNull は、googleUUID.NullUUIDをポインタに変換します。
func UUIDPtrFromNull(nu googleUUID.NullUUID) (*uuid.UUID, error) {
	if !nu.Valid {
		return nil, nil
	}

	parsedUUID, err := uuid.Parse(nu.UUID.String())
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInternal, err.Error())
	}
	return &parsedUUID, nil
}

// NullUUIDFromPtr は、ポインタをgoogleUUID.NullUUIDに変換します。
func NullUUIDFromPtr(u *uuid.UUID) googleUUID.NullUUID {
	if u == nil {
		return googleUUID.NullUUID{Valid: false}
	}
	return NewNullUUID(*u)
}

// NewNullUUID は、引数をgoogleUUID.NullUUIDに変換します。
func NewNullUUID(u uuid.UUID) googleUUID.NullUUID {
	return googleUUID.NullUUID{UUID: googleUUID.UUID(u.Bytes()), Valid: true}
}

// StringPtrFromNull は、sql.NullStringをポインタに変換します。
func StringPtrFromNull(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// NullStringFromPtr は、ポインタをsql.NullStringに変換します。
func NullStringFromPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return NewNullString(*s)
}

// NewNullString は、引数をsql.NullStringに変換します。
func NewNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

// Int16PtrFromNull は、sql.NullInt16をポインタに変換します。
func Int16PtrFromNull(n sql.NullInt16) *int16 {
	if n.Valid {
		return &n.Int16
	}
	return nil
}

// NullInt16FromPtr は、ポインタをsql.NullInt16に変換します。
func NullInt16FromPtr(p *int16) sql.NullInt16 {
	if p == nil {
		return sql.NullInt16{Valid: false}
	}
	return NewNullInt16(*p)
}

// NewNullInt16 は、引数をsql.NullInt16に変換します。
func NewNullInt16(n int16) sql.NullInt16 {
	return sql.NullInt16{Int16: n, Valid: true}
}

// Int64PtrFromNull は、sql.NullInt64をポインタに変換します。
func Int64PtrFromNull(n sql.NullInt64) *int64 {
	if n.Valid {
		return &n.Int64
	}
	return nil
}

// NullInt64FromPtr は、ポインタをsql.NullInt64に変換します。
func NullInt64FromPtr(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{Valid: false}
	}
	return NewNullInt64(*p)
}

// NewNullInt64 は、引数をsql.NullInt64に変換します。
func NewNullInt64(n int64) sql.NullInt64 {
	return sql.NullInt64{Int64: n, Valid: true}
}

// BoolPtrFromNull は、sql.NullBoolをポインタに変換します。
func BoolPtrFromNull(n sql.NullBool) *bool {
	if n.Valid {
		return &n.Bool
	}
	return nil
}

// NullBoolFromPtr は、ポインタをsql.NullBoolに変換します。
func NullBoolFromPtr(p *bool) sql.NullBool {
	if p == nil {
		return sql.NullBool{Valid: false}
	}
	return NewNullBool(*p)
}

// NewNullBool は、引数をsql.NullBoolに変換します。
func NewNullBool(b bool) sql.NullBool {
	return sql.NullBool{Bool: b, Valid: true}
}

// Float64PtrFromNull は、sql.NullFloat64をポインタに変換します。
func Float64PtrFromNull(n sql.NullFloat64) *float64 {
	if n.Valid {
		return &n.Float64
	}
	return nil
}

// NullFloat64FromPtr は、ポインタをsql.NullFloat64に変換します。
func NullFloat64FromPtr(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{Valid: false}
	}
	return NewNullFloat64(*p)
}

// NewNullFloat64 は、引数をsql.NullFloat64に変換します。
func NewNullFloat64(n float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: n, Valid: true}
}

// TimePtrFromNull は、sql.NullTimeをポインタに変換します。
func TimePtrFromNull(n sql.NullTime) *time.Time {
	if n.Valid {
		return &n.Time
	}
	return nil
}

// NullTimeFromPtr は、ポインタをsql.NullTimeに変換します。
func NullTimeFromPtr(p *time.Time) sql.NullTime {
	if p == nil {
		return sql.NullTime{Valid: false}
	}
	return NewNullTime(*p)
}

// NewNullTime は、引数をsql.NullTimeに変換します。
func NewNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}
