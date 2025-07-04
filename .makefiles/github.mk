.PHONY: gh-login ## ghコマンドでログインを行います

gh-login:
	gh auth login --hostname github.com --web
