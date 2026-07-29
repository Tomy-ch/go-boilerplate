# mise 本体の固定バージョン（全ツールステージ共通。bump 時はここだけ変更する）
ARG MISE_VERSION=v2026.6.0

# ------------------------------
# Go ツールビルダー：mise.toml のバージョン pin で Go 系ツール一式を install し、
# バイナリだけを /out に抽出する。最終 go_tools ステージに mise 本体を残さないための中継。
# ------------------------------
FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go_tools_builder

# CGO_ENABLED=0: `go:` backend で install されるツール（mockgen）を静的バイナリにする
ENV CGO_ENABLED=0

RUN apk add --no-cache bash ca-certificates curl git

ARG MISE_VERSION
ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
# パイプのままだとリトライ時に部分受信分が sh へ流れ込むため、ファイルへ落としてから実行する。
RUN curl -fsSL --retry 3 --retry-delay 2 --retry-all-errors https://mise.run -o /tmp/mise-install.sh \
    && MISE_VERSION=${MISE_VERSION} sh /tmp/mise-install.sh \
    && rm /tmp/mise-install.sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

# 未認証の GitHub API（60 req/hour・IP 単位）ではこの install 群を賄えず 403 になる。トークンは
# イメージ履歴に焼き込む build-arg を避け secret で渡す（未指定なら従来どおり未認証で成立）。
RUN --mount=type=secret,id=github_token \
    github_token="$(cat /run/secrets/github_token 2>/dev/null || true)"; \
    if [ -n "$github_token" ]; then export GITHUB_TOKEN="$github_token"; fi; \
    mise install aqua:golang-migrate/migrate \
    && mise install go:go.uber.org/mock/mockgen \
    && mise install oapi-codegen \
    && mise install sqlc \
    && mise install aqua:aquasecurity/trivy \
    && mise install actionlint \
    && mise install shellcheck \
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
    && cp "$(mise which shellcheck)" /out/ \
    && cp "$(mise which hadolint)" /out/ \
    && cp "$(mise which gitleaks)" /out/ \
    && cp "$(mise which godoc)" /out/ \
    && cp "$(mise which godoc-static)" /out/

# ------------------------------
# Go ツールボックス：migrate / mockgen / oapi-codegen / sqlc / trivy / actionlint / shellcheck / hadolint / gitleaks / godoc / godoc-static を mise.toml のバージョンで提供
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
FROM node:26.5.0-alpine@sha256:e88a35be04478413b7c71c455cd9865de9b9360e1f43456be5951032d7ac1a66 AS node_tools

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
RUN curl -fsSL --retry 3 --retry-delay 2 --retry-all-errors https://mise.run -o /tmp/mise-install.sh \
    && MISE_VERSION=${MISE_VERSION} sh /tmp/mise-install.sh \
    && rm /tmp/mise-install.sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

# install 対象は npm backend だが、mise は対象に関わらず mise.toml 全体を解決するため GitHub を引く。
RUN --mount=type=secret,id=github_token \
    github_token="$(cat /run/secrets/github_token 2>/dev/null || true)"; \
    if [ -n "$github_token" ]; then export GITHUB_TOKEN="$github_token"; fi; \
    mise install npm:@redocly/cli \
    && mise install npm:markdownlint-cli2 \
    && mise install npm:@commitlint/cli \
    && rm -rf /root/.cache /root/.npm

# scripts の依存を in-tree（/app/scripts/node_modules）へ install する。
# --ignore-scripts はサプライチェーン攻撃の主要 RCE 経路（依存の postinstall 等）を封じる防御。
# 本ツリーの install script は esbuild のみで、そのバイナリは @esbuild/<platform>（lock 済み）
# から実行時に解決されるため、script を飛ばしても portal ビルドは動く。
WORKDIR /app/scripts
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
# pipx backend の実行系は mise.toml の pipx.uvx=true により uv（mise 管理版）へ固定。
# ------------------------------
FROM python:3.14.6-slim@sha256:cea0e6040540fb2b965b6e7fb5ffa00871e632eef63719f0ea54bca189ce14a6 AS python_tools

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        make \
    && rm -rf /var/lib/apt/lists/*

ARG MISE_VERSION
ENV PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH" \
    MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise
RUN curl -fsSL --retry 3 --retry-delay 2 --retry-all-errors https://mise.run -o /tmp/mise-install.sh \
    && MISE_VERSION=${MISE_VERSION} sh /tmp/mise-install.sh \
    && rm /tmp/mise-install.sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

# uv 自体が aqua:astral-sh/uv（GitHub Releases 解決）であるうえ、mise は mise.toml 全体を解決する。
RUN --mount=type=secret,id=github_token \
    github_token="$(cat /run/secrets/github_token 2>/dev/null || true)"; \
    if [ -n "$github_token" ]; then export GITHUB_TOKEN="$github_token"; fi; \
    mise install uv \
    && mise install pipx:sqlfluff \
    && rm -rf /root/.cache

ENTRYPOINT []
CMD []
