FROM node:24.18.0-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd AS mock_auth_server

WORKDIR /app

# 依存のみ先に入れてレイヤキャッシュを効かせる
COPY mock-auth-server/package.json mock-auth-server/package-lock.json ./
RUN npm ci --omit=dev --ignore-scripts

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
