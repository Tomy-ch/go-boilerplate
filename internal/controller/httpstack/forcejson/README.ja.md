# forcejson

[English](README.md) | 日本語

レスポンスの Content-Type が未設定または `text/html` の場合に `application/json` へ強制します。

## 役割

本 API の契約は JSON のみを扱いますが、個々のハンドラやエラー経路はレスポンスの Content-Type を未設定のまま、あるいは `text/html` のまま残すことがあります。単一のミドルウェアがそれらのケースを `application/json` へ正規化し（`text/csv` など明示済みの Content-Type はそのまま維持）、各ハンドラが個別に設定しなくても JSON レスポンスが一貫した Content-Type を表明します。
