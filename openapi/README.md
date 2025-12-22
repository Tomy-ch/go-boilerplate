# OpenAPI 定義ガイドライン

このディレクトリには、本プロジェクトで使用するOpenAPI定義が格納されています。  
各ファイルは **ドメイン単位かつバージョンごと** に分かれており、責務の分離とバージョン管理を容易にします。

## ディレクトリ構成

```text
openapi/
├── v1/
│   ├── user/
│   │   └── user.yaml
│   └── health/
│       └── health.yaml
...
```

## APIバージョン管理ポリシー

- `/v1/`や`/v2/`のように**パスにバージョンを含めて管理**します
- 本番公開されたAPIは、**後方互換性を維持する必要があります**
- 本番公開された後に**破壊的変更**を行う場合は、**新しいメジャーバージョン(`/v2/...`)を作成してください**：
  - エンドポイントの削除
  - レスポンスの構造や型の変更
  - リクエスト項目の必須化 など

## セキュリティ設計ポリシー

本プロジェクトでは、OpenAPI 定義におけるセキュリティ設計において以下の方針を採用しています：

### 認証・認可に関するルール

- **JWTトークンをベースとした認証**を導入し、操作可能なAPI（POST / PUT / DELETE / PATCH）はすべてJWT必須とします
- プライベートな情報（ユーザー詳細など）を取得する `GET` 系APIにおいても、**JWT認証を必須**とします
- JWTの `sub` claim を元に、リソースオーナーかどうかの**認可判定**を実装側で行います

### ID設計における注意点

- 外部公開APIで `user_id` などのUUIDを使用しますが、これは**認証済みユーザーのみがアクセス可能な前提**で設計されています
- IDOR（Insecure Direct Object Reference）を防ぐため、**必ずリクエストユーザーの照合を行うミドルウェアまたはユースケース層での検証を実装**してください
- 管理用途で内部的なUUIDやメタ情報を扱う場合は、必ず `/admin` 配下などでスコープを分離し、**JWTトークンのロールやスコープ検証**を導入してください

### UUID公開とセキュリティ設計に関する詳細

UUIDを公開しても問題ない理由と、それを安全に運用するための設計指針は以下にまとめています：

👉 [secure-uuid.md](./secure-uuid.md)

## 設計ルール（Google API Design Guide に準拠）

- [Google API Design Guide](https://cloud.google.com/apis/design) をベースに設計します。
  - リソース名は複数形で表現します（例: `/users`, `/projects`）
  - 特定のリソースに対する操作は、リソース名の後に`/`を付けて表現します。（例: `GET /users/{id}`）
    - 特定のリソースでidなど以外の識別子を使用する場合は、`by-{identifier}` の形式で表現します。（例: `GET /users/by-email/{email}`）
  - CRUD操作はHTTPメソッドで表現します。（`GET`, `POST`, `PATCH`, `DELETE`）
  - 非CRUD操作は `:action` サフィックスで表現します。（例: `POST /users/{id}:deactivate`）

## ファイル命名と構成ルール

- ドメインごとに YAML ファイルを分ける（例: `user.yaml`, `health.yaml`）
- 複数ドメインで共通利用する構造体は、`components/`配下に切り出して管理します。
- `operationId` は Go の関数名に影響するため、一貫性のある命名を心がけます。
  - `{動詞}{対象となるリソース名}{条件・付加情報}`の`lowerCamelCase`形式で命名します。
  - 例: `createUser`, `getUserByEmail`

## oapi-codegen によるコード生成

以下のような Make ターゲットを使用して Go のコードを生成します：

```sh
make go-gen # ハンドラの生成
```

## セキュリティ関連ドキュメント一覧

| ドキュメント名 | 内容 |
| ----------- | ---- |
| [secure-uuid.md](./secure-uuid.md) | UUIDを外部公開しても安全に設計・運用するための指針 |

今後この一覧に、`auth.md`, `secure-headers.md` などを追加することでセキュリティガイドラインの拡張を想定しています。

## その他注意事項

- スキーマは極力 $ref による再利用を行い、inline schema は最小限にしてください
- Swagger 独自拡張 (x-...) の使用は必要最低限にとどめてください

## 参考リンク

- <https://github.com/oapi-codegen/oapi-codegen>
- <https://swagger.io/specification/>
- <https://cloud.google.com/apis/design>
