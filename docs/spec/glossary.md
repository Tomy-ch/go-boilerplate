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

**この表が上であり、コードが下である。** 語の意味は業務が決め、モデルはそれを表現できるよう作られる。
両者が食い違ったとき、疑うのはコードの側である——ここの行を実装に合わせて書き換えるのは、
**業務が何を話しているかをコードに決めさせる**ことであり、それはこのページが存在する理由の否定になる。

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
| 商品 | 顧客が購入できる品目。売値と在庫を持ち、公開されている間だけ顧客の目に触れる | product / Product | `product.Product` | `Product` |
| 商品カテゴリ | 顧客が商品を探すときの分類。商品はちょうど 1 つに属する | product-category / ProductCategory | `category.Category` | `ProductCategory` |
| 商品ステータス | 商品を今どう取り扱っているかの区分 | product-status / ProductStatus | `status.Status` | `ProductStatus` |
| 公開中 | 顧客がその商品を見つけて購入できる状態 | product / Product | `product.IsPublished` | — |
| 在庫僅少 | 補充しなければ品切れが近い水準まで在庫が減っている状態 | product / Product | `product.Product.IsLowStock` | — |
| 購入 | 顧客が商品を買った事実。何をいくらで買ったかが確定している | purchase / Purchase | `purchase.Purchase` | `Purchase` |
| 購入明細 | 購入 1 件に含まれる商品ごとの行。単価は買った時点の値で確定し、後の値上げに追随しない | purchase / Purchase | `purchase.PurchaseDetail` | `PurchaseDetail` |
| 購入コード | 顧客との問い合わせでその購入を指し示すための符号 | purchase / Purchase | `purchase.Purchase.Code` | `code` |
| キャンセル済み | 購入が取り消され、履行されないことが確定した状態 | purchase / Purchase | `purchase.Purchase.IsCanceled` | — |
| ユーザー | このサービスで商品を購入する人 | user / User | `user.User` | `User` |
| 在籍 | ユーザーがこのサービスの利用を続けている状態 | user / User | `user.User.IsActive` | — |
| 都道府県 | 住所を広域で区分する単位 | prefecture / Prefecture | `prefecture.Prefecture` | `Prefecture` |
| 未処理 | 購入が成立した直後の、まだ何も進んでいない状態 | purchase / Purchase | `purchase.StatusUnprocessed` | — |
| 支払い済み | 購入の代金が支払われた状態 | purchase / Purchase | `purchase.StatusPaid` | — |
| 発送済み | 購入された商品が顧客へ向けて送り出された状態 | purchase / Purchase | `purchase.StatusShipped` | — |
| 配達済み | 購入された商品が顧客に届いた状態。ここから先へは進まない | purchase / Purchase | `purchase.StatusDelivered` | — |
| 購入完了 | 購入が果たされた状態。以後は取り消せない | purchase / Purchase | `purchase.StatusCompleted` | — |
| 管理者 | 一般の利用者には許されない操作を行える役割 | user / User | `user.RoleCodeAdmin` | — |
| 退会 | ユーザーがこのサービスの利用をやめること | user / User | `user.User.MarkAsDeleted` | — |
| 購入可能 | そのユーザーが今、購入してよい状態にあること | membership | `membership.EnsurePurchasable` | — |
| 退会可能 | そのユーザーが今、退会してよい状態にあること | membership | `membership.EnsureWithdrawable` | — |
| 在庫調整 | 商品の在庫数を増やす、または減らすこと | product / Product | `product.Product.AdjustStock` | — |
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
| `New` / `New<VO>` / `Reconstruct` | 構築の入口。業務が同じ行為を別の名前で呼ぶなら、その語のほうが行になる |
| `CanTransitionTo` / `IsTerminal` / `IsZero` | 状態機械の機構。**遷移そのものは業務の語だが、遷移可否の問い方は違う** |
| `EnsureVersion` | 同時更新の検出。業務は版を持たない |
| `Decimal` / `String` / `ToMinorUnit` | 金額の表現変換 |
| `Update` / `UpdateProfile` | 属性の書き換え一般。何が変わるかを言っていない |
| `StatusCode` / `Type` | 値の符号と種別の取り出し |

ここにある名前は昇格しうる。業務が使い始めたら上の表へ移す。**この一覧は判断の記録であって、
恒久的な除外ではない。**

## 語彙を改訂するとき

モデルを作っていると、業務が 1 語で呼んでいたものが実は 2 つだった、と分かることがある。**そのとき
先に直すのはこの表であって、コードではない。** 語を分け、それぞれの定義を書き、所有を決め、
そのあとでモデルがその区別を表現するよう変わる。

順序が逆になると——コードで型を 2 つに割ってから表を追認すると——**表はコードの索引に退化する。**
索引はコードと矛盾できず、矛盾できない語彙はモデルの誤りを指摘できない。このページの価値は
コードに反対できることの中にしかない。

改訂の入口は 3 つある。いずれも人間が行う。

- 実装中に見つかった区別（上記）
- [Watch list](#watch-list) に積まれた語が決着したとき
- [`/glossary`](../../.claude/skills/glossary/SKILL.md) が報告する新出語・orphan・同音異義

## Watch list

このページが捕まえるために在る 2 つの失敗。どちらも単一の feature からは見えない。

- **同音異義** — 1 つの識別子、2 つの feature で違う定義。識別子の衝突は機械で検出できるが、定義が
  実際に違うかは読み取りであり、改名するかどうかは判断である
- **同義** — 2 つの語、1 つの概念。機械では検出できない。2 つの行を読んだ人が「これは同じものだ」と
  気づいたときに表面化する。**この表が生成されるだけでなく読まれることを前提にしているのはそのため**

同音異義・同義は現在エントリなし。以下は**行にするかどうかが未決**の語である。

- **氏名（`User.FullName`）** — 姓名を組み立てた表示用の値。業務の語か、表示の都合かが決まっていない。
- **`Purchase.Pay` / `Ship` / `Deliver` / `Cancel`** — 行き先の状態は行になったが、**遷移という行為の側**を
  別の語として持つかは未決。業務が「発送する」を名詞の「発送済み」と区別して話すなら 2 行要る。

## What this document is not

- **データ辞書ではない。** 列・型・制約は feature ごとの spec と migration のもの。ここの行が述べるのは
  語の意味であって、保存のされ方ではない
- **規則の置き場ではない。** 「購入の金額がどう決まるか」は業務知識であり `docs/spec/<feature>/` に属する。
  このページが述べるのは「購入とは何か」である
- **生成物ではない。** `/glossary` は行を提案し欠落を報告するが、文言は人間のものであり、どの名前が
  勝つかという判断もすべて人間のものである
