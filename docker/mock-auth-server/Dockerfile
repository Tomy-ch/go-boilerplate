# ------------------------------
# 本番依存の解決だけを担う段。pnpm はランタイム段へ持ち込まない
# ------------------------------
# pnpm の版は mise.toml[SSOT] のピンから取る。ここで npm から直に入れると版が二重管理になり、
# tool-cooldown ゲート（レジストリ公開日で測る窓）の管理外へ出る。
FROM node:24.18.0-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd AS mock_auth_server_deps

RUN apk add --no-cache ca-certificates

ARG MISE_VERSION
# 上流は install スクリプトの checksum を公開しないため、この固定値を ADD に照合させてから
# 実行する。上流がスクリプトを更新すると取得の時点で落ちる。追随するときは差分を見て値を更新する。
#
# CKV_DOCKER_4 が ADD を避けよと言うのは、暗黙の展開とリモート取得が検証を伴わないため。
# --checksum はその検証そのものであり、COPY はリモート URL を取れないので代替にならない。
#checkov:skip=CKV_DOCKER_4:--checksum 付きの ADD が検証付き取得の形。COPY はリモートを扱えない
ADD --checksum=sha256:b11ef4cdc9bb3d8acdb95ad1cc2905b76d47936e9b8d9e76faded4d44f39f711 \
    https://mise.run /tmp/mise-install.sh

ENV MISE_DATA_DIR=/root/.local/share/mise \
    MISE_CONFIG_DIR=/root/.config/mise \
    PATH="/root/.local/bin:/root/.local/share/mise/shims:$PATH"
RUN MISE_VERSION="${MISE_VERSION}" sh /tmp/mise-install.sh \
    && rm /tmp/mise-install.sh \
    && /root/.local/bin/mise --version

WORKDIR /app
COPY mise.toml ./
RUN mise trust mise.toml

# install 対象は npm backend だが、mise は対象に関わらず mise.toml 全体を解決するため GitHub を引く。
RUN --mount=type=secret,id=github_token \
    github_token="$(cat /run/secrets/github_token 2>/dev/null || true)"; \
    if [ -n "$github_token" ]; then export GITHUB_TOKEN="$github_token"; fi; \
    mise install npm:pnpm \
    && rm -rf /root/.cache /root/.npm

# 依存ライフサイクルスクリプトの実行可否は pnpm-workspace.yaml が宣言する
# （strictDepBuilds + allowBuilds）。宣言に無い依存が script を持ち込んだ時点で install が失敗する。
COPY mock-auth-server/package.json mock-auth-server/pnpm-lock.yaml mock-auth-server/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile --prod \
    && pnpm store prune \
    && rm -rf /root/.cache

# ------------------------------
# ランタイム段
# ------------------------------
FROM node:24.18.0-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd AS mock_auth_server

WORKDIR /app

# 依存は解決済みのものを受け取る。pnpm の node_modules は .pnpm 実体への相対シンボリックリンクで
# 構成されるため、ディレクトリごと移す必要がある。
COPY --from=mock_auth_server_deps /app/node_modules ./node_modules
COPY mock-auth-server/package.json ./

# アプリ本体（TypeScript。Node の型ストリッピングでビルド無しに実行する） / 固定 RSA 鍵 / 固定 User Fixture
COPY mock-auth-server/src ./src
COPY mock-auth-server/keys ./keys
COPY mock-auth-server/fixtures ./fixtures

ENV OIDC_PORT=4000
EXPOSE 4000

# 非 root 実行（node イメージ同梱の node ユーザー）
USER node

HEALTHCHECK --interval=2s --timeout=2s --retries=30 --start-period=3s \
  CMD wget --quiet --tries=1 --spider http://localhost:4000/health || exit 1

# .ts を直接実行（Node 24 のネイティブ型ストリッピング。tsc ビルド不要・追加ランタイム依存なし）
CMD ["node", "src/server.ts"]
