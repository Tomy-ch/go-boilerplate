# outbound

English | [日本語](README.ja.md)

`outbound` is a directory that groups **DI middleware modules for extending HTTP response (output) processing**.

It handles **response transformation / error handling / output format enforcement / panic recovery** after request processing.

## Role

This package isolates the **response (output) side** of the composable server pipeline — the mirror of the request-side `inbound` group. Bundling response transformation, error mapping, output-format enforcement, and panic recovery into a single DI unit lets the output behavior be prioritized and evolved independently of the rest of the server, while keeping this controller-layer plumbing from leaking into the domain or usecase layers.

## Modules

|Module|Type|Description|
|---|---|---|
|`RecoveryModule()`|Use|Catch panics, log with stack trace, return 500|
|`ErrorHandlerModule()`|Configurator|Unified error-to-HTTP-response mapping with logging|
|`ForceJSONModule()`|Use|Force response Content-Type to JSON|

## Notes

- Priority follows `extension.UseMiddleware` rules — adjusted to avoid order conflicts with other middleware
- Recovery should be **one of the first middleware to execute**
- ErrorHandler replaces Echo's `HTTPErrorHandler` and is provided as ServeCfg
- Outbound middleware belongs to the controller layer — **must not depend on domain/usecase**
- To add response processing, add new outbound middleware to this directory
