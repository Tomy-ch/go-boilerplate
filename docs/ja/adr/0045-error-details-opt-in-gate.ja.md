---
status: accepted
date: 2026-07-17
deciders: [maintainers]
tags: [errors, architecture, api, security]
---

# ADR-0045: スキーマ分割によるエラー details の opt-in ゲート

English canonical: [0045-error-details-opt-in-gate.md](../../adr/0045-error-details-opt-in-gate.md)

## ステータス

accepted

[ADR-0044](0044-error-metadata-code-message-details.ja.md) を精緻化する。

## コンテキスト

[ADR-0044](0044-error-metadata-code-message-details.ja.md) は `apperror.Meta` を導入し、エラー
発生箇所が `details`(不正フィールド名等の公開安全な識別子)を付与してレスポンスに露出できる
ようにした。しかし出荷状態では露出が **fail-open** だった: 単一の共有 `ErrorResponse` スキーマ
(details は任意フィールド)が全 error レスポンスの裏側にあったため、チェーンのどこかで `Meta`
の details が付けば**全エンドポイント**でそれが描画される。

これは漏えいリスクである。具体的な引き金は、保存済み行からのエンティティ再構築
(`rowToUser` → `user.New`)だった: データ不整合が `422` としてクライアントに届き、その `details`
が内部フィールド名を名指しする — 露出を意図していないエンドポイントで。この特定経路を
ハードニングした後も、「何かが剥がさない限り details は漏れる」という構造的既定は残り、
セキュリティ関連フィールドとしては誤った向きだった。

露出を**エンドポイントごとの opt-in** にし、その判断を API 契約で表現し(契約・生成クライアント型・
実行時挙動が一致するように)、transport の edge で **fail-closed** に強制したい。

## 決定

error レスポンスのエンベロープを 2 スキーマに分割し、details を edge でゲートする。

1. **契約(SSOT)。** `openapi/components/schemas/ErrorResponse.yaml` を base エンベロープ
   (`code` / `message` / `requestId`、`details` **なし**)にする。新規
   `ErrorResponseWithDetails.yaml` が `details` を追加する。details を意図的に露出する
   レスポンスだけが `WithDetails` スキーマを参照する(現状は `PostUsers` / `PutUsersDetail` /
   `PatchUsersDetail` の `422`、`errors/UnprocessableEntity422.yaml` 経由)。他の error レスポンス
   はすべて base スキーマを参照したまま。**どの operation が `ErrorResponseWithDetails` を
   参照するかが opt-in スイッチ**であり、それは OpenAPI spec に完全に閉じている。

2. **builder 型。** 型生成用エンドポイント(`GenerateErrorSchema`)が `ErrorResponseWithDetails`
   を参照するため、response builder の生成型は superset(`gen.ErrorResponseWithDetails`)になる。
   `HTTPErrorResponse` はこの superset を埋め込む — 唯一の builder DTO は常に `Details`
   フィールドを**持つ**。`details` が wire に届くかは builder ではなく下流で決まる
   (builder は ADR-0044 どおり request 非依存のまま)。

3. **edge の fail-closed ゲート。** `DetailPolicy`(起動時に spec から `gorillamux.NewRouter` +
   `operationId → details 公開可` の前計算マップで 1 度だけ構築)を `errorhandler` に注入する。
   エラー経路で、レスポンスが `details` を持つ場合、handler はリクエストの operation を解決し、
   その operation が opt-in していない限り**クライアント wire** から `details` を落とす。
   `resp` オブジェクト(したがってログ)には完全な details を残す。

これは既存の `requestId` イディオムの鏡像である: request 非依存の `error/response` package が
骨格を作り(`requestId` を空にし、エラーが持つ `details` を付与する)、request を知る
`errorhandler` が仕上げる(`requestId` を埋め、opt-in でなければ `details` を剥がす)。ゲートは
Host 非依存 — policy 用 router は servers を除去した spec の複製から作るため、proxy / test の
Host でもパス + メソッドで解決できる。

## 帰結

### ポジティブな帰結

- details 露出が **fail-closed**: 新エンドポイントは `ErrorResponseWithDetails` を宣言するまで
  `details` を返さないため、宣言忘れが漏えいにならない。
- opt-in 判断が OpenAPI 契約中の単一の機械可読な事実になる。生成クライアント型と実行時挙動が
  一致し、分割は監査可能。
- wire が省略しても、デバッグ用にログは完全な `details` を保持する。
- エンドポイントごとの middleware も、ホットパスのコストも無い(router はエラー経路でのみ走る)。

### ネガティブな帰結

- 「details がいつ出るか」が 2 package にまたがる(`error/response` が付与、`errorhandler` が
  ゲート)— `requestId` イディオムが既に持つのと同じ 2 箇所コスト。
- `ErrorResponseWithDetails` 構造体は、`422` を宣言する handler `gen` package に重複する
  (oapi-codegen のパッケージ単位生成の通常挙動)。
- details を露出**すべき**なのにスキーマ宣言を忘れたエンドポイントは、静かに何も返さない。
  スキーマが唯一の opt-in スイッチであることは `docs/rules.md` と `apperror` / `error/response`
  の README に明記する。

## 検討した代替案

### vendor 拡張(`x-expose-error-details`)

スキーマ分割ではなく operation / response の vendor 拡張で opt-in を印す。棄却: スキーマ分割は
契約そのもの(と生成クライアント型)に「どのレスポンスが `details` を持つか」を語らせる。
vendor 拡張はクライアントから見えないサイドチャネル。

### `error/response` builder でゲート

`NewHTTPErrorFromAppError` に `exposeDetails` フラグを渡す。棄却: opt-in 判断は request 依存
(マッチした operation が必要)だが、builder は意図的に request 非依存(純粋な `apperror → 形`
写像)。edge 時の request スコープ仕上げは `errorhandler` の既存の役割(`requestId`・ログ・
commit 判定)。

### エンドポイントごとの opt-in middleware

details を露出する各ルートにフラグを立てる middleware。棄却: OpenAPI 契約に既にある真実を
重複させ drift する。加えて全リクエストにホットパスの仕事を足す。

## 備考

- 出典: `internal/apperror/README.ja.md`(エラーメタ情報 節)、
  `internal/controller/error/response/README.ja.md`、
  `internal/controller/httpstack/errorhandler/README.ja.md`。
- base `ErrorResponse` の Go 型は今も handler `gen` package にレスポンス alias
  (`BadRequest400 = ErrorResponse` 等)として生成されるが、手書きコードは使わない
  ドキュメント用生成物。builder package(`error/response/gen`)は superset のみ生成する。
