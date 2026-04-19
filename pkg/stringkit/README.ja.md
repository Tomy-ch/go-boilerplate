# stringkit

[English](README.md) | 日本語

文字列の長さ（ルーン数）に基づくバリデーション関数群とエラーメッセージ生成を提供します。

## 公開 API

|関数|説明|
|---|---|
|`RuneCount(s)`|UTF-8 ルーン数を返す|
|`InRange(s, min, max)`|長さが閉区間内か判定|
|`MaxOrLess(s, max)`|長さが最大値以下か判定|
|`MinOrMore(s, min)`|長さが最小値以上か判定|
|`StrictInRange(s, min, max)`|長さが開区間内か判定|
|`LessThanMax(s, max)`|長さが最大値未満か判定|
|`GreaterThanMin(s, min)`|長さが最小値超過か判定|

各関数に対応する `ErrorMsg` 関数があり、バリデーションエラーメッセージを生成できます。

## 注意点

バイト長ではなく UTF-8 ルーン数で動作します。
