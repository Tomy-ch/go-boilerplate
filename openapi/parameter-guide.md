# OpenAPI Parameter Quick Reference

English | [日本語](parameter-guide.ja.md)

## $ref Examples

```yaml
parameters:
  - $ref: '../components/parameters/pagination/PageParam.yaml'
  - $ref: '../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../components/parameters/search/KeywordParam.yaml'
  - $ref: '../components/parameters/user/UserIdParam.yaml'
```

## Naming Convention

|Element|Convention|Example|
|---|---|---|
|Path name|slug format|`/v1/users/{userId}`|
|Directory|lowercase by concern|`pagination/`, `search/`, `user/`|
|File name|PascalCase + Param|`PageParam.yaml`, `UserIdParam.yaml`|

## Detailed Guide

See [components/parameters/README.md](components/parameters/README.md) for full design policy and available parameters.
