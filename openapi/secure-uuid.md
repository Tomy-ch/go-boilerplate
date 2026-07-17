# UUID Exposure Security Evaluation

English | [日本語](secure-uuid.ja.md)

This project adopts an API design that exposes UUIDs (e.g., `userId`) externally. The following multi-layered defenses eliminate or mitigate security concerns.

## Prerequisites and Design Philosophy

- Domain Entity structs have no `json` tags — they cannot be directly output via API
- API responses are composed by converting DTO → oapi-codegen types, with output fields explicitly controlled
- All non-GET operations require JWT authentication
  - GET requests also require JWT when accessing private user information

## Evaluation of UUID Exposure

|Scenario|Conclusion|Notes|
|---|---|---|
|Exposing UUID externally|Safe|JWT authentication and authorization are required|
|GET for sensitive information|Safe|JWT required; inaccessible to non-owners|
|IDOR attacks via UUID|Structurally impossible|Authorization layer exists|
|Response field leakage|Controlled|DTO layer explicitly selects fields|

## Security Defense Layers

1. **Structural defense** — Entities without `json` tags eliminate the risk of accidental output

2. **Explicit control via DTO** — Unnecessary internal information is excluded from responses

3. **JWT authentication + sub authorization** — Request authenticity is guaranteed; third-party access is blocked

4. **Responsibility separation by API classification** — Admin and general APIs have separated responsibilities, restricting internal data handling

## Conclusion

In this architecture, exposing UUIDs as public identifiers is safe due to multi-layered defenses through authentication, authorization, and output control, making it a **practical and secure operational design**.
