# stringkit

English | [日本語](README.ja.md)

String length validation functions based on rune count, with error message generation.

## Public API

|Function|Description|
|---|---|
|`RuneCount(s)`|Return UTF-8 rune count|
|`InRange(s, min, max)`|Check if length is within closed interval|
|`MaxOrLess(s, max)`|Check if length <= max|
|`MinOrMore(s, min)`|Check if length >= min|
|`StrictInRange(s, min, max)`|Check if length is within open interval|
|`LessThanMax(s, max)`|Check if length < max|
|`GreaterThanMin(s, min)`|Check if length > min|

Each function has a corresponding `ErrorMsg` function for generating validation error messages.

## Notes

Operates on UTF-8 rune count, not byte length.
