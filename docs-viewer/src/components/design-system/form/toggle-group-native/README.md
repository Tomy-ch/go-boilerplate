# ToggleGroupNative

## 用途

関連する切り替えを 1 つの集合として並べ、選んだ値を form として送信します。表示通貨やランキング期間のように、選択肢が横に並ぶ切り替え群に使います。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `ToggleGroupNative` | 集合を表す `fieldset` です。`aria-label` で何の切り替えかを示します。 |
| `ToggleGroupNativeItem` | 集合の中の 1 項目です。実体は視覚的に隠した native input と、それを映す `label` です。 |

## 利用ケース

- 表示通貨・ランキング期間のような、form として送る排他選択
- 表示する列のような、form として送る複数選択

URL にも form にも載せない即時の表示切替には `ToggleGroupClient` を使います。

## ToggleGroupClient との使い分け

| | `ToggleGroupNative` | `ToggleGroupClient` |
| --- | --- | --- |
| 実体 | native の radio / checkbox | button（Radix） |
| form の値 | **そのまま送信される** | 持たない |
| hydration | 不要 | 必要 |
| 項目間の移動 | browser 標準（radio は矢印キー） | roving tabindex（Radix） |

## 責務境界

SSR first の選定では `○` に当たります。項目が native の radio / checkbox なので、選択は form の値としてそのまま送信され、初期表示も Server 側で確定します。client runtime を必要としません。

排他選択は項目を `type="radio"`、複数選択は `type="checkbox"` にし、集合内で同じ `name` を与えます。選択肢どうしの移動は browser の標準動作に従います。

`fieldset` として公開されるため、`aria-label` か `aria-labelledby` で**何の切り替えかを必ず示します**。`legend` を置く場合はそちらが名前になります。

送信後の処理、URL の組み立て、選択の永続化は持ちません。

### 隠した input の扱い

項目は `label` を表示要素にし、その中の input を `sr-only` で視覚的にだけ隠しています。`display: none` や `aria-hidden` では支援技術からも keyboard からも到達できなくなるため使いません。focus は input が受け取り、見た目は `label` が `has-[:checked]` / `has-[:focus-visible]` で追従します。

選択中の面と大きさは `toggleVariants` を共有しているため `Toggle` と揃います。隣接する項目は境界を重ね、角丸は両端だけに付けてひと続きの segmented control に見せます。

`RadioGroupNative` とは意味論が同じで見た目だけが違います。ドット付きの縦並びなら `RadioGroupNative`、押下面の横並びならこちらを使います。

## テスト

テストは名前を持つ `group` として公開されること、排他選択が radio・複数選択が checkbox になること、`name` / `value` を native 属性として持つこと、排他的な切り替えと複数同時選択、隠した input が支援技術と keyboard から到達できること、disabled、選択中の面と大きさが `Toggle` と同じ token であること、a11y 自動検査を確認します。
