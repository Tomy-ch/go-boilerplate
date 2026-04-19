# shutdowner DI ラッパー

[English](README.md) | 日本語

`go.uber.org/fx` の `fx.Shutdowner` を抽象化した `Shutdowner` インターフェースを提供するパッケージです。

DI コンテナから取得した `fx.Shutdowner` をラップし、アプリケーションコードやテストから利用しやすくします。

## 公開 API

|型 / 関数|説明|
|---|---|
|`Shutdowner`|`Shutdown(...fx.ShutdownOption) error` を抽象化したインターフェース|
|`NewShutdowner(sd fx.Shutdowner)`|`fx.Shutdowner` をラップした具象インスタンスを生成|
|`Module()`|`Shutdowner` を DI コンテナに登録する `fx.Module` を返す|

## なぜ抽象化するのか

- `fx.Shutdowner` を直接使うと、アプリケーションコードが fx フレームワークに結合する
- `Shutdowner` インターフェースにより、テストでのモック注入が容易になる
- fx 依存を DI 層に閉じ込められる

## 注意点

- ラッパーは極めて薄い実装 — `fx.Shutdowner` を保持して `Shutdown` を委譲するだけ
- `Shutdown` はプロセス停止やクリーンアップをトリガーするため、呼び出し側で副作用に注意
- `mock/` には `mockgen` による自動生成モックが格納されている
