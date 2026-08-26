# SearchFieldClient

## 用途

打鍵に追従してキーワード検索を通知する検索欄です。入力が止まったところで検索語を呼び出し元へ渡し、送信ボタンを押さなくても結果が変わる導線を作ります。

## 役割と公開 component

| Component / 値 | 役割 |
| --- | --- |
| `SearchFieldClient` | `search` 要素の landmark として、検索入力と消去ボタンをまとめます。入力が止まると `onSearch` を呼びます。 |
| `SEARCH_FIELD_DEBOUNCE_MS` | 入力が止まってから通知するまでの既定の待ち時間（ミリ秒）です。 |

主な props は次のとおりです。

| props | 役割 |
| --- | --- |
| `label`（必須） | 検索入力のアクセシブルな名前です。視覚的なラベルを持たないため、これだけが「何を検索する欄か」を伝えます。 |
| `onSearch`（必須） | 入力が止まったときに現在の検索語を受け取ります。消去された場合は空文字列です。 |
| `defaultValue` | 初期表示する検索語です。現在の検索条件を反映する場合に渡します。 |
| `debounceMs` | 通知までの待ち時間です。既定は `SEARCH_FIELD_DEBOUNCE_MS`。 |
| `clearLabel` | 消去ボタンのアクセシブルな名前です。既定は「検索語を消去」。 |

## 利用ケース

- 一覧の主導線で、打鍵しながら結果を絞り込みたい場合
- 取得が重く、打鍵ごとの呼び出しを待ち時間でまとめたい場合

JavaScript が無くても送信できる形が要る場合や、検索が主導線でない場合は `SearchFieldNative` を使います。候補集合から選ぶことがゴールの UI には `Command` を使います。

キーワード以外の条件（状態・期間・価格帯など）も並ぶ画面では、この検索欄を [`FilterBar`](../../navigation/filter-bar/README.md) の中へ置きます。`FilterBar` は検索欄を持たず、適用中の条件・件数・全解除の導線を束ねる外枠なので、排他ではなく入れ子の関係です。この component 単体で足りるのは、絞り込みがキーワードだけの場合です。

## 責務境界

入力の保持と待ち時間の制御のため hydration が必要な client island です。Server Component からは直接 render できません。

検索の実行、結果の取得、URL の組み立ては持ちません。入力が止まると `onSearch` を呼ぶだけで、router の操作も行いません。この分担は `Pagination` と同じで、`components` は URL を解釈しません。

見た目は `InputGroup` を合成して得ており、この component は独自の class を持ちません。検索アイコン付きの入力欄という見た目の owner は `InputGroup` 側にあります。

### 結果は Server Component で描画する

`onSearch` で受け取った検索語は、呼び出し元が `searchParams` へ載せ、結果は Server Component で描画します。結果まで client 側で取得すると、URL と表示が一致しなくなり、共有・履歴・戻る操作が壊れます。この component が client なのは入力の操作性のためだけであり、データの取得と描画を client へ移すためではありません。

### `onSearch` は安定した関数を渡す

`onSearch` の参照が変わるたびに待ち時間が測り直されます。render のたびに新しい関数を渡すと通知が発火しません。呼び出し元は `useCallback` などで参照を安定させます。

初期表示だけでは通知しません。`defaultValue` は現在の検索条件を映すためのものであり、mount と同時に検索し直す必要はないためです。

### `search` 要素と landmark

支援技術の landmark 一覧から到達できます。同じ画面に検索欄を複数置く場合は、`aria-label` で landmark を区別します。

`SearchFieldNative` と違い form ではないため、Enter による送信は行いません。

landmark は `role="search"` 属性ではなく HTML の `search` 要素で表します。browser は `search` 要素を `search` role へマップしますが、テストで使う `aria-query` 5.3.0 はまだこの要素を登録していないため、`getByRole("search")` からは引けません。これはツール側の未対応であり、実装を `div role="search"` へ戻して回避することはしません。テストは `data-slot` で要素を取得し、`search` 要素であることを直接検証します。

消去ボタンは検索語があるときだけ描画し、押すと入力を空にして focus を入力へ戻します。空文字列も `onSearch` で通知されるため、呼び出し元は検索条件の解除として扱えます。

## テスト

テストは `search` 要素の landmark に `type="search"` の入力を置くこと、初期表示では通知しないこと、入力が止まってから通知すること、入力が続く間は通知をまとめて最後の値だけを通知すること、待ち時間を変えられること、消去ボタンの出し分けと文言、消去時に入力が空になり focus が戻り空文字列が通知されること、a11y 自動検査を確認します。待ち時間の検証には fake timer を使い、a11y 自動検査だけは検査側が実時間を必要とするため実 timer に戻します。
