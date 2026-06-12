# testecho

English | [日本語](README.ja.md)

Builder-pattern HTTP test client for Echo handler tests.

## Public API

|Method|Description|
|---|---|
|`NewEchoTestClient(t, e)`|Create test client|
|`WithAppErrorHandler()`|Install the production error handler (overwrites the Echo's `HTTPErrorHandler`)|
|`Method(m)`|Set HTTP method|
|`RoutePattern(p)`|Set route pattern (e.g. `/users/:id`)|
|`RequestURL(u)`|Set actual request URL|
|`JSONBody(v)`|Set JSON request body|
|`RawBody(r, contentType)`|Set raw request body|
|`Header(k, v)`|Set request header|
|`AuthBearer(token)`|Set Bearer token|
|`PathParams(params)`|Set path parameters|
|`QueryParams(params)`|Set query parameters|
|`Build()`|Return Request / ResponseRecorder / echo.Context|
|`Serve()`|Send request to Echo and return ResponseRecorder|
