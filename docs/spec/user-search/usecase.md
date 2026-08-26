# User Search — Usecase Spec

> 既存実装（`internal/usecase/user/search`）を spec 化したもの。手書き実装から逆生成した現状仕様。
> user 集約に対する読み取り専用ユースケース。固有の集約を持たないため domain.md は独立させず、
> 参照する集約の spec（`docs/spec/user/domain.md`）を見る。

## Overview

ユーザー検索ユースケースは、キーワードとアクティブ状態でユーザーを検索する read-only なユースケース。`user.Repository`（domain Repository）の `SearchByKeyword` / `CountByKeyword` を直接呼び、取得した `User` エンティティを検索結果 DTO（`UserSearchResult`）へ写像して返す。都道府県名は `prefecture.Repository.FindByIDs` でまとめて解決してから埋める（N+1 回避）。トランザクションは不要。

読み取り対象は正本である `users` テーブルそのものであり、派生射影ではないため QueryService ではなく Repository に委譲する（判定基準は `docs/design/data-access-pattern.md`）。

キーワードは `tools/search.ParseSearchTokens` でトークン分割してから `user.Repository` へ渡す（アプリケーションポリシー）。

検索は呼出元以外のユーザーを開示するため、3 メソッドとも admin 限定とする。所有者を持たないリソース（`ownerID = nil`）として `authz.Authorizer` に問い合わせることで所有者フォールバックを成立させず、admin ロールを持つ主体だけを通す。件数を返す `CountUsersByKeyword` も一覧と同格に扱う（件数だけでも特定の人物が登録されているかを推測できるため）。認可は入力検証よりも前に置き、拒否された呼出元が検索条件の妥当性すら観測できないようにする。

## Interface

```yaml
package: internal/usecase/user/search
name: Usecase
methods:
  # いずれも他ユーザーを開示するため admin 限定。認可判定のため認証主体 authn を受け取る
  - name: ListUsersByKeyword
    signature: ListUsersByKeyword(ctx context.Context, authn *auth.Authn, filter *SearchParams, page *paging.Page) (UserSearchResults, error)
  - name: CountUsersByKeyword
    signature: CountUsersByKeyword(ctx context.Context, authn *auth.Authn, filter *SearchParams) (int64, error)
  - name: ListUsersByKeywordWithTotal
    signature: ListUsersByKeywordWithTotal(ctx context.Context, authn *auth.Authn, filter *SearchParams, page *paging.Page) (*UserSearchListView, error)
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
- authorizer             # boundary/authz.Authorizer（検索の認可判定。admin 限定）
```

## Workflow

### ListUsersByKeyword

```yaml
tx_required: false
steps:
  - authorizer.Authorize で検索の認可を確認（所有者を持たないリソースとして問い合わせるため所有者フォールバックが成立せず admin のみ許可。authn が nil なら ErrUnauthenticated）
  - filter / page が nil の場合は apperror.ErrInvalidArgument を返す
  - tools/search.ParseSearchTokens で filter.Keyword をトークン分割する
  - user_repository.SearchByKeyword でキーワード / active 条件とページング（limit/offset）に基づき取得する
  - 取得した各ユーザーの prefectureID を集め、prefecture_repository.FindByIDs で都道府県をまとめて解決する（子 span で計測）
  - 各ユーザーを UserSearchResult へ写像し、都道府県名を埋めて返す
calls:
  - authorizer.Authorize
  - user_repository.SearchByKeyword
  - prefecture_repository.FindByIDs
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）。認可判定そのものを行わない
  - 認可が拒否された場合は authz.ErrForbidden（403）。入力検証にもリポジトリにも到達しない
  - filter / page が nil の場合は apperror.ErrInvalidArgument
  - user_repository.SearchByKeyword のエラーをそのまま伝播する
  - ユーザーが参照する都道府県を解決できない場合は参照整合性破れとして apperror.ErrInternal
```

### CountUsersByKeyword

```yaml
tx_required: false
steps:
  - authorizer.Authorize で検索の認可を確認（ListUsersByKeyword と同じ admin 限定の判定。authn が nil なら ErrUnauthenticated。件数は特定ユーザーの存在を推測させるため一覧と同格に扱う）
  - filter が nil の場合は apperror.ErrInvalidArgument を返す
  - tools/search.ParseSearchTokens で filter.Keyword をトークン分割する
  - user_repository.CountByKeyword で検索条件に一致する総件数を取得して返す
calls:
  - authorizer.Authorize
  - user_repository.CountByKeyword
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）。認可判定そのものを行わない
  - 認可が拒否された場合は authz.ErrForbidden（403）。入力検証にもリポジトリにも到達しない
  - filter が nil の場合は apperror.ErrInvalidArgument
  - user_repository.CountByKeyword のエラーをそのまま伝播する
```

### ListUsersByKeywordWithTotal

```yaml
tx_required: false
steps:
  - authorizer.Authorize で検索の認可を確認（ListUsersByKeyword と同じ admin 限定の判定。authn が nil なら ErrUnauthenticated）
  - 認可済みの内部処理として検索結果一覧と総件数を取得し、UserSearchListView{Items, Total} を組み立てて返す（認可はこの入口で 1 度だけ行い、内部処理では繰り返さない）
calls:
  - authorizer.Authorize
  - user_repository.SearchByKeyword
  - prefecture_repository.FindByIDs
  - user_repository.CountByKeyword
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）。認可判定そのものを行わない
  - 認可が拒否された場合は authz.ErrForbidden（403）。入力検証にもリポジトリにも到達しない
  - 一覧取得・件数取得のエラーをそのまま伝播する
```
