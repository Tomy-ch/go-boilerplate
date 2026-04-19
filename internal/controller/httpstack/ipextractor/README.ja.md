# ipextractor

[English](README.md) | 日本語

環境に応じたクライアント IP 抽出戦略を設定します。

## 公開 API

|関数|説明|
|---|---|
|`New(e, appCfg, secCfg)`|Echo インスタンスに IP 抽出を設定|
|`NewIPExtractor(appCfg, secCfg)`|本番では X-Forwarded-For + CIDR、開発では直接抽出の `echo.IPExtractor` を返す|
