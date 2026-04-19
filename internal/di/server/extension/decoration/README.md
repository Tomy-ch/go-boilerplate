# decoration

English | [日本語](README.ja.md)

`decoration` is a layer that provides **server startup visual/display features** (banner display / default port setting) via DI.

It groups "startup decoration" features that help developers immediately understand the status when the application starts.

## Modules

|Module|Type|Description|
|---|---|---|
|`BannerModule()`|Configurator|Control Echo banner visibility based on environment|
|`DefaultPortModule()`|Configurator|Control port number display based on environment|

## Notes

- Banner display is UI decoration — **must not depend on business logic (domain/usecase)**
- `DefaultPort` depends on `ApplicationConfig`, so be aware of environment differences
- Decoration is purely "startup assistance" and should not be mixed with essential middleware
