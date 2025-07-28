# ------------------------------
# OpenAPI
# ------------------------------
FROM node:24.4-alpine AS node_tools

WORKDIR /app

RUN npm install -g swagger-cli@latest
RUN npm install -g @redocly/cli@latest

COPY openapi/ openapi/

# ------------------------------
# Goでのコード生成
# ------------------------------
FROM golang:1.24.4-alpine AS go_tools

WORKDIR /app

RUN go install go.uber.org/mock/mockgen@latest \
&& go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

ENV PATH="/go/bin:$PATH"

COPY . .
