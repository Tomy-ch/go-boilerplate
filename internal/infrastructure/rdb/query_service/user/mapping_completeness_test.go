package user

import "testing"

func Test_fetchSearchAll(t *testing.T) {
	t.Parallel()
	t.Skip("Test_service_FindByFilter（Active=nil 経路）の実 DB テストでカバー")
}

func Test_fetchSearchActive(t *testing.T) {
	t.Parallel()
	t.Skip("Test_service_FindByFilter（Active=true 経路）の実 DB テストでカバー")
}

func Test_fetchSearchDeleted(t *testing.T) {
	t.Parallel()
	t.Skip("Test_service_FindByFilter（Active=false 経路）の実 DB テストでカバー")
}
