# Input

## 用途

単一行の native `input` を表示・送信します。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `Input` | native `input` の属性・form 送信を保ったまま見た目を統一する、SSR first の単一行入力です。 |

## 利用ケース

`name`、`type`、`autoComplete`、`required` を使う通常の form 入力に使います。

## 責務境界

項目名は `Label`、説明・検証エラーは `Field` または feature が構成します。`aria-invalid` と `aria-describedby` の関連付けも呼び出し側が行います。

## テスト

 native 属性と a11y を確認します。
