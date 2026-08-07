# Textarea

## 用途

複数行の native `textarea` を表示・送信します。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `Textarea` | native `textarea` の属性・form 送信を保ったまま見た目を統一する、SSR first の複数行入力です。 |

## 利用ケース

`name`、`rows`、`required`、`value` などを使う複数行の form 入力に使います。

## 責務境界

項目名・説明・検証エラーは `Label` / `Field` または feature が構成します。`aria-invalid` と `aria-describedby` の関連付けも呼び出し側が行います。

## テスト

 native 属性と a11y を確認します。
