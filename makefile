# 変数定義
# 環境（local / test / prd など）。未指定なら local
ENV ?= local

# 依存されるファイル
include .makefiles/github/operation/release-util.mk
include .makefiles/github/setting/github.mk
include .makefiles/github/setting/branch-ruleset.mk
include .makefiles/github/setting/label-setting.mk

# 依存されないファイル
include .makefiles/github/operation/release-branch.mk
include .makefiles/github/operation/release-tag.mk
include .makefiles/github/operation/setup-repository.mk
include .makefiles/dev/develop.mk
include .makefiles/database/database.mk
include .makefiles/gen/generate.mk

.PHONY: help
help:
	@bash scripts/make_help.sh

