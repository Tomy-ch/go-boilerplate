# decimal

[English](README.md) | 日本語

`github.com/shopspring/decimal` をラップした exact-decimal 型です。金額・レートなどの正確な十進量を `float64` 誤差なく表現します。

## 役割

`float64` は `0.1` や `19.99` といった十進小数を正確に表現できないため、金額やレートを float で持つと解析した瞬間に値が壊れます。本パッケージは 10 進の正確な量（多倍長整数の係数 + 指数）を提供し、どの層もこれを使います。さらに vendor 依存を seam の裏に隠蔽し、application コードが `shopspring/decimal` を直接 import しないようにします（`pkg/uuid` の前例に準拠）。

業務意味論は一切持ちません。通貨・非負・最小単位の選択はドメイン層の値オブジェクト（`internal/domain/kernel/money`）が所有します。本パッケージは純粋な十進算術・丸め・スケール変換と DB / ワイヤ境界だけを担います。

## ラップ対象

`github.com/shopspring/decimal`

## 注意点

- ワイヤ表現は **JSON 文字列**（`"19.99"`）です。JSON number は IEEE754 double として復元され精度を失うためです。`UnmarshalJSON` は JSON number も受理し、素の数値を出す外部ペイロードを桁落ちなく取り込みます。
- `ToScaledInt64(n)` は n 桁で 0 から遠い方向へ丸め、`10^n` を掛けて最小単位整数を返します。`int64` を超える場合は `ErrOverflow` を返します。これは汎用機構であり、*どの* `n`（最小単位の桁数）かは呼び出し側が所有する policy です。
- `NUMERIC` 境界のため `sql.Scanner` / `driver.Valuer` を実装します。sqlc override が `NUMERIC` 列を本型へ整合させます。
- `MustParse` はテスト専用です。production では使用しないでください。
- エラーラップのため `pkg/xerrors` に依存します — これは `pkg/` → `pkg/` 依存で唯一許可された例外です（depguard `independent_pkg` で強制）。
