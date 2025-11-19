# ------------------------------
# Go (builder) - ここでだけ go install する
# ------------------------------
FROM golang:1.24.4-alpine AS go_tools_builder

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
FROM golang:1.24.4-alpine AS go_tools

WORKDIR /app

ENV PATH="/go/bin:$PATH"

RUN apk add --no-cache bash

COPY --from=go_tools_builder /out/ /usr/local/bin/

ENTRYPOINT []
CMD []

# ------------------------------
# Node.js
# ------------------------------
FROM node:24.4-alpine AS node_tools

WORKDIR /app

RUN npm install -g swagger-cli@latest --no-audit --no-fund \
  && npm install -g @redocly/cli@latest

ENTRYPOINT []
CMD []
