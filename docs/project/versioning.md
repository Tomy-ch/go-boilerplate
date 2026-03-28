# Versioning Policy

This project adopts **Semantic Versioning (SemVer)**.

- MAJOR
- MINOR
- PATCH

## Version Definitions

- **MAJOR**  
  Breaking changes (changes that break backward compatibility)

- **MINOR**  
  Feature additions that maintain backward compatibility

- **PATCH**  
  Bug fixes and non-breaking improvements

## Release Branch Strategy

This project adopts a **release-centric branching model**.

- Feature development branches off from the latest `release/*` branch
- Changes are reflected to `develop`, `staging`, and `production` only via release branches
- Direct commits to protected branches are prohibited

## Release Procedure

### Tagging

Issue tags using the following commands:

```bash
make release-major-tag
make release-minor-tag
make release-patch-tag
```

Manual creation of tags is prohibited.

### Creating the Next Release Branch

```bash
make release-major-branch
make release-minor-branch
make release-patch-branch
```

### Hotfix Procedure

When an urgent fix is required:

- Create a `hotfix/*` branch from the `production` branch using `make hotfix-patch-branch`
- Apply the fix on the `hotfix/*` branch and merge it into `production`
- From the updated `production`, create the next `release/*` branch using `make release-patch-branch`, and merge into `develop` / `staging` / `production` via that `release/*` branch
- When the `release/*` branch is merged into `production`, issue a PATCH version tag (`make release-patch-tag`)

## Rules for Breaking Changes

- Breaking changes are allowed only in MAJOR versions
- API contract changes must follow the OpenAPI-first policy
- If OpenAPI changes are involved, code generation must always be executed

## Principles

- Direct editing of version numbers is prohibited
- Tags must be issued only via predefined make commands
- Follow branch protection rules
- Strictly adhere to semantic versioning
