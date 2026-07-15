# mise 本体の固定バージョン（全ツールステージ共通。bump 時はここだけ変更する）
ARG MISE_VERSION=v2026.6.0

# ------------------------------
# Go ツールビルダー：mise.toml のバージョン pin で Go 系ツール一式を install し、
# バイナリだけを /out に抽出する。最終 go_tools ステージに mise 本体を残さないための中継。
# ------------------------------
FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go_tools_builder

# CGO_ENABLED=0: `go:` backend で install されるツール（mockgen）を静的バイナリにする
ENV CGO_ENABLED=0

# パイプを含む RUN（mise インストール）の失敗を確実に検知するため pipefail を有効化（hadolint DL4006）
SHELL ["/bin/ash", "-o", "pipefail", "-c"]

RUN apk add --no-cache bash ca-certificates curl git

ARG MISE_VERSION
ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
RUN curl -fsSL https://mise.run | MISE_VERSION=${MISE_VERSION} sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

RUN mise install aqua:golang-migrate/migrate \
    && mise install go:go.uber.org/mock/mockgen \
    && mise install oapi-codegen \
    && mise install sqlc \
    && mise install aqua:aquasecurity/trivy \
    && mise install actionlint \
    && mise install hadolint \
    && mise install gitleaks \
    && mise install go:golang.org/x/tools/cmd/godoc \
    && mise install go:codeberg.org/tslocum/godoc-static

RUN mkdir /out \
    && cp "$(mise which migrate)" /out/ \
    && cp "$(mise which mockgen)" /out/ \
    && cp "$(mise which oapi-codegen)" /out/ \
    && cp "$(mise which sqlc)" /out/ \
    && cp "$(mise which trivy)" /out/ \
    && cp "$(mise which actionlint)" /out/ \
    && cp "$(mise which hadolint)" /out/ \
    && cp "$(mise which gitleaks)" /out/ \
    && cp "$(mise which godoc)" /out/ \
    && cp "$(mise which godoc-static)" /out/

# ------------------------------
# Go ツールボックス：migrate / mockgen / oapi-codegen / sqlc / trivy / actionlint / hadolint / gitleaks / godoc / godoc-static を mise.toml のバージョンで提供
# Go ランタイムは golang 公式 base image を使用（mise.toml の go 値と `make sync-go-version` で同期）
# ------------------------------
FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go_tools

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
# 補助スクリプトが依存する npm パッケージをローカル node_modules にインストールしておく
# ------------------------------
FROM node:24.18.0-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd AS node_tools

# パイプを含む RUN（mise インストール）の失敗を確実に検知するため pipefail を有効化（hadolint DL4006）
SHELL ["/bin/ash", "-o", "pipefail", "-c"]

RUN apk add --no-cache \
    bash \
    ca-certificates \
    curl \
    git \
    make

ARG MISE_VERSION
ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
RUN curl -fsSL https://mise.run | MISE_VERSION=${MISE_VERSION} sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

RUN mise install npm:@redocly/cli \
    && mise install npm:markdownlint-cli2 \
    && mise install npm:@commitlint/cli \
    && rm -rf /root/.cache /root/.npm

# scripts の依存を in-tree（/app/scripts/node_modules）へ install する。
# --ignore-scripts はサプライチェーン攻撃の主要 RCE 経路（依存の postinstall 等）を封じる防御。
# 本ツリーの install script は esbuild のみで、そのバイナリは @esbuild/<platform>（lock 済み）
# から実行時に解決されるため、script を飛ばしても portal ビルドは動く。
COPY docker/tools/package.json docker/tools/package-lock.json docker/tools/.npmrc ./
RUN npm ci --ignore-scripts \
    && npm cache clean --force \
    && rm -rf /root/.npm

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
FROM python:3.14.6-slim@sha256:d3400aa122fa42cf0af0dbe8ec3091b047eac5c8f7e3539f7135e86d855dc015 AS python_tools

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        make \
        pipx \
    && rm -rf /var/lib/apt/lists/*

# パイプを含む RUN（mise インストール）の失敗を確実に検知するため pipefail を有効化（hadolint DL4006）
# bash は上の apt-get で導入済み（python:slim は debian ベースで /bin/ash を持たない）
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG MISE_VERSION
ENV PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH" \
    MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise
RUN curl -fsSL https://mise.run | MISE_VERSION=${MISE_VERSION} sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

RUN mise install pipx:sqlfluff \
    && rm -rf /root/.cache

ENTRYPOINT []
CMD []
