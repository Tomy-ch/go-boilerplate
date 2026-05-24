# ------------------------------
# Go (builder) - ここでだけ go install する
# ------------------------------
FROM golang:1.26.3-alpine AS go_tools_builder

ENV CGO_ENABLED=0 \
    GOBIN=/out

RUN apk add --no-cache git

# SQLMigration系
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# コード生成系
RUN go install go.uber.org/mock/mockgen@latest \
  && go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# クエリビルダー生成系
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

RUN rm -rf /go/pkg/mod /root/.cache/go-build

# ------------------------------
# Go
# ------------------------------
FROM golang:1.26.3-alpine AS go_tools

RUN apk add --no-cache bash make postgresql-client

WORKDIR /app

COPY --from=go_tools_builder /out/ /usr/local/bin/

ENV GOCACHE=/tmp/go-build \
    GOMODCACHE=/tmp/go-mod

ENTRYPOINT []
CMD []

# ------------------------------
# Node.js
# ------------------------------
FROM node:24.14-alpine AS node_tools

RUN apk add --no-cache bash make

WORKDIR /opt/tools

COPY docker/tools/package.json ./
RUN npm install

# node_modulesの参照パスを追加
ENV NODE_PATH=/opt/tools/node_modules

RUN npm install -g @redocly/cli@latest \
  && npm install -g markdownlint-cli2@latest

WORKDIR /app

ENTRYPOINT []
CMD []

# ------------------------------
# Python
# ------------------------------
FROM python:3.14.2-slim AS python_tools

RUN apt-get update

RUN apt-get install -y --no-install-recommends make bash

RUN rm -rf /var/lib/apt/lists/*

WORKDIR /app
RUN pip install --no-cache-dir "sqlfluff[jinja]"

ENTRYPOINT []
CMD []
