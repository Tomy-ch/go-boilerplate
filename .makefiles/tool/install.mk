.PHONY: install

install:
	go install mvdan.cc/gofumpt@latest
	go install github.com/segmentio/golines@latest
	go install github.com/evilmartians/lefthook@latest
	echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zprofile
