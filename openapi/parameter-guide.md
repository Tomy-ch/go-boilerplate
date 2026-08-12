# OpenAPI Parameter Quick Reference

English | [日本語](parameter-guide.ja.md)

## $ref Examples

```yaml
# from openapi/paths/v1/users.yaml
parameters:
  - $ref: '../../components/parameters/pagination/PageParam.yaml'
  - $ref: '../../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../../components/parameters/search/KeywordParam.yaml'
```

The number of `../` segments follows the depth of the referencing file: two from
`paths/v1/*.yaml`, three from `paths/v1/<group>/*.yaml`, and so on.

## Naming Convention

|Element|Convention|Example|
|---|---|---|
|Path name|slug format|`/v1/users/{userId}`|
|Directory|lowercase by concern|`pagination/`, `search/`, `user/`|
|File name|PascalCase + Param|`PageParam.yaml`, `UserIdParam.yaml`|

## Detailed Guide

See [components/parameters/README.md](components/parameters/README.md) for full design policy and available parameters.
