# Scrollbar

## 用途

スクロールする面すべてに共通する scrollbar の見た目を、一箇所で定めます。局所スクロール領域では、scrollbar が「ここは別途スクロールする領域である」ことを示す手掛かりになります。

## 役割と公開 component

公開する React component はありません。`scrollbar.css` が `:root` へ宣言する CSS 基盤です。`globals.css` から import しており、利用側の指定は要りません。

| 宣言 | 役割 |
| --- | --- |
| `scrollbar-width: thin` | scrollbar の太さを揃えます。 |
| `scrollbar-color` | thumb を `muted-foreground`、track を透明にします。 |

## 利用ケース

- ページ本体のスクロール
- `ScrollArea` による局所スクロール領域
- `textarea` / `pre` / `overflow` を持つ任意の要素

いずれも個別の指定は不要です。

## 責務境界

`scrollbar-color` と `scrollbar-width` は**継承プロパティ**です。`:root` で一度宣言すれば配下のスクロール面すべてへ行き渡るため、component ごとには持たせません。component 側で指定すると、同じアプリの中で場所ごとに scrollbar の見た目が割れます。

`auto` 以外の値を与えると overlay scrollbar が classic へ切り替わります。macOS や touch 環境の既定では scrollbar がスクロールするまで現れず、局所スクロール領域の存在に気づけません。常時表示にすることで、触れる前から領域の存在・残量・現在地が判ります。この three つは scroll 位置から導かれるため、静的なアイコンや装飾と違って表示が実態とずれません。

代償として、classic scrollbar は幅を占め、利用者が OS で選んだ overlay の設定を上書きします。ページ内に独立したスクロール面が現れる UI では、その存在を示す手掛かりが他にないため、この上書きを受け入れています。

対応していないブラウザでは browser 既定の scrollbar がそのまま使われます。表示が消えたり誤った状態を示したりはせず、現状から退行しません。

スクロール位置の保持、末尾検知、追加読み込みは持ちません。領域の大きさと方向は `ScrollArea` の担当です。

`scrollbar-gutter` は宣言していません。継承されず要素ごとの layout 判断になるため、内容の増減で横ずれが問題になった時点で、その領域側で指定します。

## テスト

CSS のみで JavaScript を持たないため、単体テストは置いていません。scrollbar の描画は browser と OS が担い、jsdom では再現できません。
