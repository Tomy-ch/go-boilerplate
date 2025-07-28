FROM nginx:1.29-otel AS document_viewer

COPY ./docker/document/config/default.conf /etc/nginx/conf.d/default.conf
COPY ./docs/ /usr/share/nginx/html/
