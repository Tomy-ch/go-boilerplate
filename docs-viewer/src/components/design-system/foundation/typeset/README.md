# Typeset

## 用途

sanitizer 済みの Markdown / HTML を一定の組版 rhythm で表示します。

## 役割と公開 class

| Class | 役割 |
| --- | --- |
| `.typeset` | HTML 要素へ共通の組版 rhythm を適用する起点です。 |
| `.typeset-<preset>` | font、size、leading、flow を文脈ごとに上書きする preset です。 |
| `.not-typeset` / `[data-not-typeset]` | 配下を Typeset の適用対象から外す escape hatch です。 |
| `.typeset-scroll` | 横に収まらない table などを横スクロール可能にする wrapper です。 |

`typeset.css` がこれらを定義する CSS foundation で、Story は `Foundation/Typeset` に置きます。

## 利用ケース

記事、ドキュメント、ストリーミング表示など、HTML 要素の組版を共通化する場面で renderer の外側へ付与します。

## 責務境界

renderer・sanitizer・最大幅・業務コンテンツを持ちません。安全な HTML と layout は呼び出し側が所有します。

## テスト

CSS のみで JavaScript を持たないため、単体テストは置いていません。組版の見え方は実際の描画でしか確かめられず、jsdom では再現できません。
