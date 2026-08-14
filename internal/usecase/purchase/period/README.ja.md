# period

[English](README.md) | 日本語

購入の読み取りにおける**対象期間** — 一覧や集計を、どの範囲の注文日に限定するか — を解決します。本パッケージが持つのは 2 つの型と、その間の翻訳です。

- **`Spec`** — 利用者が要求したもの。クエリパラメータをそのまま写した `Kind`（`all` / `month` / `range` / `recent`）と、その区分が必要とするフィールドを持ちます。この時点では何も検証しません。
- **`Window`** — 解決された答え。両端を含む 2 つの暦日か、「絞り込まない」のいずれかです。ゼロ値が「絞り込まない」を表すため、期間を指定しない呼び出しは既定で全期間になります。

絞り込む `Window` を得る唯一の経路が `Resolve` で、「直近 10 日」のような相対指定が 2 つの具体的な日付になるのもここです。

## なぜ usecase 層に置くか

他の 2 つの置き場所はどちらも、このリポジトリでは誤りです。

**domain ではない。** [`internal/domain/README.md`](../../../domain/README.md) は *Search specifications* と *Aggregation processing* を domain の非責務として列挙しています。対象期間は購入集約の不変条件でも状態遷移でもなく、集約に対して発する問いを狭める条件です。

**infrastructure でもない。** infrastructure は決定済みの読み取りを実行する層であって、読み取りの意味を決める層ではありません。`recent` を暦日へ解決することは「どの日が対象か」という**業務可視の決定**であり、API はその日付をクライアントへ返します。クエリサービスの内側で決めると、レスポンスを所有する層からその決定が見えなくなり、DB 無しには検証できなくなります。ここに置くことは [`internal/usecase/README.md`](../../README.md) の Time Handling Policy（時刻取得は usecase 層に一元管理し、`clock.Clock` バウンダリ経由で内側へ渡す）とも一致します。

> `dashboard` feature は自分のウィンドウをクエリサービスの内側で解決しています。そちらが 2 つの形のうち古い方で、Time Handling Policy から外れている側です。倣うべきは本パッケージの形です。

`Window` は reify された Specification ではなく**問い合わせパラメータ**です。利用者が選んだ日付を運ぶという点で、`purchase.ListFeedParams` が keyset 境界を運ぶのと同じ位置づけになります。これらの読み取りに伴う業務判定（例: 「キャンセル済みは集計対象から除く」）の方は、[`internal/domain/README.md`](../../../domain/README.md) が記録する Evans からの逸脱のとおり、値を持つ側の名前付き述語として留めます。

## 振る舞い

|`Spec.Kind`|必須フィールド|解決される期間|
|---|---|---|
|`all`（未設定・未知の区分を含む）|—|絞り込まない（`Filtered()` が `false`）|
|`month`|`Month`（`YYYY-MM`）|その月の月初日 → 月末日|
|`range`|`From` / `To`|指定された 2 つの暦日（両端を含む）|
|`recent`|`Days`|今日の `Days` 日前 → 今日（両端を含む）|

- どの区分も注入された `*time.Location` で解決し、`time.Local` は使いません。コンテナの既定は UTC なので、依存させると設定と異なる暦日で集計してしまいます。
- **`now` はロケーションへ変換し、`From` / `To` は変換しません。** `now` は現地の暦日として読むべき「瞬間」である一方、`From` / `To` は利用者が既に名指しした「暦日」であり、変換すると UTC より西のロケーションで前日へずれます。
- 月末日は「翌月の月初から 1 日戻す」で求めるため、月の日数表を持たずに 28 / 29 / 31 日が出ます。
- `Bounds()` は SQL 述語で使う半開区間 `[after, before)` を返します。`before` は終了日の**翌日**で、これが終了日当日の任意の時刻に置かれた注文を含める仕組みです。
- エラーはいずれも `apperror.ErrInvalidArgument` です（区分ごとの必須フィールド欠落、`Month` の書式不正、`To` が `From` より前、`Days` が 1 未満）。wire 契約（OpenAPI）はこれより意図的に厳しく、`Days` の上限はそちらが持ちます。ここでの検査は usecase を直接呼ぶ経路のための安全弁です。

## 使い方

```go
// controller: 検証せず、クエリパラメータをそのまま Spec へ詰める。
spec := period.Spec{Kind: period.Kind(*params.Period), Days: ptr.Map(params.Days, ...)}

// usecase: 一度だけ解決し、同じ window をクエリとレスポンスの両方に使う。
window, err := period.Resolve(spec, u.clk.Now(), u.loc)
if err != nil {
    return SummaryView{}, err
}
if window.Filtered() {
    after, before := window.Bounds() // SQL 述語用の [after, before)
}
```

解決は 1 リクエストにつき 1 回にし、`Window` を持ち回してください。同じ `Spec` を後から解決し直すと、リクエストが日付をまたいだ場合に別の日に着地します。

## テスト方針

本パッケージは **Repository も Boundary も持ちません**（`Resolve` は clock を自分で引かず `now` を引数で受けます）。したがって usecase 層の Testing Strategy のうち、モックした Repository / Boundary / トランザクションに関する観点は**適用対象外**であり、欠落ではありません。代わりに固めているのは暦の計算です — 各区分の振り分け、月末と閏年の境界、ロケーション依存の「今日」（UTC では前日に当たる瞬間を含む）、そして半開区間の上限。
