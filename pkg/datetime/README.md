# datetime

English | [日本語](README.ja.md)

Provides date/time parsing utilities supporting multiple formats with timezone awareness.

## Public API

|Function|Description|
|---|---|
|`ParseRFC3339(s)`|Parse RFC3339 format|
|`ParseRFC3339UTC(s)`|Parse RFC3339 format (UTC)|
|`ParseRFC3339Nano(s)`|Parse RFC3339Nano format|
|`ParseISO8601(s)`|Parse ISO8601 format|
|`ParseDateTime(s)`|Parse standard datetime format|
|`ParseDateOnly(s)`|Parse date-only format|
|`ParseCustomLayout(layout, s)`|Parse with arbitrary layout|

All functions have `InLocation` variants for parsing with a specified timezone.

## Wraps

Standard library `time` package.
