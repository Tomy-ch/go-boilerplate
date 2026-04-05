# `testauth` パッケージ

`testauth` パッケージは、通常 MW で設定される認証情報をテストコード内で簡単に設定するためのユーティリティを提供します。

## 使い方

`MakeAvailableAuthn` 関数を使用して、テストコンテキストに認証情報を設定します。

```go
  ctx := context.Background()
  ctx = testauth.MakeAvailableAuthn(t, ctx, userID.String()) // ここで認証情報を設定
  ctrl := gomock.NewController(t)
```

この関数は、指定されたユーザーIDを持つ認証情報をコンテキストに追加し、コントローラーで認証情報を利用できるようにします。
