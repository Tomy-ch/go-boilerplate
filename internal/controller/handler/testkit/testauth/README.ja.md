# testauth

[English](README.md) | 日本語

通常 MW で設定される認証情報を、テストコード内で設定するためのユーティリティを提供します。

## 使い方

`MakeAvailableAuthn` 関数を使用して、テストコンテキストに認証情報を設定します。

```go
  ctx := context.Background()
  ctx = testauth.MakeAvailableAuthn(ctx, t, userID.String()) // ここで認証情報を設定
  ctrl := gomock.NewController(t)
```

この関数は、指定されたユーザーIDを持つ認証情報をコンテキストに追加し、テスト対象のコントローラーで利用できるようにします。

subject は内部 UserID を兼ねます。UUID として解釈できる場合は返される `Authn` の UserID が解決済みになり、そうでない場合は未解決のままです（「認証済みだが内部ユーザー未解決」の経路はこれで作ります）。ゼロ値 UUID の subject は `auth.Authn` が解決を拒否する（`ErrUserIDZero`）ため、テストが失敗します。
