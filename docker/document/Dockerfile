#checkov:skip=CKV_DOCKER_3:nginx は :80 を束縛するため master プロセスに root が要る（worker は公式イメージが nginx ユーザーへ落とす）
# 上の #checkov:skip は checkov 自身の判定からしか外れない。code scanning へは checkov.yaml の
# 「Drop suppressed results」が落としており、片方だけでは消えない。
FROM nginx:1.31-alpine@sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2 AS document_viewer

COPY ./docker/document/config/default.conf /etc/nginx/conf.d/default.conf

# JSON 表記の exec 形式には `||` が無く、wget が返す 4 / 8 などの終了コードを
# Docker が unhealthy と解釈する 1 へ丸められない。判定を正しく出すためのシェル形式。
# hadolint ignore=DL3025
HEALTHCHECK --interval=10s --timeout=3s --retries=3 --start-period=5s \
  CMD wget --quiet --tries=1 --spider http://localhost:80/ || exit 1
