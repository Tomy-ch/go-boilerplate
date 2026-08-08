# Toggle

## 用途

「今その表示が適用されているか」を押下状態として示し、切り替えられるようにします。表示密度や折り返しの入り切りのような、画面の見え方を変える操作に使います。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `Toggle` | `aria-pressed` を持つ切り替えボタンです。押下状態は `pressed` として受け取り、自身では保持しません。 |
| `toggleVariants` | 見た目の variant を組み立てる helper です。同じ見た目を要する `ToggleGroup` などから再利用します。 |

## 利用ケース

- 一覧の表示密度、折り返しの有無など、画面の見え方を切り替える操作
- 適用中かどうかを面の塗りで示したい単独のボタン

## Switch との使い分け

見た目が似ていても責務が違います。分かれ目は「**設定を変えるのか、今の見え方を変えるのか**」です。

| | `Toggle` | `Switch` |
| --- | --- | --- |
| 意味論 | `button` + `aria-pressed` | `checkbox` / `switch` |
| 表すもの | 表示の適用状態（今この見え方になっている） | 設定の入り / 切り（通知を受け取る、公開する） |
| 送信値 | 持たない（操作ボタン） | **持つ**（form の値になる） |
| 効き方 | その画面の見せ方を変える | 保存して永続する |
| 例 | 表示密度、折り返しの有無 | 通知の受け取り、公開 / 非公開 |

保存されて次に開いたときも残るなら [`SwitchNative`](../switch-native/README.md) / [`SwitchClient`](../switch-client/README.md)、その場の見え方だけならこちらです。form の値として複数選択や同意を送るなら `CheckboxNative` を使います。

## 責務境界

SSR first の選定では `◎` に当たります。押下状態を `pressed` として受け取るだけで state を持たないため、`"use client"` を必要としません。その結果、**Server Component からも Client Component からも同じように使えます**。

- URL や form で切り替える場合 … Server 側で `pressed` を決めて渡す
- browser 側の一時的な state で切り替える場合 … 呼び出し元の client island が `useState` の値を渡す

**client island 版は用意していません。** shadcn の生成物は Radix の `Toggle` を使いますが、それが加えるのは非制御 state（`defaultPressed`）と `data-state` 属性だけで、押下の反応そのものは native の `button` が担います。呼び出し元が state を持てば同じことができるため、この 1 つで両方の使い方を満たせます。`Tabs` や `Slider` の client 版と違い、native では実現できない機能はありません。

既定の `type` は `"button"` です。URL に載せて切り替える場合は、呼び出し元が `type="submit"` と `name` / `value` を与えて native form へ載せるか、`Link` で置き換えます。切り替え後の値の決定、URL の組み立て、送信は持ちません。

icon だけを置く場合は `aria-label` でアクセシブルな名前を与えます。押下状態は `aria-pressed` が伝えるため、**名前を「〜を有効にする」「〜を無効にする」と状態で切り替えません**。名前は変えずに状態だけを変えます。

focus 表示は `outline` の idiom（`focus-visible:outline-2` / `outline-offset-2` / `outline-foreground`）に揃えています。押下中と hover の面はどちらも `accent` を使います。生成物は hover に `muted` を当てますが、このリポジトリの `muted` は `muted-foreground` と同じ値のため、背景と文字が同色になって読めなくなります。

## テスト

テストは押下状態を `aria-pressed` で公開すること、状態が変わってもアクセシブルな名前が変わらないこと、呼び出し元の state を反映すること、既定が `type="button"` であること、`type` / `name` / `value` を上書きして native form へ載せられること、disabled、`variant` と `size`、icon のみの命名、a11y 自動検査を確認します。
