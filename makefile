# Makefile
.DEFAULT_GOAL := help

# 依存されるファイル
# Docker関連
include .makefiles/docker/compose.mk
# DB関連
include .makefiles/database/vars.mk
# Go言語関連
include .makefiles/go/vars.mk
# 負荷配分（重いターゲットが参照するため、それらより前に読む）
include .makefiles/load.mk

# 依存されないファイル
# DB関連
include .makefiles/database/migrate.mk
include .makefiles/database/dml-merge.mk
include .makefiles/database/seed.mk
include .makefiles/database/fix.mk
include .makefiles/database/gen.mk
include .makefiles/database/pool.mk
# Application関連
include .makefiles/app/server.mk
include .makefiles/app/job.mk
include .makefiles/app/worker.mk
include .makefiles/app/mock-auth.mk
include .makefiles/app/env.mk
# GitHub関連
include .makefiles/github/operation/release-branch.mk
include .makefiles/github/operation/release-tag.mk
include .makefiles/github/setting/github.mk
include .makefiles/github/setting/branch-ruleset.mk
include .makefiles/github/setting/label-setting.mk
include .makefiles/github/lint.mk
include .makefiles/github/commitlint.mk
include .makefiles/github/pin.mk
include .makefiles/github/workflows.mk
# Go言語関連
include .makefiles/go/fmt.mk
include .makefiles/go/gen.mk
include .makefiles/go/golangci-lint.mk
include .makefiles/go/installer.mk
include .makefiles/go/lib.mk
include .makefiles/go/test.mk
include .makefiles/go/sqlc.mk
# ドキュメント関連
include .makefiles/docs/gen.mk
# OpenAPI関連
include .makefiles/openapi/gen.mk
include .makefiles/openapi/mock-auth.mk
# SQL関連
include .makefiles/sql/fix.mk
include .makefiles/sql/lint.mk
# Markdown関連
include .makefiles/markdown/lint.mk
# セキュリティ関連
include .makefiles/security/trivy.mk
include .makefiles/security/gitleaks.mk
include .makefiles/security/npm-cooldown.mk
include .makefiles/security/go-cooldown.mk
include .makefiles/security/zizmor.mk
# Docker関連
include .makefiles/docker/lint.mk
include .makefiles/docker/pin.mk

# 一括実行系ファイル
# GitHub関連
include .makefiles/github/operation/setup-repository.mk
# DB関連
include .makefiles/database/start-up.mk
# 生成関連
include .makefiles/gen/gen.mk

.PHONY: help
help:
	@node scripts/make_help.mjs

