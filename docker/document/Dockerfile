#checkov:skip=CKV_DOCKER_3:nginx は :80 を束縛するため master プロセスに root が要る（worker は公式イメージが nginx ユーザーへ落とす）
FROM nginx:1.31-alpine@sha256:4a73073bd557c65b759505da037898b61f1be6cbcc3c2c3aeac22d2a470c1752 AS document_viewer

COPY ./docker/document/config/default.conf /etc/nginx/conf.d/default.conf

# JSON 表記の exec 形式には `||` が無く、wget が返す 4 / 8 などの終了コードを
# Docker が unhealthy と解釈する 1 へ丸められない。判定を正しく出すためのシェル形式。
# hadolint ignore=DL3025
HEALTHCHECK --interval=10s --timeout=3s --retries=3 --start-period=5s \
  CMD wget --quiet --tries=1 --spider http://localhost:80/ || exit 1
