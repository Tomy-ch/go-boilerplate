# safecast

English | [日本語](README.ja.md)

Provides safe type conversion with overflow detection.

## Public API

|Function / Variable|Description|
|---|---|
|`UintToInt(x uint) (int, error)`|Safe conversion from `uint` to `int`|
|`ErrOverflow`|Error returned when overflow occurs|

## Notes

Returns `ErrOverflow` when the value exceeds `math.MaxInt`.
