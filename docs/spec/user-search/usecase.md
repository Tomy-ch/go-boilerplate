# User Search — Usecase Spec

> 既存実装（`internal/usecase/user/search`）を spec 化したもの。手書き実装から逆生成した現状仕様。
> user 集約の query 側（CQRS の Query）。集約を直接生成せず QueryService 経由で読み取り DTO を返すため domain.md は持たない（参照する集約は `user`、spec は `docs/spec/user/domain.md`）。

## Overview

ユーザー検索ユースケースは、キーワードとアクティブ状態でユーザーを検索する read-only な Query サービス。`UserSearchQueryService`（QueryService boundary）を介して検索結果 DTO を直接返す。CQRS の軽量分離方針に従い、ドメインエンティティへの変換は行わず QueryService の結果（`UserSearchResults`）をそのまま返す。トランザクションは不要。

キーワードは `tools/search.ParseSearchTokens` でトークン分割してから QueryService に渡す（アプリケーションポリシー）。

## Interface

```yaml
package: internal/usecase/user/search
name: Usecase
methods:
  - name: ListUsersByKeyword
    signature: ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Paging) (query.UserSearchResults, error)
  - name: CountUsersByKeyword
    signature: CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error)
```

## DTOs

```yaml
- name: SearchParams
  description: ユーザー検索の入力パラメータ。
  fields:
    - name: Keyword
      type: "*string"
    - name: Active
      type: "*bool"
# 以下は query サブパッケージ（internal/usecase/user/search/query）で定義される QueryService の入出力型。
- name: query.UserSearchFilter
  description: QueryService に渡す検索条件。usecase 側で SearchParams + トークン分割から構築。
  fields:
    - name: Active
      type: "*bool"
    - name: Keywords
      type: "[]string"
- name: query.UserSearchResult
  description: 検索結果 1 件分の DTO（QueryService が返す）。
  fields:
    - name: FirstName
      type: string
    - name: LastName
      type: string
    - name: Email
      type: string
    - name: Phone
      type: string
    - name: PostalCode
      type: string
    - name: PrefectureName
      type: string
    - name: City
      type: string
    - name: Street
      type: string
    - name: Building
      type: "*string"
    - name: RegisteredAt
      type: time.Time
    - name: DeletedAt
      type: "*time.Time"
```

## Dependencies

```yaml
- tracer              # observability.TracerFactory -> LayerTracer
- user_query_service  # user/search/query.UserSearchQueryService（キーワード検索の QueryService boundary）
```

## Workflow

### ListUsersByKeyword

```yaml
tx_required: false
steps:
  - tools/search.ParseSearchTokens で filter.Keyword をトークン分割
  - Active + Keywords から query.UserSearchFilter を構築
  - user_query_service.FindByFilter でページング付き（limit/offset）に検索結果を取得して返す
calls:
  - user_query_service.FindByFilter
errors:
  - user_query_service.FindByFilter のエラーをそのまま伝播
```

### CountUsersByKeyword

```yaml
tx_required: false
steps:
  - tools/search.ParseSearchTokens で filter.Keyword をトークン分割
  - Active + Keywords から query.UserSearchFilter を構築
  - user_query_service.CountByFilter で検索条件に一致する総件数を取得して返す
calls:
  - user_query_service.CountByFilter
errors:
  - user_query_service.CountByFilter のエラーをそのまま伝播
```
