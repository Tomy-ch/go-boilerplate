.PHONY: go-update ## goenvの更新を実行

go-update:
	@anyenv update
	@goenv install "$(cat .go-version)"
