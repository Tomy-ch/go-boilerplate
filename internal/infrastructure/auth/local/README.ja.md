# local（ローカル向け簡易認証）

[English](README.md) | 日本語

ローカル開発環境や CI / テスト環境で利用する **簡易的な `Authenticator` 実装**です。外部認証プロバイダのエミュレーション用であり、**本番運用での使用は想定していません**。

## 役割

- 開発時の手早い動作確認のために、`Authenticator` インターフェース経由で認証情報を提供
- 本番認証サービスの代替として、簡易的に認証済みの主体（`Authn`）を返す

## 公開 API

|関数 / 型|説明|
|---|---|
|`New()`|`LocalMockAuthenticator` を生成（`authbd.Authenticator` を返す）|
|`Authenticate(ctx, cred)`|トークンから Subject を抽出し `Authn` を返す|
|`ErrLocalMockAuthenticatorInvalidToken`|無効・空トークン時のエラー|

## トークン形式

```text
Authorization: Bearer debug:user123
```

- プレフィックス `debug:` を除去して Subject を抽出
- Subject: `user123`、Provider: `mock`
- 署名検証は行わない

## 本番への置き換え

`internal/di/module/core/auth.go` の `provideAuthenticator` を編集し、環境（local / stg / prd）に応じて DI 登録する実装を切り替えてください。

## 注意点

- 署名検証・トークン有効期限チェック・リプレイ防止を行わないため、セキュリティは担保されない
- 本番環境では使用しないこと
- コード内のセキュリティ関連設定やハードコード値を運用環境に持ち込まないこと
