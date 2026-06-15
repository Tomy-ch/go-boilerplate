# datetime

[English](README.md) | 日本語

複数の日時フォーマットに対応し、タイムゾーンを考慮したパースユーティリティを提供します。

## 公開 API

|関数|説明|
|---|---|
|`ParseRFC3339(s)`|RFC3339 形式のパース|
|`ParseRFC3339UTC(s)`|RFC3339 形式のパース（UTC）|
|`ParseRFC3339Nano(s)`|RFC3339Nano 形式のパース|
|`ParseISO8601(s)`|ISO8601 形式のパース|
|`ParseDateTime(s)`|標準 datetime 形式のパース|
|`ParseDateOnly(s)`|日付のみのパース|
|`ParseCustomLayout(layout, s)`|任意のレイアウトによるパース|

すべての関数に `ToLocation` バリアント（例: `ParseRFC3339ToLocation`）があり、タイムゾーンを指定したパースが可能です。

## ラップ対象

標準ライブラリ `time` パッケージ
