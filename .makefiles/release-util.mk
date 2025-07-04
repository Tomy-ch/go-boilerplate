.PHONY: fetch-tags

fetch-tags: ## Repoのタグを最新化
	@git fetch --tags origin

get-latest-version = $(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1)

next-micro = $(shell echo $(1) | awk -F. -v OFS=. '{$$3++; print $$1,$$2,$$3}')
next-minor = $(shell echo $(1) | awk -F. -v OFS=. '{$$2++; $$3=0; print $$1,$$2,$$3}')
next-major = $(shell echo $(1) | awk -F. -v OFS=. '{$$1++; $$2=0; $$3=0; print $$1,$$2,$$3}')
