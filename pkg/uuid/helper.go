package uuid

import (
	guuid "github.com/google/uuid"
)

// fromGoogle は、GoogleのUUIDを独自のUUID型に変換します。
func fromGoogle(g guuid.UUID) UUID {
	var u UUID
	copy(u.b[:], g[:])
	return u
}

// toGoogle は、独自のUUID型をGoogleのUUIDに変換します。
func toGoogle(u UUID) guuid.UUID {
	var g guuid.UUID
	copy(g[:], u.b[:])
	return g
}
