# Dialog

## 用途

内容の補助表示や通常の編集操作を、画面を覆う modal として開きます。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `Dialog` | 開閉状態と focus trap・Escape・背面の inert 化を管理する client-side root です。 |
| `DialogTrigger` | Dialog を開く trigger です。`Button` や link を使う場合は `asChild` で合成します。 |
| `DialogContent` | overlay と Portal を伴って dialog 本体を描画します。既定で右上に閉じる操作を置きます。 |
| `DialogClose` | dialog を閉じる操作です。footer のキャンセル・閉じるに使います。 |
| `DialogHeader` | title と説明をまとめる領域です。 |
| `DialogTitle` | dialog のアクセシブルな名前になる title です。 |
| `DialogDescription` | dialog の目的や影響を説明する本文です。 |
| `DialogFooter` | 操作を並べる領域です。配置だけを担い、操作自体は持ちません。 |
| `DialogOverlay` | 背面を覆う overlay です。`DialogContent` が内部で描画します。 |
| `DialogPortal` | 描画先の Portal です。`DialogContent` が内部で使います。 |

## 利用ケース

一覧から詳細を開く、名称や設定を編集する、画像を拡大するなど、画面遷移せずに内容を確認・編集したい場面に使います。

退会・削除のような取り消せない操作の確認には使いません。その用途は `role="alertdialog"` の意味論を持つ `AlertDialog` を使います。

## 責務境界

focus trap・Escape・開閉・背面の inert 化のため hydration が必要な client island です。表示する文言、取得、保存、業務判断、開閉を URL へ載せるかの選択は持ちません。内容自体に client runtime が要らない場合は、Server Component で組み立てた要素を `children` として渡します。

`DialogFooter` は配置だけを担い、閉じる操作を自前で描画しません。閉じる操作は `DialogClose` を `Button` へ合成して呼び出し元が置きます。shadcn の生成物にある footer 側の `showCloseButton` は、`DialogContent` の閉じる操作と重複し文言を component へ持ち込むため採っていません。

`DialogContent` はアクセシブルな名前として `DialogTitle` を必ず子に置きます。説明が要る場合は `DialogDescription` を添え、不要な場合は `aria-describedby={undefined}` を明示します。どちらも無いと Radix が警告します。

面はページ内容の上へ重なるため、背景は `bg-background`、境界は `border-border` の semantic token で不透明に描画します。トークンに定義のない class は Tailwind が CSS を出力せず、面が透明のまま背後の文字と重なって contrast を失います。

vendor は現在 Radix と lucide ですが、公開 API に vendor 名は含めません。

## テスト

テストは開くまで内容を描画しないこと、title と説明の関連付け、`alertdialog` ではなく `dialog` の意味論であること、面が不透明であること、右上と footer の双方から閉じられること、`showCloseButton` の切り替え、Escape での閉じ、`DialogPortal` / `DialogOverlay` の明示指定、a11y 自動検査を確認します。
