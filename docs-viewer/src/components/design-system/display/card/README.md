# Card

## 用途

関連する情報や補助操作を一つの視覚的なまとまりにします。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `Card` | 関連する情報と操作を包む外枠です。 |
| `CardHeader` | 見出し・説明・header 内の操作をまとめる領域です。 |
| `CardTitle` | Card の主題を示す見出しです。 |
| `CardDescription` | 見出しを補足する説明文です。 |
| `CardAction` | header の末尾に置く補助操作の領域です。 |
| `CardContent` | Card の主な内容を置く領域です。 |
| `CardFooter` | 主な内容に続く補助情報や操作を置く領域です。 |

## 利用ケース

概要、補助情報、要約値などをひとまとまりに表示する場面に使います。

## 責務境界

業務型・取得処理・遷移先・クリック範囲は持ちません。意味論と操作は feature が組み立てます。

## テスト

 subcomponent の合成・className の拡張・a11y を確認します。
