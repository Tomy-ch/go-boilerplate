# ------------------------------
# ビルド用
# ------------------------------
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

ENV GOPROXY=off \
    GOSUMDB=off \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

ARG VERSION=dev
ARG REVISION=none
ARG BUILD_DATE=2024-12-31T21:00:00Z

RUN apk add --no-cache \
  ca-certificates \
  tzdata

COPY go.mod go.sum ./

RUN go mod tidy

COPY vendor/ ./vendor/

COPY . .

RUN go list -mod=vendor all

RUN go build -mod=vendor -trimpath \
  -ldflags="-s -w \
    -X 'boilerplate-go/internal/system.Version=${VERSION}' \
    -X 'boilerplate-go/internal/system.Revision=${REVISION}' \
    -X 'boilerplate-go/internal/system.BuildDate=${BUILD_DATE}'" \
  -o /app/bin/server ./cmd/main.go

# ------------------------------
# 本番用
# ------------------------------
FROM alpine:3.23 AS prd

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

RUN addgroup -S app && adduser -S app -G app
USER app

COPY --from=builder --chown=app:app /app/bin/server /app/server
COPY --from=builder --chown=app:app /app/env/.env.prd /env/.env.prd
COPY --from=builder --chown=app:app /app/database/migrations /database/migrations
COPY --from=builder --chown=app:app /app/scripts/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh
CMD ["/entrypoint.sh"]

# ------------------------------
# ローカル環境
# ------------------------------
FROM golang:1.26.1-alpine AS local

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
