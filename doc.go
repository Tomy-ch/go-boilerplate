// Package root は go-boilerplate（軽量 Onion Architecture による RESTful API のひな形）の
// モジュールルートであり、godoc のランディングを兼ねます。
//
// アーキテクチャは controller → usecase → domain の依存方向で構成し、infrastructure が
// domain のインターフェースを実装します。OpenAPI ファースト・sqlc による型安全な DB アクセスを
// 採用し、ビジネスロジックは internal/ 配下に閉じます。各レイヤの詳細は README.md および
// docs/architecture.md を参照してください。
//
// ルートパッケージ自体は、バイナリへ焼き込む設定とマイグレーションを embed.FS として公開する
// 役割だけを持ちます（FS を参照）。go:embed が親ディレクトリを遡れない制約上、埋め込み対象の
// env/.env と database/migrations を参照できるモジュールルートに配置しています。
package root
