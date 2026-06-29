package user

const (
	minLength             = 1
	maxFirstNameLength    = 100
	maxLastNameLength     = 100
	maxPasswordHashLength = 255
	maxEmailLength        = 100
	maxPhoneLength        = 20
	maxCityLength         = 100
	maxStreetLength       = 255
	maxBuildingLength     = 255
	maxPostalCodeLength   = 8

	// MaxRawPasswordLength は、平文パスワードの最大文字数です。
	MaxRawPasswordLength = 64
	// MinRawPasswordLength は、平文パスワードの最小文字数です。
	MinRawPasswordLength = 8
)
