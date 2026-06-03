# ------------------------------
# ビルド用
# ------------------------------
# NOTE: 本番利用時にはダイジェストを指定することを推奨
FROM golang:1.26.3-alpine AS builder

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
COPY vendor/ ./vendor/

COPY . .

RUN go build -mod=vendor -trimpath \
  -ldflags="-s -w \
    -X 'go-boilerplate/internal/system.Version=${VERSION}' \
    -X 'go-boilerplate/internal/system.Revision=${REVISION}' \
    -X 'go-boilerplate/internal/system.BuildDate=${BUILD_DATE}'" \
  -o /app/bin/server ./cmd/main.go

# ------------------------------
# 実行用コンテナ: アプリケーションのサーバとして常駐起動する用途
# ------------------------------
# NOTE: 本番利用時にはダイジェストを指定することを推奨
FROM alpine:3.23 AS runtime

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S app -G app
USER app

COPY --from=builder --chown=app:app /app/bin/server /app/server
COPY --from=builder --chown=app:app /app/env/.env.dev /app/env/.env.dev
COPY --from=builder --chown=app:app /app/env/.env.stg /app/env/.env.stg
COPY --from=builder --chown=app:app /app/env/.env.prd /app/env/.env.prd

WORKDIR /app

ENV ENV=prd

CMD ["/app/server", "serve"]

# ------------------------------
# マイグレーション用コンテナ: デプロイ前に1回だけ実行されるジョブ用途
# サーバとして常駐させてはいけない
# ------------------------------
FROM runtime AS migration

COPY --from=builder --chown=app:app /app/database/migrations /app/database/migrations

WORKDIR /app

ENV ENV=prd

CMD ["/app/server", "migrate-up"]

# ------------------------------
# 開発用ツールビルダー：mise.toml のバージョン pin で開発用 Go ツール一式を install し、
# バイナリだけを /out に抽出する。最終 tooling ステージに mise 本体を残さないための中継。
# ------------------------------
FROM golang:1.26.3-alpine AS tooling_builder

# CGO_ENABLED=0: `go:` backend のツールを静的バイナリにする（go_tools_builder と対称）
ENV CGO_ENABLED=0

RUN apk add --no-cache bash ca-certificates curl git

ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
RUN curl -fsSL https://mise.run | sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

RUN mise install air \
  && mise install aqua:go-delve/delve \
  && mise install golines \
  && mise install gofumpt \
  && mise install golangci-lint \
  && mise install lefthook

RUN mkdir /out \
  && cp "$(mise which air)" /out/ \
  && cp "$(mise which dlv)" /out/ \
  && cp "$(mise which golines)" /out/ \
  && cp "$(mise which gofumpt)" /out/ \
  && cp "$(mise which golangci-lint)" /out/ \
  && cp "$(mise which lefthook)" /out/

# ------------------------------
# ローカル環境
# ------------------------------
FROM golang:1.26.3-alpine AS tooling

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

# tooling_builder から開発用 Go ツールのバイナリだけを取り込む（mise 本体や install キャッシュは含まない）
COPY --from=tooling_builder /out/ /usr/local/bin/

WORKDIR /app

# コンテナ起動時に実行するコマンドを設定
CMD ["air", "-c", ".air.toml"]
