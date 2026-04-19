# security

[English](README.md) | 日本語

パスワードのハッシュ化・比較のための `Encrypter` インターフェースを提供します。

```go
type Encrypter interface {
    Hash(password string) (string, error)
    Compare(hash, password string) (bool, error)
}
```

## 設計意図

- 暗号アルゴリズムの詳細（bcrypt, argon2 等）を Usecase から隠蔽
- ビジネスロジックに影響を与えずにアルゴリズムを差し替え可能に
- テスト時にモック差し替えが可能

## 実装

`internal/infrastructure/security/` に bcrypt ベースの実装が配置されています。

## 注意点

- `Compare` はパスワード不一致時に `(false, nil)` を返す — 不一致はエラーではない
- bcrypt コストは `config.SecurityConfig.BcryptCost()` で設定
