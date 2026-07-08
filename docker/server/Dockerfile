# ------------------------------
# ビルド用
# ------------------------------
# NOTE: 本番利用時にはダイジェストを指定することを推奨
FROM golang:1.27rc2-alpine AS builder

WORKDIR /app

# buildx の自動 platform ARG に追従する（--platform 指定時はその arch、未指定/classic builder では既定 linux/amd64）
ARG TARGETOS=linux
ARG TARGETARCH=amd64

ENV GOPROXY=off \
    GOSUMDB=off \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

ARG VERSION=dev
ARG REVISION=none
# BUILD_DATE は --build-arg 未指定ビルド用の placeholder（本番デプロイは実 timestamp を注入）。
# version handler が RFC3339 として厳格 parse し不正値は /version を 500 にするため、有効な RFC3339 を維持すること。
ARG BUILD_DATE=2024-12-31T21:00:00Z

# デプロイ対象環境（prd/stg/dev/ci）。当該 env を埋め込み対象 env/.env へ焼き直す。
ARG APP_ENV=prd

COPY . .

RUN cp "env/.env.${APP_ENV}" env/.env \
  && go build -mod=vendor -trimpath \
  -ldflags="-s -w \
    -X 'go-boilerplate/internal/system.Version=${VERSION}' \
    -X 'go-boilerplate/internal/system.Revision=${REVISION}' \
    -X 'go-boilerplate/internal/system.BuildDate=${BUILD_DATE}'" \
  -o /app/bin/server ./cmd/

# ------------------------------
# 実行用コンテナ: アプリケーションのサーバとして常駐起動する用途
# ------------------------------
# NOTE: 本番利用時にはダイジェストを指定することを推奨
FROM alpine:3.24 AS runtime

RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app
USER app

# env / migrations はバイナリへ埋め込み済みのため、追加コピーは不要。
COPY --from=builder --chown=app:app /app/bin/server /app/server

WORKDIR /app

# 既定はサーバ常駐。マイグレーションは同一イメージを command override で実行する
# （例: 起動コマンドを ["/app/server", "migrate-up"] に差し替えてジョブ実行）。
CMD ["/app/server", "serve"]

# ------------------------------
# 開発用ツールビルダー：mise.toml のバージョン pin で開発用 Go ツール一式を install し、
# バイナリだけを /out に抽出する。最終 tooling ステージに mise 本体を残さないための中継。
# ------------------------------
FROM golang:1.27rc2-alpine AS tooling_builder

# CGO_ENABLED=0: `go:` backend のツールを静的バイナリにする（go_tools_builder と対称）
ENV CGO_ENABLED=0

# パイプを含む RUN（mise インストール）の失敗を確実に検知するため pipefail を有効化（hadolint DL4006）
SHELL ["/bin/ash", "-o", "pipefail", "-c"]

RUN apk add --no-cache bash ca-certificates curl git

ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
RUN curl -fsSL https://mise.run | MISE_VERSION=v2026.6.0 sh \
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
FROM golang:1.27rc2-alpine AS tooling

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
