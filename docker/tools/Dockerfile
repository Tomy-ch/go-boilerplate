# ------------------------------
# Go ツールビルダー：mise.toml のバージョン pin で Go 系ツール一式を install し、
# バイナリだけを /out に抽出する。最終 go_tools ステージに mise 本体を残さないための中継。
# ------------------------------
FROM golang:1.26.4-alpine AS go_tools_builder

# CGO_ENABLED=0: `go:` backend で install されるツール（mockgen）を静的バイナリにする
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

RUN mise install aqua:golang-migrate/migrate \
    && mise install go:go.uber.org/mock/mockgen \
    && mise install oapi-codegen \
    && mise install sqlc

RUN mkdir /out \
    && cp "$(mise which migrate)" /out/ \
    && cp "$(mise which mockgen)" /out/ \
    && cp "$(mise which oapi-codegen)" /out/ \
    && cp "$(mise which sqlc)" /out/

# ------------------------------
# Go ツールボックス：migrate / mockgen / oapi-codegen / sqlc を mise.toml のバージョンで提供
# Go ランタイムは golang 公式 base image を使用（mise.toml の go 値と `make sync-go-version` で同期）
# ------------------------------
FROM golang:1.26.4-alpine AS go_tools

RUN apk add --no-cache \
    bash \
    ca-certificates \
    make \
    postgresql-client

# go_tools_builder からツールのバイナリだけを取り込む（mise 本体や install キャッシュは含まない）
COPY --from=go_tools_builder /out/ /usr/local/bin/

WORKDIR /app

ENV GOCACHE=/tmp/go-build \
    GOMODCACHE=/tmp/go-mod

ENTRYPOINT []
CMD []

# ------------------------------
# Node ツールボックス：redocly / markdownlint-cli2 を mise.toml のバージョンで提供
# Node ランタイムは公式 node:X.Y-alpine（musl 対応のプリビルド）を使用。
# mise の Node ビルドは alpine 上でソースコンパイルを要求するため、ここでは公式 base に頼る。
# 補助スクリプト (gen-portal-docs.cjs / gen-docs-json.cjs) が参照する js-yaml は
# ローカル node_modules にインストールしておく
# ------------------------------
FROM node:24.14-alpine AS node_tools

RUN apk add --no-cache \
    bash \
    ca-certificates \
    curl \
    git \
    make

ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
RUN curl -fsSL https://mise.run | sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

RUN mise install npm:@redocly/cli \
    && mise install npm:markdownlint-cli2 \
    && rm -rf /root/.cache /root/.npm

WORKDIR /opt/tools
COPY docker/tools/package.json ./
RUN npm install
ENV NODE_PATH=/opt/tools/node_modules

WORKDIR /app

ENTRYPOINT []
CMD []

# ------------------------------
# Python ツールボックス：sqlfluff を mise.toml のバージョンで提供
# Python ランタイムは公式 python:X.Y.Z-slim（プリビルド）を使用。
# mise の Python ビルドは python-build によるソースコンパイルが必要で時間がかかるため、ここでは公式 base に頼る。
# NOTE: mise の pipx バックエンドが extras 構文（`sqlfluff[jinja]`）を受け付けないため、
#       jinja extras 無しの素の sqlfluff を入れている。
#       jinja テンプレートを SQL ファイルで使う場合は別途見直し。
# ------------------------------
FROM python:3.14.2-slim AS python_tools

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        make \
        pipx \
    && rm -rf /var/lib/apt/lists/*

ENV PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH" \
    MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise
RUN curl -fsSL https://mise.run | sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

RUN mise install pipx:sqlfluff \
    && rm -rf /root/.cache

ENTRYPOINT []
CMD []
