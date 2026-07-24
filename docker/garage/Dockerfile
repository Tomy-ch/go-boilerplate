# syntax=docker/dockerfile:1
# ローカル開発用 Garage イメージ。公式 dxflrs/garage は scratch ベースで shell を持たず、
# プロビジョニング用の init スクリプトを実行できない。そこで alpine に garage バイナリを
# 載せ、server 起動と provisioning の両方を同一イメージで賄えるようにする。
FROM dxflrs/garage:v1.0.1@sha256:a5706cf1f3d7b349ac5133ec59ad8181b709270b75e5f2fa7b3e1a5c07d67137 AS garage

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
COPY --from=garage /garage /usr/local/bin/garage
COPY docker/garage/init.sh /usr/local/bin/garage-init.sh
RUN chmod +x /usr/local/bin/garage-init.sh
