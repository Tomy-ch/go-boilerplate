# CommandService 実装ガイド

## 空でもこのディレクトリが存在する理由

データアクセスのディレクトリは `database/dml/` の DML カテゴリと 1 対 1 で対応します。SQL には存在
するのに Go には存在しないカテゴリは、誰も決めていないカテゴリだからです
([`../README.md`](../README.md) § Directory Structure)。ディレクトリを決めるのはこの対応であって、
実装がたまたま入っているかどうかではありません。したがって最初の CommandService が書かれるまでは、
このガイドだけを持ちます。

## ここに置いてよいもの

判断基準はここでは繰り返しません。[`../README.md`](../README.md) § command_service が持っており、
このパッケージはその 2 つの規則抜きには理解できません。

- **What may live here** — 集約を読み込んで保存する形では表現できない書き込みだけ。read-modify-save
  で表せるものは Repository の担当です。
- **Conditions are derived, never authored** — ここで SQL に強制するガードは、既に存在するドメイン
  不変条件を言い直したものでなければならず、同じドメインの sentinel を返します。

背後にある決定は
[ADR-0032 (lightweight-cqrs)](../../../../docs/adr/0032-lightweight-cqrs.md) と
[ADR-0034 (commandservice-atomicity-criterion)](../../../../docs/adr/0034-commandservice-atomicity-criterion.md)
です。書き込み順序は
[ADR-0036 (ordered-pessimistic-row-locks)](../../../../docs/adr/0036-ordered-pessimistic-row-locks.md)
に従います。

## 配置

```txt
command_service/
 └ <aggregate>/
     └ <aggregate>_command_service.go
```

インターフェースは Domain 層ではなく **Usecase 層**の、QueryService インターフェースの隣に置きます。
理由は [`../query_service/README.md`](../query_service/README.md) が述べるとおり、その形が集約単位
ではなくユースケース単位で決まるからです。

```txt
internal/usecase/<aggregate>/command/<aggregate>_command_service.go
```

Infrastructure はそのインターフェースを実装するだけです。

## コンストラクタ

依存は引数で受け取り、内部では何も構築しません。

```go
func New(
    db driver.DatabaseDriver,
    tf observability.TracerFactory,
) command.CommandService {
    return &commandService{
        db:     db,
        tracer: tf.Infra(),
    }
}
```

## トランザクション

CommandService は**トランザクションを自分で開きません**。`ctx` 経由で渡されたものの上で実行し、
それは `driver.New(ctx, db)` が拾います。境界を持つのは Usecase であり、書き込みは
`idempotency.Run` の下に入れ子になるためです。ここで開くと最初の境界の内側に 2 つ目の境界ができ、
外側のロールバックがこの書き込みを覆わなくなります。

```go
db := gen.New(driver.New(ctx, c.db))
```

## Outbox

CommandService は outbox イベントを**発行しません**。それは Usecase の責務で、`system_cqrs`
カテゴリが担います。ここから外しておくことが、同じ書き込みを、イベントを出すかどうかを自分で
決めるワークフローへ合成できる状態を保ちます。

## エラーと可観測性

sqlc のエラーはすべて `pgerror.NormalizeError` で正規化し、メソッドは注入された `TracerFactory`
を通じて infrastructure 層の span を開きます。いずれも Repository / QueryService と同じです。
[`../pgerror/README.md`](../pgerror/README.md) と [`../README.md`](../README.md) § Observability
を参照してください。

## DI

コンストラクタは `persistenceModule`（`internal/di/module/persistence.go`）の `command_service`
サブモジュールへ登録します。提供する型は Usecase 層のインターフェースであり、具体の struct では
ありません。

```go
fx.Module("command_service",
    fx.Provide(
        <aggregate>cmd.New,
    ),
),
```

## テスト

インテグレーションテストで、触れたすべてのテーブルへの原子的な書き込み効果を書き込み後の `SELECT`
で表明し、古いロック値を使って「ドメインの検査は通るが DB の述語が弾く」状態を作って fail-closed
ガードを表明し、制約違反の正規化を表明します。既定の `testkit` ヘルパーで再現できないものを含む
テスト戦略の全体は [`../README.md`](../README.md) § Test Strategy にあります。
