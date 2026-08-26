# OpenAPI Parameter Quick Reference

English | [日本語](parameter-guide.ja.md)

## $ref Examples

<!-- sample-api:replace-begin -->
```yaml
# from openapi/paths/v1/users.yaml
parameters:
  - $ref: '../../components/parameters/pagination/PageParam.yaml'
  - $ref: '../../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../../components/parameters/search/KeywordParam.yaml'
```
<!-- sample-api:replace-with -->
<!-- = ```yaml -->
<!-- = # from openapi/paths/v1/<resource>.yaml -->
<!-- = parameters: -->
<!-- =   - $ref: '../../components/parameters/pagination/PageParam.yaml' -->
<!-- =   - $ref: '../../components/parameters/pagination/PerPageParam.yaml' -->
<!-- = ``` -->
<!-- sample-api:replace-end -->

The number of `../` segments follows the depth of the referencing file: two from
`paths/v1/*.yaml`, three from `paths/v1/<group>/*.yaml`, and so on.

## Naming Convention

<!-- sample-api:replace-begin -->
|Element|Convention|Example|
|---|---|---|
|Path name|slug format|`/v1/users/{userId}`|
|Directory|lowercase by concern|`pagination/`, `search/`, `user/`|
|File name|PascalCase + Param|`PageParam.yaml`, `UserIdParam.yaml`|
<!-- sample-api:replace-with -->
<!-- = |Element|Convention|Example| -->
<!-- = |---|---|---| -->
<!-- = |Path name|slug format|`/v1/<resource>/{<resource>Id}`| -->
<!-- = |Directory|lowercase by concern|`pagination/`, `idempotency/`| -->
<!-- = |File name|PascalCase + Param|`PageParam.yaml`, `IdempotencyKeyParam.yaml`| -->
<!-- sample-api:replace-end -->

## Detailed Guide

See [components/parameters/README.md](components/parameters/README.md) for full design policy and available parameters.
