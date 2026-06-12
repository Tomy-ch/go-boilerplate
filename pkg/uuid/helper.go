package uuid

import (
	guuid "github.com/google/uuid"
)

func fromGoogle(g guuid.UUID) UUID {
	return UUID{b: g}
}

func toGoogle(u UUID) guuid.UUID {
	return guuid.UUID(u.b)
}
