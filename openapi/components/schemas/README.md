# 📘 OpenAPI 定義設計ポリシー

このドキュメントは、`swagger-cli` を使用する前提で構築された OpenAPI ファイル群の設計・分割方針を示します。

## 🧱 ディレクトリ構造ルール

| 種別 | ディレクトリ | 内容 |
|------------------|-----------------------------------|-----------------------------------|
| スキーマ定義     | `components/schemas/` | データ構造本体の定義（User, Errorなど） |
| リクエスト本体   | `components/request/` | リクエストの意味づけ（content, required等） |
| レスポンス定義   | `components/responses/` | HTTPステータスごとの返却の意味づけ |
| パラメータ定義   | `components/parameters/` | クエリ・パスパラメータの定義 |
| パスエンドポイント | `paths/` | `/v1/users`, `/health` などAPIルート定義 |

## 🔩 schemas（構造定義）

- 必ず `components/schemas/` に配置
- 1ファイルに1スキーマを定義（トップレベルは `type: object` のみ）
- `components:` ブロックは不要
- ファイル名とスキーマ名は PascalCase に統一

📄 例: `UserResponse.yaml`

```yaml
type: object
required:
  - name
  - email
properties:
  name:
    type: string
  email:
    type: string
    format: email
  phone:
    type: string
    nullable: true
```

### requestとresponsesについて

本来のopenapiでは、リクエストとレスポンスは`requestBodies` と `responses` で定義されます。

しかし、`swagger-cli`と`oapi-codegen`の制約により、`requestBodies` と `responses` としてはファイルを分けずに、`schemas` として定義してください。

### 🧩 PATCH対応ポリシー

- 共通構造体（例：`UserBaseInput`）を `schemas/` に定義
- 各リクエスト用途では、明確にラップして意味づけを行う

📄 `schemas/UserPatchRequest.yaml`

```yaml
allOf:
  - $ref: './UserBaseInput.yaml'
additionalProperties: false
```

📄 `requestBodies/UserPatchRequest.yaml`

```yaml
description: ユーザー情報の更新（PATCH）用
required: true
content:
  application/json:
    schema:
      $ref: '../../schemas/UserPatchRequest.yaml'
```

## 🎛 parameters（パス・クエリ）

- 1パラメータごとに1ファイル
- `name`, `in`, `schema`, `example` を記述
- ドメイン別 or 目的別でフォルダ分割可能（例：`user/`, `pagination/`）

📄 例: `parameters/UserIdParam.yaml`

```yaml
name: user_id
in: path
required: true
description: ユーザーのUUID
schema:
  type: string
  format: uuid
  example: "123e4567-e89b-12d3-a456-426614174000"
```

## 🚫 避けるべき構成

- `呼び出し側のschema` で schema 本体を直接定義（→ 再利用性・責務分離 NG）
- 全てのAPIで `UserBaseInput` を直接使う（→ 意味の曖昧化）
- `allOf` の乱用（→ ツール生成物が肥大化）

## 💡 補足と運用Tips

- **swagger-cli との互換性のため `#/components/...` 形式は使わない**
- すべての `$ref` は YAML 相対パスで統一
- `UserResponse`, `UserPostRequest` などは **処理の意味単位で命名**
- `schemas/` 配下のファイルが「リクエスト・レスポンス両用」になることは極力避ける

## ✅ チェックリスト

- [ ] schemas は 1ファイル1スキーマになっているか
- [ ] requestBodies/responses を schemas として定義しているか
- [ ] parameters は 1ファイル1パラメータになっているか
- [ ] PATCH 用には専用ラッパー（UserPatchRequestなど）を作っているか
- [ ] `$ref` はパス相対形式で統一されているか

この構成とポリシーに従うことで、swagger-cli での bundle / lint / CI 対応が容易になり、ツール互換性・保守性・チーム間の設計認知も向上します。
