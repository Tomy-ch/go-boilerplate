# 入力境界値のオーナーシップ

[English](boundary-ownership.md) | 日本語

`maxLength: 50` のような制約は単一の事実に見えますが、**同じ境界値が複数のレイヤに存在し、それぞれ別の関心事がオーナー**になっていることがあります。本ガイドは「誰が何を所有するか」を定義し、OpenAPI の制約が **domain の業務ルールと取り違えられること**を防ぎます。

> [!IMPORTANT]
> OpenAPI の `minLength` / `maxLength` / `minimum` / `maximum` / `pattern` は **ワイヤー契約**（HTTP API がネットワーク越しに受け入れる／約束する形）を表します。domain の業務ルールの正本（source of truth）では **ありません**。値が一致しないことは正当にあり得ます。OpenAPI の制約を「これが domain の上限だ」と読まないでください。

## 「同じ」数値に対する2つの別オーナー

|関心事|オーナー|置き場所|意味|
|---|---|---|---|
|**ワイヤー契約**|OpenAPI|`openapi/components/schemas/*.yaml`|API が HTTP 越しに受け入れる（リクエスト）／約束する（レスポンス）形|
|**業務ルール**|domain|`internal/domain/<aggregate>/constant.go`|業務として valid と認める値|
|**格納容量**|DB|`database/migrations/*.sql`|物理カラムの上限（例: `VARCHAR(100)`）|

これらは偶然同じ数値を共有することが多いですが、**答えている問いが違い**、**変更される理由も違います**。OpenAPI は domain の値を所有しているのではなく、ワイヤーが満たすべき契約を宣言しているだけです。

## 方向ごとの不変条件

レイヤ間の関係は**方向によって非対称**です：

```text
OpenAPI リクエスト制約  ⊆  domain ルール  ⊆  OpenAPI レスポンス容量
       (最も厳しい)                                (最も緩い)
```

- **リクエスト** — OpenAPI は domain より*厳しく*てよい。リクエスト検証ミドルウェア（`internal/controller/httpstack/oapi/`）が範囲外の入力を domain に届く**前に**弾くため、ワイヤー側を厳しくするのは安全な向きです。
- **レスポンス** — OpenAPI レスポンス制約は domain が出しうる値を**包含（superset）**していなければなりません。domain（または HTTP 以外の書き込み経路）がレスポンススキーマの禁じる値を生成できると、サーバは契約違反のレスポンスを出し、それを**サーバ側では誰も捕捉できません**（実行時のレスポンス検証は行っていません。[`internal/controller/httpstack/oapi/README`](../internal/controller/httpstack/oapi) 参照）。唯一それが表面化するのはクライアント側で生成された検証（例: `orval` + `zod`）であり、サーバ側の契約違反を発見する場所として裏返っています。

<!-- sample-api:begin -->
## 教材としての具体例：`firstName` の長さ

本リポジトリは**リクエスト側に意図的に食い違った値を教材として残しています**：

|レイヤ|`firstName` の最大長|強制する主体|
|---|---|---|
|OpenAPI リクエスト (`UserBaseInputRequest.yaml`)|`50`|リクエスト検証ミドルウェア（実行時）|
|domain (`constant.go` の `maxFirstNameLength`)|`100`|エンティティのコンストラクタ|
|DB (`first_name`)|`VARCHAR(100)`|データベース|
|OpenAPI レスポンス (`UserResponse.yaml`)|`100`|契約上の約束——`domain ⊆ レスポンス` を満たすよう domain に揃えた|

この例が教えること：

- **リクエストのワイヤー契約（50）は domain 容量（100）より意図的に厳しい。** OpenAPI の `50` を「domain のルール」と読むのは誤り——domain のルールは `100` です。これは**正当で安全な向き**（`リクエスト ⊆ domain`）：ミドルウェアが 50 超の入力を domain に届く前に弾きます。
- **レスポンス制約はリクエスト（50）ではなく domain（100）に揃える。** レスポンスとリクエストは別の関心事で、レスポンスは*domain が出しうるすべて*を包含する必要があります（`domain ⊆ レスポンス`）。こうすれば HTTP 以外の経路（seed・バッチ・将来のエンドポイント）で書かれた値でもレスポンス契約に違反しません。ここを 50 にすると、その隙間が再発し、クライアントの `zod` だけが気づくサーバ側の契約違反に戻ってしまいます。

この例の主旨は「すべての数値が*一致すべき*」ということ**ではなく**、各値が**別のレイヤに別の理由で所有されている**ことです。リクエストは domain より厳しくてよい（教材としての食い違い）が、レスポンスは domain より厳しくしてはいけない（守るべき不変条件）。「OpenAPI の制約」と「domain のルール」を混同するのが避けるべき誤りです。

<!-- sample-api:end -->

## メンテナー向けルール

- domain 定数を OpenAPI に（あるいは逆に）コピーして「常に等しく保たねばならない」と**思い込まない**こと。各値はそれ自身の関心事から決める。
- 方向の不変条件を**保つ**こと：`リクエスト ⊆ domain ⊆ レスポンス容量`。
- リクエスト制約が domain ルールより**厳しい**のは問題なし（ミドルウェアが先に弾く）。
- **レスポンス**制約を厳しくするときは、どの書き込み経路もその範囲外の値を出さないことを確認する。
- 不変条件を CI で守りたい場合は、`openapi.gen.yaml` と domain 定数を読み `リクエスト ≤ domain ≤ レスポンス` を assert するテストを追加する。

## 各値の置き場所

|レイヤ|パス|
|---|---|
|OpenAPI リクエスト／レスポンス制約|`openapi/components/schemas/*.yaml`|
|domain 境界定数|`internal/domain/<aggregate>/constant.go`|
|DB カラム上限|`database/migrations/*.sql`|
|クライアント側検証（消費者自身の関心事）|本 spec から生成（例: `orval` + `zod`）|
