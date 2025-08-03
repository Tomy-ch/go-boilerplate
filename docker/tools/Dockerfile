# ------------------------------
# Go
# ------------------------------
FROM golang:1.24.4-alpine AS go_tools

WORKDIR /app

ENV PATH="/go/bin:$PATH"

RUN apk add --no-cache bash

# SQLMigration系
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
  && go install github.com/jackc/tern@latest

# コード生成系
RUN go install go.uber.org/mock/mockgen@latest \
  && go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

ENTRYPOINT []
CMD []

# ------------------------------
# Node.js
# ------------------------------
FROM node:24.4-alpine AS node_tools

WORKDIR /app

RUN npm install -g swagger-cli@latest \
  && npm install -g @redocly/cli@latest

ENTRYPOINT []
CMD []
