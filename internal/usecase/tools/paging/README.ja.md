# paging

[English](README.md) | 日本語

ページネーションの共通値オブジェクトを提供します。2つの戦略を**いずれも妥当なアプリケーションポリシー**として提供します（ページネーションは domain のルールではなく usecase 層の関心事）。

- **オフセット方式（`Page`）** — 1ベースの page/perPage を limit/offset に変換。単純で任意ページへのランダムアクセスが可能だが、深いページで `OFFSET` のスキャンが増えて劣化する。
- **カーソル方式 / keyset（`Cursor`）** — 不透明カーソル（直前ページ末尾行のソートキー）を受け取り、`WHERE (sort_keys) < (:cursor)` クエリに用いる。深いページでも安定して高速で、大規模データや無限スクロールに推奨。本パッケージは**輸送（エンコード/デコード）・検証・件数ポリシー**のみを担い、キーを型付きのソート列へ解釈し直す処理（例: RFC3339 → time、UUID 文字列 → uuid）は**クエリ層**の責務。

本パッケージは**件数ポリシーそのもの**（`Limit` / `LimitPolicy`）も担います。これにより、ページネーションを一切持たない読み取り — ランキングや在庫僅少カードのような top-N 一覧 — も「未指定なら既定値、上限超過ならクランプ」という同じ規約を再実装せずに共有できます。`Page` / `Cursor` は本パッケージ自身の `defaultPerPage` / `maxPerPage` を与えて `Limit` の上に構築されており、top-N の呼び出し元は自前の `LimitPolicy` を渡します（フィールド名を持つため、既定値と上限を呼び出し側で取り違えることがありません）。

## 定数

|定数|値|説明|
|---|---|---|
|`defaultPerPage`|50|デフォルトの1ページあたり件数（`Page` / `Cursor`）|
|`maxPerPage`|200|最大の1ページあたり件数（`Page` / `Cursor`）|
|`minPage`|1|最小ページ番号|
|`maxPage`|10,000|最大ページ番号|

## 挙動

**件数（`Limit`）**

- `first` ≤ 0 または nil → `policy.Default` を使用
- `first` > `policy.Max` → `policy.Max` にクランプ
- `Value32()` は安全な int32 変換のため `math.MaxInt32` でクランプ

**オフセット方式（`Page`）**

- `perPage` ≤ 0 または nil → `defaultPerPage`（50）を使用
- `perPage` > `maxPerPage` → `maxPerPage`（200）にクランプ
- `page` ≤ 0 または nil → `minPage`（1）を使用
- `page` > `maxPage` → `apperror.ErrInvalidArgument` を返す
- `Limit32()` / `Offset32()` は安全な int32 変換のためクランプ

**カーソル方式（`Cursor`）**

- `first` ≤ 0 または nil → `defaultPerPage`（50）を使用
- `first` > `maxPerPage` → `maxPerPage`（200）にクランプ
- `after` が nil または空 → 先頭ページ（`HasCursor()` は `false`、`Keys()` は空）
- `after` が不正（base64 / JSON 不正、または空キーセット）→ `apperror.ErrInvalidArgument` を返す
- keyset は**ページ番号の上限を持たない** — これがカーソルページネーションの本質なので、`maxPage` 相当のエラーは存在しない
- カーソル文字列の形式は**不透明**（`base64url(JSON 文字列配列)`）。クライアント側はブラックボックスとして扱う。

## 使用例

### top-N（ページネーションなし）

```go
var lowStockLimitPolicy = paging.LimitPolicy{Default: 20, Max: 100}

limit := paging.NewLimit(req.Limit, lowStockLimitPolicy)
// req.Limit が nil なら limit.Value() == 20。limit.Value32() を SQL の LIMIT に渡す。
```

### オフセット方式

```go
pg, err := paging.NewPageFrom1Based(ptr.To(2), ptr.To(20))
// pg.Limit() == 20, pg.Offset() == 20
```

### カーソル方式

```go
// 1) リクエストを解釈（先頭ページは after == nil）。
cur, err := paging.NewCursor(req.After, req.First)

// 2) クエリ層: limit+1 件取得する。cur.HasCursor() の場合は cur.Keys() を
//    用いて keyset 条件を適用する（文字列を型付き列へ解釈）:
//      WHERE (created_at, id) < ($1, $2) ORDER BY created_at DESC, id DESC
//      LIMIT cur.Limit32() + 1
//    1件多く取得することで「次ページの有無」を判定できる。

// 3) 「表示した末尾行」のソートキーから次カーソルを生成:
next := paging.EncodeCursor(last.CreatedAt.Format(time.RFC3339Nano), last.ID.String())
// 余分な1件が存在した場合のみ next を返し、無ければ "" （末尾）。
```

> ソートキーはタプルとして**一意かつ全順序**でなければならない（主キー等のタイブレーカーを末尾に付ける）。さもないとページ間で行のスキップや重複が起きる。

## テストカバレッジ例外

以下の未被覆分岐は**失敗しない防御分岐**として、ほぼ 100% の被覆期待の対象外とする。これを
塗るための contrived テストや追加実装は行わない:

- `cursor.go` `EncodeCursor` — `json.Marshal(keys)` のエラー return。`keys` は `[]string`
  で常に marshal 可能なため、エラー経路は到達不能。

**ガバナンス:** カバレッジ例外は**任意に追加しない**。新規エントリはアーキテクト等の適切な
承認者の承認を要する。
