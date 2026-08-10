#checkov:skip=CKV_DOCKER_3:nginx は :80 を束縛するため master プロセスに root が要る（worker は公式イメージが nginx ユーザーへ落とす）
FROM nginx:1.31-alpine@sha256:54f2a904c251d5a34adf545a72d32515a15e08418dae0266e23be2e18c66fefa AS document_viewer

COPY ./docker/document/config/default.conf /etc/nginx/conf.d/default.conf

HEALTHCHECK --interval=10s --timeout=3s --retries=3 --start-period=5s \
  CMD wget --quiet --tries=1 --spider http://localhost:80/ || exit 1
