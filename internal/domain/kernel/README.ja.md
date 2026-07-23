# kernel

[English](README.md) | 日本語

domain の **共有カーネル**（DDD *Shared Kernel*）: 集約横断で共有する業務 VO を、意図的に小さく保った
集合として置く場所です。他の `internal/domain/**` パッケージは `kernel/` を import してよく、`kernel/`
自身は `pkg/**` と `internal/apperror` のみに依存し、集約には依存しません。

## なぜ存在するか

`money.Price` のような VO は複数の集約から使われます（現在 `product.price`、目前で
`purchase_details.unit_price` / `purchases.*_amount`）。単一集約には置けず（他集約が中を参照することになる）、
`pkg/` にも置けません（業務ロジック禁止）。そのためここに置きます — [ADR-0104](../../../docs/adr/0104-domain-shared-kernel.md) 参照。

## 入場基準（`shared` / `common` のようなゴミ箱ではない）

以下を **すべて** 満たす場合のみパッケージを追加します。

- **値オブジェクト / ドメイン概念**であること（集約や集約固有のサービスではない）
- 実際に **2 つ以上の集約**から使われる（または目前でそうなる）こと — 「いつか再利用するかも」は不可
- `pkg/` に置けない **業務意味論**（通貨・非負・最小単位・税 …）を持つこと — 文脈非依存の汎用ユーティリティなら `pkg/` へ
- 追加は **共同所有**の判断であること — カーネルの変更は依存する全集約に波及するため保守的に進める

この基準を満たさない型は、所有する集約内に留めてください。

## 強制

depguard（`.golangci-full.yaml` の `maintain_a_sound_domain`）が domain ファイルに対し
`internal/domain/` を deny し、`internal/domain/kernel` を allow します。よって domain→kernel は許可、
domain→他集約は禁止です。

## パッケージ

- `money` — `Price` 値オブジェクト（非負の価格スケール Decimal・最小単位変換を所有）。
  正確な十進の器は `pkg/decimal`（[ADR-0102](../../../docs/adr/0102-exact-decimal-pkg-wrap.md)）。
