# syntax=docker/dockerfile:1
# ローカル開発用 Garage のプロビジョニング専用イメージ（server 本体は公式イメージをそのまま使う）。
# 公式 dxflrs/garage は scratch ベースで shell を持たず init スクリプトを実行できないため、
# alpine へ garage バイナリを載せて CLI を shell から叩けるようにする。
# init スクリプト自体は compose が実体をマウントするため、イメージには焼かない。
FROM dxflrs/garage:v2.3.0@sha256:866bd13ed2038ba7e7190e840482bc27234c4afaf77be8cfa439ae088c1e4690 AS garage

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
COPY --from=garage /garage /usr/local/bin/garage
