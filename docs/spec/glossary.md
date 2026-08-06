# Glossary — Business Vocabulary Spec

> このシステムの**業務語彙**（Ubiquitous Language）の正本。1 つの語に 1 つの意味、1 つの名前を、
> 会話と spec とコードで共有するための統括 spec。
> 生成の支援は [`/glossary`](../../.claude/skills/glossary/SKILL.md) が行う。**正名は決めない。**

## Overview

業務語彙はここ、`docs/spec/` に住む。`README.md` や `docs/adr/**` には住まない。あちらが述べるのは
実装の構造と、その構造を選んだ意思決定である。**層 README へ育ってしまった業務語は、家を出た語である。**
同じ語がどこか別の場所で定義し直され、それに誰も気づかない。

feature ごとの spec だけでは足りず、その理由がこのページの存在理由そのものである。**Evans が最も
警戒すること——1 つの語が 2 つの意味を持つ、2 つの語が 1 つの意味を指す——は feature を跨いだときにしか
起きない。** feature 単位の spec しか無いリポジトリが持っているのは方言の集合であって、言語ではない。

このプロジェクトは既に 2 度踏んでいる。どちらも検査ではなく偶然の発見だった。`value object` が
`internal/domain` と `pkg/` で違う意味を持っていた件と、`kernel` が 3 つの意味を持っていた件である。

## How a term earns a row

**業務を知っている人が「これは業務の語だ」と認める**とき、その語は行を得る。これはドメイン層が
「この条件は基準か」を判定するときと同じ問いであり、同じであることは意図している。

構造や機構の名前は、コード中でどれだけ目立っていても該当しない。`Attributes`・`Repository`・
`ListParams`・`FeedCursor` が述べているのはこのシステムの作りであって、業務が話していることではない。
それらは [Mechanism vocabulary](#mechanism-vocabulary) に記録し、生成器が繰り返し提案しないようにする。

| 列 | 何を書くか |
| --- | --- |
| 用語 | 業務が言うとおりの語 |
| 定義 | ドメインエキスパートが受け入れる 1 文。実装の語彙を使わずに書く |
| 所有 | 意味を所有する feature と集約。**ちょうど 1 つ**。2 つあるならその衝突こそが findings |
| コードシンボル | それを担う公開識別子（`package.Type`） |
| 公開名 | 公開契約（OpenAPI schema / field）での名前。表に出ない語は空 |

公開名が空なのは普通である。すべての業務語が API 境界を越えるわけではない。**コードシンボルが空なのは
findings** — 業務が使っていてモデルが使っていない語は、実装漏れか、そもそも語ではないかのどちらか。

## Terms

<!-- sample-api:begin -->
以下の行はサンプル機能一式に由来し、**サンプルを撤去すると消える**。残るのはこのページと、ここに書かれた
規則のほうである。言語はこのテンプレートを採用する側のものである。

| 用語 | 定義 | 所有 | コードシンボル | 公開名 |
| --- | --- | --- | --- | --- |
| 商品 | 販売の対象となる品目。価格・在庫・公開状態を持つ | product / Product | `product.Product` | `Product` |
| 商品カテゴリ | 商品を分類する区分。参照マスタ | product-category / ProductCategory | `category.Category` | `ProductCategory` |
| 商品ステータス | 商品の取扱状態を表す区分。参照マスタ | product-status / ProductStatus | `status.Status` | `ProductStatus` |
| 公開中 | 顧客に見えている状態。公開日時が設定されていることを指す | product / Product | `product.IsPublished` | — |
| 在庫僅少 | 補充を要する水準まで在庫が減っている状態 | product / Product | `product.Product.IsLowStock` | — |
| 購入 | 顧客が商品を購入した事実。明細・金額・状態を持つ | purchase / Purchase | `purchase.Purchase` | `Purchase` |
| 購入明細 | 購入 1 件に含まれる商品ごとの行。購入時点の単価スナップショットを持つ | purchase / Purchase | `purchase.PurchaseDetail` | `PurchaseDetail` |
| 購入コード | 購入を業務上識別する符号 | purchase / Purchase | `purchase.Purchase.Code` | `code` |
| キャンセル済み | 購入が取り消された状態 | purchase / Purchase | `purchase.Purchase.IsCanceled` | — |
| 発送可能 | 購入が発送してよい状態。支払いを終え、まだ発送していないことを指す | purchase / Purchase | `purchase.Purchase.IsShippable` | — |
| まとめ発送 | 発送待ちの購入のうち、1 便にまとめて発送してよい組 | purchase / Purchase | `dispatch.GroupForDispatch` | — |
| ユーザー | このシステムを利用する人 | user / User | `user.User` | `User` |
| 在籍 | ユーザーが退会していない状態 | user / User | `user.User.IsActive` | — |
| 都道府県 | 住所の広域区分。参照マスタ | prefecture / Prefecture | `prefecture.Prefecture` | `Prefecture` |
<!-- sample-api:end -->

## Mechanism vocabulary

以下の公開名は業務語**ではない**。生成器が提案し続けないため、そして読者がコード上の目立ちを業務上の
重要さと取り違えないために記録する。

| 名前 | 業務語でない理由 |
| --- | --- |
| `Attributes` / `Profile` / `PurchaseDetailAttributes` | 位置引数の取り違えを防ぐための引数の束 |
| `Repository` / `LockRepository` / `RoleRepository` | 永続化の抽象。業務はリポジトリを持たない |
| `ListParams` / `ListFeedParams` / `Counts` / `FeedCursor` / `FeedItem` | 読み取り経路の問い合わせとページングの形 |
| `DetailInput` / `LockedProduct` | コンストラクタの入力であって、業務が名前を付けるものではない |
| `Event` / `EventType` | 事実の封筒。事実の**名前**は語だが、封筒は語ではない |
| `StatusRef` / `CategoryRef` | 同一性に表示用の属性を添えた集約横断参照 |

ここにある名前は昇格しうる。業務が使い始めたら上の表へ移す。**この一覧は判断の記録であって、
恒久的な除外ではない。**

## Watch list

このページが捕まえるために在る 2 つの失敗。どちらも単一の feature からは見えない。

- **同音異義** — 1 つの識別子、2 つの feature で違う定義。識別子の衝突は機械で検出できるが、定義が
  実際に違うかは読み取りであり、改名するかどうかは判断である
- **同義** — 2 つの語、1 つの概念。機械では検出できない。2 つの行を読んだ人が「これは同じものだ」と
  気づいたときに表面化する。**この表が生成されるだけでなく読まれることを前提にしているのはそのため**

現在エントリなし。

## What this document is not

- **データ辞書ではない。** 列・型・制約は feature ごとの spec と migration のもの。ここの行が述べるのは
  語の意味であって、保存のされ方ではない
- **規則の置き場ではない。** 「購入の金額がどう決まるか」は業務知識であり `docs/spec/<feature>/` に属する。
  このページが述べるのは「購入とは何か」である
- **生成物ではない。** `/glossary` は行を提案し欠落を報告するが、文言は人間のものであり、どの名前が
  勝つかという判断もすべて人間のものである
