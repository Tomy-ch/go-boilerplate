# 依存されるファイル
include .makefiles/release-util.mk
include .makefiles/github.mk

# 依存されないファイル
include .makefiles/release-branch.mk
include .makefiles/release-tag.mk
include .makefiles/setup-repository.mk
include .makefiles/update.mk
