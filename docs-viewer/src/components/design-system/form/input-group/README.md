# InputGroup

## 用途

単位記号・アイコン・補助操作を入力欄と一続きの枠に収め、入力欄の外へ並べた場合よりも「どの入力に属する情報か」を明確にします。

## 役割と公開 component

| Component | 役割 |
| --- | --- |
| `InputGroup` | 入力欄と addon を一続きの枠として囲む外枠です。枠線・角丸・focus 表示・invalid 表示・disabled 表示を担います。 |
| `InputGroupAddon` | 枠内で入力欄の前後・上下へ置く領域です。`align` で `inline-start` / `inline-end` / `block-start` / `block-end` を選びます。 |
| `InputGroupText` | addon 内へ単位・記号・短い説明を置く文字列です。 |
| `InputGroupButton` | 枠内へ収まる寸法へ調整した補助操作ボタンです。`xs` / `sm` / `icon-xs` / `icon-sm` の大きさを持ちます。 |
| `InputGroupInput` | 枠を外枠へ委ねた単一行の入力欄です。 |
| `InputGroupTextarea` | 枠を外枠へ委ねた複数行の入力欄です。 |

`align` / `size` に渡せる値の集合は、`input-group.definition.ts` が `INPUT_GROUP_ADDON_ALIGN` / `INPUT_GROUP_BUTTON_SIZE` として公開します。

## 利用ケース

- 数量・割合など、単位を伴う値の入力欄へ単位記号を添える場合
- キーワード入力の先頭へ検索アイコンを、末尾へ実行・消去の操作を収める場合
- 複数行の入力欄の上下へ、入力の説明や補助操作の列を積む場合

入力欄の外に置いても意味が通る補足には `Field` の `FieldDescription` を使います。入力欄そのものの項目名は `Label` / `Field` が与えます。

**入力を促す説明は `placeholder` に置きます。** addon に置くと入力領域と同じ枠の中に文字列が並ぶため、入力すれば消えるものだと読まれます。addon に置くのは、入力しても消えない情報（単位・記法の案内・補助操作）だけです。

## 責務境界

SSR first の選定では `△` に当たります。既定は `Input` / `Textarea` と `Label` / `Field` を Server Component として組み立てる構成であり、記号や操作を**入力欄の枠の内側**へ収める必要が確定した場合にこの component を選びます。`InputGroupAddon` は押された位置から枠内の control へ focus を委譲するため hydration が必要で、Server Component からは直接 render できません。addon に置く内容自体に client runtime が要らない場合は、Server Component で組み立てた要素を `children` として渡します。

値の state、検証、送信、エラー文言は持ちません。`aria-invalid` を受け取って外枠の表示を変えるだけで、invalid かどうかの判断は feature が行います。

**block 方向（`block-start` / `block-end`）の addon には区切り線を既定で引きます。** 入力欄の上下へ積まれた別の行であり、線が無いと入力領域と地続きに見えます。inline 方向（`inline-start` / `inline-end`）は入力欄と同じ行に収まるため引きません。線を消す場合は addon へ `border-b-0` / `border-t-0` を渡します。

**枠内の control は自分の focus 表示を持ちません。** focus 表示は `outline` で描かれるため、`InputGroupInput` / `InputGroupTextarea` は `outline` 側で打ち消します。`ring` を打ち消しても消えず、外枠の輪と二重になります。

外枠は focus・invalid・disabled の三つを control の状態から導出します。枠線は control の `disabled` に反応して控えめな色へ落ちるため、呼び出し元の追加指定は要りません。addon も同時に減光したい場合だけ、外枠へ `data-disabled="true"` を渡します。

`InputGroupInput` の枠は入力内容では伸縮しません。focus と invalid の ring はどちらも影として描かれ、枠線は太さを変えずに色だけが変わるため、状態が変わっても周囲のレイアウトは動きません。invalid でレイアウトが動くのは、feature がエラー文言を追加したときだけで、伸びる量は文言そのものの高さです。

`InputGroupTextarea` だけは内容に応じて高さが伸び、外枠もそれに追随します。table の cell のように行の高さが揃っている場所へ置くと周囲が押し下げられるため、その場合は単一行の `InputGroupInput` を選びます。

外枠と addon は `role="group"` を持ちます。いずれも名前を持たない group であり、入力欄のアクセシブルな名前は `Label` の `htmlFor` か `aria-label` で別に与えます。addon の記号・アイコンは装飾であり、control の名前にも説明にもなりません。`InputGroupAddon` 自体は focus を受け取らず、keyboard 利用者は control へ直接 tab で到達します。addon へ操作を置く場合は `InputGroupButton` を使い、アイコンだけのときは `aria-label` で名前を与えます。

`InputGroupButton` の既定は `type="button"` で、form の中に置いても送信しません。検索実行など submit させたい場合だけ `type="submit"` を指定します。

外部の interaction library は使わず、variant 定義に `class-variance-authority` を使うだけです。addon から control への focus 委譲は、`data-slot="input-group-control"` を目印にした DOM 走査で行います。

## テスト

テストは group の構造、`Label` によるアクセシブルな名前と `aria-describedby` による説明、`align` の反映、addon から単一行・複数行それぞれの control への focus 委譲、addon 内 button を押したときに focus を奪わないこと、control を持たない枠と枠の外へ置いた addon で何も起きないこと、button の既定が form を送信しないこと、`type="submit"` のときに `name` と値が送信されること、native 属性と `disabled` / `aria-invalid` の伝播、a11y 自動検査を確認します。
