# paging

[English](README.md) | 日本語

1ベースの page/perPage パラメータを limit/offset に変換するページネーション共通構造体とロジックを提供します。

## 公開 API

|関数 / メソッド|説明|
|---|---|
|`NewPagingFrom1Based(page, perPage *int)`|1ベースのページ番号と件数から `Paging` を生成|
|`Limit()` / `Limit32()`|取得上限を返す（int / int32）|
|`Offset()` / `Offset32()`|オフセットを返す（int / int32）|

## 定数

|定数|値|説明|
|---|---|---|
|`defaultPerPage`|50|デフォルトの1ページあたり件数|
|`maxPerPage`|200|最大の1ページあたり件数|
|`minPage`|1|最小ページ番号|
|`maxPage`|10,000|最大ページ番号|

## 挙動

- `perPage` ≤ 0 または nil → `defaultPerPage`（50）を使用
- `perPage` > `maxPerPage` → `maxPerPage`（200）にクランプ
- `page` ≤ 0 または nil → `minPage`（1）を使用
- `page` > `maxPage` → `apperror.ErrInvalidArgument` を返す
- `Limit32()` / `Offset32()` は安全な int32 変換のためクランプ

## 使用例

```go
pg, err := paging.NewPagingFrom1Based(ptr.To(2), ptr.To(20))
// pg.Limit() == 20, pg.Offset() == 20
```
