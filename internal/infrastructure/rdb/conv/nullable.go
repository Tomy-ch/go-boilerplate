// Package conv は、database/sql のnullable型をポインタと変換するためのユーティリティを提供します。
package conv

import (
	"database/sql"
	"time"
)

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
	return sql.NullString{String: *s, Valid: true}
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
	return sql.NullInt16{Int16: *p, Valid: true}
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
	return sql.NullInt64{Int64: *p, Valid: true}
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
	return sql.NullBool{Bool: *p, Valid: true}
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
	return sql.NullFloat64{Float64: *p, Valid: true}
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
	return sql.NullTime{Time: *p, Valid: true}
}
