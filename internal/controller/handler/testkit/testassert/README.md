# testassert

English | [日本語](README.ja.md)

Assertion helpers for Controller layer tests.

## Public API

|Function|Description|
|---|---|
|`AssertJSONEqual[T](t, expectedCode, expectedResponse, actualResponse)`|Verify HTTP status code and JSON body|
|`AssertEchoRouterMethods(t, expectedMethods, actualRoute)`|Verify registered route HTTP methods|
|`AssertEchoRouterPath(t, expectedPath, actualRoute)`|Verify registered route path|
