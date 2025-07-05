# 依存されるファイル
include .makefiles/github/operation/release-util.mk
include .makefiles/github/setting/github.mk
include .makefiles/github/setting/branch-ruleset.mk
include .makefiles/github/setting/label-setting.mk

# 依存されないファイル
include .makefiles/github/operation/release-branch.mk
include .makefiles/github/operation/release-tag.mk
include .makefiles/github/operation/setup-repository.mk
include .makefiles/go/format.mk
include .makefiles/go/develop.mk
include .makefiles/go/start.mk
include .makefiles/tool/update.mk
include .makefiles/tool/install.mk
