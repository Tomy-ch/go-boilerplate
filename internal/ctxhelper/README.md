# ctxhelper

このパッケージは、context.ContextとEchoフレームワークのコンテキストを操作するためのヘルパー関数を提供します。

## 実装方法

このパッケージでは、同じ実装が多く作成することになるため、`make gen-ctxkey`を実行してコード生成を行います。

## 使用方法

`make gen-ctxkey`を実行すると、以下のようなコードが生成されます。

実行する際には、nameとtypeを指定する必要があります。

### 実行例

```bash
% make gen-ctxkey name=UserID type=string
✅ Generated: internal/controller/ctxhelper/user_id_ctx.go
✅ Generated: internal/controller/ctxhelper/user_id_ctx_test.go
%
```
