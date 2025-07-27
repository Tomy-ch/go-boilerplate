# ------------------------------
# 開発環境
# ------------------------------
FROM golang:1.24.4-alpine as local

# ビルドなどに必要なパッケージをインストール
RUN apk add --no-cache \
  build-base \
  binutils-gold \
  bash \
  curl \
  git \
  upx \
  libc6-compat \
  gcompat \
  tzdata \
  make

# Goのモジュール
RUN go install github.com/air-verse/air@latest \
  && go install github.com/go-delve/delve/cmd/dlv@latest \
  && go install github.com/segmentio/golines@latest \
  && go install github.com/evilmartians/lefthook@latest \
  && go install mvdan.cc/gofumpt@latest \
  && go install github.com/fatih/gomodifytags@latest \
  && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# go installでインストールしたバイナリのパスを通す
ENV PATH="/go/bin:$PATH"

# 作業ディレクトリ(/app)を設定
WORKDIR /app

# コンテナ起動時に実行するコマンドを設定
CMD ["air", "-c", ".air.toml"]
