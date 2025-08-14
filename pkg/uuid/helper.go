package uuid

import (
	guuid "github.com/google/uuid"
)

func fromGoogle(g guuid.UUID) UUID {
	var u UUID
	copy(u.b[:], g[:])
	return u
}

func toGoogle(u UUID) guuid.UUID {
	var g guuid.UUID
	copy(g[:], u.b[:])
	return g
}
