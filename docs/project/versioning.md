# Versioning Policy

This repository follows **Semantic Versioning (SemVer)**.

Version numbers consist of three parts:

- MAJOR
- MINOR
- PATCH

## Version Definitions

- **MAJOR**  
  Introduces breaking changes that are not backward compatible.

- **MINOR**  
  Adds new features while maintaining backward compatibility.

- **PATCH**  
  Includes bug fixes and non-breaking improvements.

## Release Branch Strategy

This repository uses a **release-centric branching model**.

- Feature development branches from the latest `release/*` branch
- Changes are propagated to `develop`, `staging`, and `production` **only via release branches**
- Direct commits to protected branches are prohibited

## Release Procedure

### Tag Creation

Tags must be created using the following commands:

```bash
make release-major-tag
make release-minor-tag
make release-patch-tag
```

Manual tag creation is **not allowed**.

### Creating the Next Release Branch

Use the following commands to create the next release branch:

```bash
make release-major-branch
make release-minor-branch
make release-patch-branch
```

### Hotfix Procedure

When an urgent fix is required:

- Create a `hotfix/*` branch from the `production` branch using `make hotfix-patch-branch`
- Apply the fix on the `hotfix/*` branch and merge it into `production`
- From the updated `production` branch, create the next `release/*` branch using `make release-patch-branch`
- Merge changes into `develop`, `staging`, and `production` **only through the new `release/*` branch**
- Once the `release/*` branch is merged into `production`, create the PATCH version tag using `make release-patch-tag`

## Rules for Breaking Changes

- Breaking changes are allowed **only in MAJOR versions**
- API contract changes must follow the **OpenAPI-first policy**
- When OpenAPI specifications change, **code generation must be executed**

## Principles

The following rules must always be followed:

- Direct modification of version numbers is prohibited
- Tags must be created only through predefined `make` commands
- Branch protection rules must be respected
- Semantic Versioning must be strictly followed
