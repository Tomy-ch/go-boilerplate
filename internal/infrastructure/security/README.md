# security パッケージ

概要: このパッケージは、暗号化をはじめとするセキュリティ関連の機能を提供します。

`BcryptHasher` を実装し、パスワードのハッシュ化と比較を行います。

## 提供する主な機能

- `NewBcryptHasher()` 関数: パスワードをハッシュ化するための `BcryptHasher` を生成します。
- `Hash(password string) (string, error)` メソッド: パスワードをハッシュ化します。
- `Compare(hashedPassword, password string) error` メソッド: ハッシュ化されたパスワードと平文のパスワードを比較します。

## 使い方

環境ごとやサービスごとに適切な `BcryptHasher` を実装し、アプリケーションのセキュリティを確保します。

システムへの取り込みは、`internal/di/module/infrastructure.go` の `security` に実装を追加してください。
