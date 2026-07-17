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

	minRoleNameLength = 1
	maxRoleNameLength = 100

	// MaxRawPasswordLength は、平文パスワードの最大文字数です。
	MaxRawPasswordLength = 64
	// MinRawPasswordLength は、平文パスワードの最小文字数です。
	MinRawPasswordLength = 8
)

// プロフィール系フィールドの識別子。検証失敗時に apperror.Meta の Details として
// レスポンスへ公開されるため、API リクエストのプロパティ名と一致させています。
const (
	// FieldFirstName は、名フィールドの識別子です。
	FieldFirstName = "firstName"
	// FieldLastName は、姓フィールドの識別子です。
	FieldLastName = "lastName"
	// FieldEmail は、メールアドレスフィールドの識別子です。
	FieldEmail = "email"
	// FieldPhone は、電話番号フィールドの識別子です。
	FieldPhone = "phone"
	// FieldPrefectureID は、都道府県フィールドの識別子です。ドメインの検証対象は
	// prefectureID(uuid)ですが、公開する識別子はクライアントが送信する API プロパティ名
	// "prefecture"(都道府県名)に合わせています。
	FieldPrefectureID = "prefecture"
	// FieldCity は、市区町村フィールドの識別子です。
	FieldCity = "city"
	// FieldStreet は、番地フィールドの識別子です。
	FieldStreet = "street"
	// FieldBuilding は、建物名フィールドの識別子です。
	FieldBuilding = "building"
	// FieldPostalCode は、郵便番号フィールドの識別子です。
	FieldPostalCode = "postalCode"
)
