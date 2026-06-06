# GitHub Actions Workflows

English | [日本語](README.ja.md)

This directory contains GitHub Actions workflow definitions for CI/CD. Workflows are grouped by purpose: pull-request gates (lint / test / security scans), push-triggered deployments, and documentation regeneration on release branches.

## Trigger Strategy

| Group | When it runs | What it does |
| --- | --- | --- |
| CI Checks | every pull request | Block merge if lint / test / generated-artifact consistency fails |
| Security | every PR (and push to default) | Surface vulnerabilities in code, dependencies, images, and Go runtime |
| Deployment | push to `production` / `staging` / `develop` | Build artifacts, run migration, deploy app or docs portal |
| Documentation | push to `release/*` | Regenerate OpenAPI / ER / portal docs and open an auto-sync PR |

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
|Dependency Vulnerability Scan|`trivy-fs.yaml`|Trivy filesystem scan for library vulnerabilities (developer-facing)|
|Release Dependency Vulnerability Scan|`trivy-release-gate.yaml`|Trivy filesystem scan on PRs into develop/staging/production|
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

## Notes

- `auto-generate-docs.yaml` opens an auto-PR whose branch is named `auto/docs-update/<base>-<run-id>`; the workflow skips itself on that branch to avoid recursion.
- All deployment workflows require their target branch (`production` / `staging` / `develop`) to be branch-protected; merges must flow through PR review.
- Security scans run on every PR; if a high-severity CodeQL or Trivy finding appears, the corresponding branch-protection rule blocks merge.
- The `Detect changes` step in `auto-generate-docs.yaml` excludes coverage HTML and SchemaSpy timestamp churn so cosmetic regenerations do not open noise PRs.
