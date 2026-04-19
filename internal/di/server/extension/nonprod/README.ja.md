# nonprod

[English](README.md) | 日本語

`nonprod` は、**非本番環境（development / staging / local）専用のサーバー拡張機能**を DI として提供するレイヤーです。

本番環境では無効化されるべき挙動（Echo のデバッグモードなど）を、環境設定に基づいて安全に適用します。

## モジュール一覧

|モジュール|種別|説明|
|---|---|---|
|`DebugModeModule()`|Configurator|非本番環境でのみ Echo デバッグモードを有効化|

## 注意点

- **必ず ApplicationConfig を参照し、本番環境では動かないようにすること**
- デバッグモードは「非本番専用」— 本番に漏れるとセキュリティリスクとなる
- ServeCfg は Echo インスタンスに直接副作用を与えるため、**domain / usecase 層に依存させないこと**
- 非本番向け設定を追加する場合は、この `nonprod` ディレクトリにモジュールを拡張することを推奨
