# Accordion

## 用途

関連する複数の詳細を、必要な項目だけ開いて確認できるようにします。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `Accordion` | 複数の項目を縦にまとめる外枠です。 |
| `AccordionItem` | native の `details` による一つの開閉項目です。 |
| `AccordionTrigger` | native の `summary` として、項目の見出しと開閉操作を提供します。 |
| `AccordionContent` | 項目を開いたときに表示する詳細内容です。 |

## 利用ケース

補足情報、設定群、狭い画面で段階的に見せる説明を表示する場合に使います。複数項目が同時に開いてよい内容に向きます。

## 責務境界

SSR first の native `details` / `summary` であり、hydration は不要です。常に一項目だけを開く制御、開閉状態の外部同期、アニメーション、高度な keyboard 操作は持ちません。それらが必要になった場合だけ client island を追加します。

## テスト

test は native 開閉、初期 open、a11y を確認します。
