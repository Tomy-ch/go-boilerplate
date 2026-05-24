# GitHub Actions Workflows

English | [日本語](README.ja.md)

This directory contains GitHub Actions workflow definitions for CI/CD.

## Workflow List

### CI Checks (Pull Request)

|Workflow|File|Description|
|---|---|---|
|Golang Lint|`lint.yaml`|Run golangci-lint on Go code|
|Golang Test|`test.yaml`|Run Go tests with coverage reporting|
|Go Module Consistency|`tidy-check.yaml`|Verify go.mod / go.sum are tidied|
|SQL Lint|`sql-lint.yaml`|Run sqlfluff on migration / DML / seed SQL files|
|Migration Check|`migration-check.yaml`|Validate migration files (duplicates, gaps, up/down pairing)|
|Generated Go Artifacts|`gen-go-artifacts-check.yaml`|Verify generated Go code matches committed artifacts|
|Generated DB Artifacts|`gen-db-artifacts-check.yaml`|Verify generated sqlc code matches committed artifacts|
|Generated OpenAPI Artifacts|`gen-oapi-artifacts-check.yaml`|Verify OpenAPI bundle and docs match committed artifacts|
|Application Boot|`app-di-startup-check.yaml`|Verify application starts successfully with DB|

### Security (Pull Request)

|Workflow|File|Description|
|---|---|---|
|Code Security Scan|`code-ql.yaml`|CodeQL analysis for security vulnerabilities|
|Dependency Vulnerability Scan|`trivy-fs.yaml`|Trivy filesystem scan for OS/library vulnerabilities|
|Docker Image Scan|`image-scan.yaml`|Build image, generate SBOM, run Trivy scan|
|Go Vulnerability Analysis|`vulnerability-check.yaml`|govulncheck for actionable Go vulnerabilities|

### Deployment (Push)

|Workflow|File|Trigger|Description|
|---|---|---|---|
|Application Deployment|`deploy-app.yaml`|push to production/staging/develop|Build and push Docker images, run migration and deploy|
|Deploy Docs Portal|`deploy-docs.yaml`|push to production (docs changes)|Deploy documentation portal to GitHub Pages|

### Documentation (Push)

|Workflow|File|Trigger|Description|
|---|---|---|---|
|Auto-generate Docs PR|`auto-generate-docs.yaml`|push to release/* branches|Auto-generate OpenAPI docs, ER diagrams, portal docs|
