# ------------------------------
# ビルド用
# ------------------------------
FROM golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS builder

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

# デプロイ対象環境（prd/stg/dev）。当該 env を埋め込み対象 env/.env へ焼き直す。
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
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS runtime

RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -u 10001 -G app app
# UID / GID を数値で固定するのは、Kubernetes の runAsNonRoot / runAsUser のように
# イメージの外から実行ユーザーを検証する仕組みが名前を解決できないため。
USER 10001

# コンテナのローカルタイム（Go の time.Local）。tzdata と対で決まり、実行時 env の TZ で上書きできる。
# 機構の役割分担と変更手順は env/README.md の Changing the Timezone を参照。
ENV TZ=Asia/Tokyo

# env / migrations はバイナリへ埋め込み済みのため、追加コピーは不要。
COPY --from=builder --chown=app:app /app/bin/server /app/server

WORKDIR /app

# 待受ポートは SERVER_PORT の既定値。上書きしたときは compose 側の healthcheck が優先されるため、
# ここはイメージ単体で起動した場合の判定になる。
# JSON 表記の exec 形式には `||` が無く、wget が返す 4 / 8 などの終了コードを
# Docker が unhealthy と解釈する 1 へ丸められない。判定を正しく出すためのシェル形式。
# hadolint ignore=DL3025
HEALTHCHECK --interval=10s --timeout=3s --retries=3 --start-period=10s \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# マイグレーションは専用イメージを持たず、この同一イメージを command override で実行する
# （["/app/server", "migrate-up"]）。詳細は docker/server/README.md の runtime 節。
CMD ["/app/server", "serve"]

# ------------------------------
# 開発用ツールビルダー：mise.toml のバージョン pin で開発用 Go ツール一式を install し、
# バイナリだけを /out に抽出する。最終 tooling ステージに mise 本体を残さないための中継。
# ------------------------------
FROM golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS tooling_builder

# CGO_ENABLED=0: `go:` backend のツールを静的バイナリにする（go_tools_builder と対称）
ENV CGO_ENABLED=0

RUN apk add --no-cache bash ca-certificates curl git

ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
# パイプのままだとリトライ時に部分受信分が sh へ流れ込むため、ファイルへ落としてから実行する。
RUN curl -fsSL --retry 3 --retry-delay 2 --retry-all-errors https://mise.run -o /tmp/mise-install.sh \
    && MISE_VERSION=v2026.6.0 sh /tmp/mise-install.sh \
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
  mise install air \
  && mise install go:github.com/go-delve/delve/cmd/dlv \
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
FROM golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS tooling

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

# コンテナのローカルタイム。runtime ステージと揃え、開発中も本番と同じ time.Local で動かす。
# 詳細は runtime ステージの同じ宣言のコメントを参照。
ENV TZ=Asia/Tokyo

# tooling_builder から開発用 Go ツールのバイナリだけを取り込む（mise 本体や install キャッシュは含まない）
COPY --from=tooling_builder /out/ /usr/local/bin/

WORKDIR /app

# コンテナ起動時に実行するコマンドを設定
CMD ["air", "-c", ".air.toml"]
