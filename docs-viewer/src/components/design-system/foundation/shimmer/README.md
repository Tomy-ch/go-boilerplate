# Shimmer

## 用途

進捗が測れない処理が今も動いていることを、面の上を流れる帯で示します。

## 使い方

`shimmer` class を付けるだけの opt-in の CSS 基盤です。React component は公開しません。

```tsx
<div className="shimmer h-16 rounded-md bg-muted" />
```

状態に応じて出す場合は Tailwind の variant と合成します。

```tsx
<span className="group-data-[state=uploading]/attachment:shimmer">{fileName}</span>
```

## `Skeleton` との違い

| | 伝えること |
| --- | --- |
| `Skeleton`（`animate-pulse`） | ここに箱がある |
| `shimmer` | 止まっていない |

置き換えではありません。読み込み中の形を示したうえで「まだ動いている」ことも伝えたい長い処理では、両方を使います。

**同じ要素には付けられません。** `animate-pulse` と `shimmer` はどちらも `animation` プロパティを使うため、後に効いたほうだけが残ります。両方を出す場合は `Skeleton` の子として重ねます。

```tsx
<Skeleton className="h-16 w-full">
  <div className="shimmer size-full rounded-md" />
</Skeleton>
```

## 併用が必要な理由

`prefers-reduced-motion` では帯ごと消えます。動きを止めたときに帯だけが残ると、止まった装飾が画面に居座るためです。

**消えた状態では処理中であることが何も伝わりません。** `Skeleton` か待機の文言を必ず併用してください。これは呼び出し元の責務です。

## 見た目の決め方

帯の色は前景色から作ります。「明るい帯」を固定すると light テーマでは背景に埋もれるため、どちらのテーマでも面との差が出る側から取ります。

帯の幅は面の 40% です。面と同じ幅にすると、流れているのではなく全体が明滅しているように見えます。

周期は 1.6 秒です。遅いと止まって見え、速いと注意を奪うため、その間で採っています。帯の開始と終了は面の外に置き、端で現れたり消えたりせず通り過ぎたように見せます。

## 責務境界

処理中かどうかの判定、待機の文言、表示時間は持ちません。付けるかどうかを呼び出し元が決めます。

文字に付けた場合、帯は文字の背後を通ります。文字自体の色は変えないので、読めるまま残ります。
