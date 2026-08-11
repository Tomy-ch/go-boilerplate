# User Search — Usecase Spec

> 既存実装（`internal/usecase/user/search`）を spec 化したもの。手書き実装から逆生成した現状仕様。
> user 集約に対する読み取り専用ユースケース。固有の集約を持たないため domain.md は独立させず、
> 参照する集約の spec（`docs/spec/user/domain.md`）を見る。

## Overview

ユーザー検索ユースケースは、キーワードとアクティブ状態でユーザーを検索する read-only なユースケース。`user.Repository`（domain Repository）の `SearchByKeyword` / `CountByKeyword` を直接呼び、取得した `User` エンティティを検索結果 DTO（`UserSearchResult`）へ写像して返す。都道府県名は `prefecture.Repository.FindByIDs` でまとめて解決してから埋める（N+1 回避）。トランザクションは不要。

読み取り対象は正本である `users` テーブルそのものであり、派生射影ではないため QueryService ではなく Repository に委譲する（判定基準は `docs/design/data-access-pattern.md`）。

キーワードは `tools/search.ParseSearchTokens` でトークン分割してから `user.Repository` へ渡す（アプリケーションポリシー）。

## Interface

```yaml
package: internal/usecase/user/search
name: Usecase
methods:
  - name: ListUsersByKeyword
    signature: ListUsersByKeyword(ctx context.Context, filter *SearchParams, page *paging.Page) (UserSearchResults, error)
  - name: CountUsersByKeyword
    signature: CountUsersByKeyword(ctx context.Context, filter *SearchParams) (int64, error)
  - name: ListUsersByKeywordWithTotal
    signature: ListUsersByKeywordWithTotal(ctx context.Context, filter *SearchParams, page *paging.Page) (*UserSearchListView, error)
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
- name: UserSearchResult
  description: 検索結果 1 件分の DTO。User エンティティから写像し、都道府県名を解決して埋める。
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
- name: UserSearchResults
  description: UserSearchResult の一覧。
  type: "[]*UserSearchResult"
- name: UserSearchListView
  description: 検索結果（ページ分の一覧と総件数）。
  fields:
    - name: Items
      type: UserSearchResults
    - name: Total
      type: int64
```

## Dependencies

```yaml
- tracer                 # observability.TracerFactory -> LayerTracer
- user_repository        # domain/user.Repository（SearchByKeyword / CountByKeyword）
- prefecture_repository  # domain/prefecture.Repository（FindByIDs で都道府県名をまとめて解決。N+1 回避）
```

## Workflow

### ListUsersByKeyword

```yaml
tx_required: false
steps:
  - filter / page が nil の場合は apperror.ErrInvalidArgument を返す
  - tools/search.ParseSearchTokens で filter.Keyword をトークン分割する
  - user_repository.SearchByKeyword でキーワード / active 条件とページング（limit/offset）に基づき取得する
  - 取得した各ユーザーの prefectureID を集め、prefecture_repository.FindByIDs で都道府県をまとめて解決する（子 span で計測）
  - 各ユーザーを UserSearchResult へ写像し、都道府県名を埋めて返す
calls:
  - user_repository.SearchByKeyword
  - prefecture_repository.FindByIDs
errors:
  - filter / page が nil の場合は apperror.ErrInvalidArgument
  - user_repository.SearchByKeyword のエラーをそのまま伝播する
  - ユーザーが参照する都道府県を解決できない場合は参照整合性破れとして apperror.ErrInternal
```

### CountUsersByKeyword

```yaml
tx_required: false
steps:
  - filter が nil の場合は apperror.ErrInvalidArgument を返す
  - tools/search.ParseSearchTokens で filter.Keyword をトークン分割する
  - user_repository.CountByKeyword で検索条件に一致する総件数を取得して返す
calls:
  - user_repository.CountByKeyword
errors:
  - filter が nil の場合は apperror.ErrInvalidArgument
  - user_repository.CountByKeyword のエラーをそのまま伝播する
```

### ListUsersByKeywordWithTotal

```yaml
tx_required: false
steps:
  - ListUsersByKeyword で検索結果一覧を取得する
  - CountUsersByKeyword で総件数を取得する
  - UserSearchListView{Items, Total} を組み立てて返す
calls:
  - user_repository.SearchByKeyword
  - prefecture_repository.FindByIDs
  - user_repository.CountByKeyword
errors:
  - ListUsersByKeyword / CountUsersByKeyword のエラーをそのまま伝播する
```
