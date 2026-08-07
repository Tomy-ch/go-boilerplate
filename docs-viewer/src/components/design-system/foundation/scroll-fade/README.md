# ScrollFade

## 用途

スクロールする領域の端をぼかし、続きがあることを示す CSS 基盤です。React component は公開しません。

| utility | 方向 |
| --- | --- |
| `scroll-fade-x` | 横 |
| `scroll-fade-y` | 縦 |

## いつ使うか

**scrollbar を消した領域だけです。** [`scrollbar`](../scrollbar/README.md) が存在・残量・現在地を常時示すため、scrollbar がある領域にこの演出は要りません。

要るのは `scrollbar-none` を当てた領域です。そこでは続きがあることを示す手掛かりが他にありません。`AttachmentGroup` がこれに当たります。

```tsx
<div className="scroll-fade-x flex gap-3 overflow-x-auto scrollbar-none">…</div>
<div className="scroll-fade-y flex flex-col gap-3 overflow-y-auto scrollbar-none">…</div>
```

収まりきる領域に付けても端の飾りにしかなりません。溢れうる領域にだけ付けます。

## 挙動

端に着いた側のぼかしは外れます。先頭では手前の端がぼけず、末尾まで送ると先の端のぼかしが外れるため、scrollbar が無くても「先頭に居る」「まだ続く」が端の見た目だけで判ります。

追従には scroll-driven animation を使い、`@supports` で囲っています。使えない環境では両端をぼかした状態で止まります。続きが無いのにぼけるより、続きがあるのに手掛かりが無いほうが困るため、そちらへ倒しています。

ぼかし幅は `--scroll-fade-size` で、既定は 6（`--spacing` 換算）です。

## 2 方向は重ねられません

`scroll-fade-x` と `scroll-fade-y` を同じ要素へ付けても、両方向はぼけません。どちらも `mask-image` を使うため、後に効いたほうだけが残ります。両方向へスクロールする領域には使わないでください。

## 実装上の制約

端ごとのぼかし量は長さではなく **0〜1 の比**で持ちます。長さそのものを keyframe で動かすと値に `var()` を含むことになり、登録済み custom property の keyframe として解決されず、animation ごと無効になります。比なら keyframe を定数だけで書けるため、ぼかし幅は utility 側の 1 か所に残せます。

## 責務境界

スクロール領域そのもの、scrollbar を消すかどうか、中身の並べ方は持ちません。付けるかどうかを呼び出し元が決めます。
