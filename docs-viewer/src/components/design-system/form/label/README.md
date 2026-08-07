# Label

## 用途

form control の項目名を利用者へ伝えます。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `Label` | native `label` として control の項目名を表示し、`htmlFor` により対象 input と関連付けます。 |

## 利用ケース

`htmlFor` と対象 control の一意な `id` を対応させて使います。

## 責務境界

説明・必須表示・検証エラー・field 全体の layout は持ちません。`Field` または feature が構成します。

## テスト

項目名との関連付けと a11y を確認します。
