## Trivyによる脆弱性・設定不備・ライセンススキャンコマンド群
# -----Dockerコンテナ内で実行するコマンド群-----
.PHONY: trivy-fs ## 依存ライブラリの脆弱性を Trivy fs でスキャン
.PHONY: trivy-config ## Dockerfileの設定不備を Trivy config でスキャン
.PHONY: trivy-license ## 依存ライブラリのライセンスを Trivy fs でスキャン
# -----CI内で実行するコマンド群-----
.PHONY: trivy-fs-ci ## 依存ライブラリの脆弱性を Trivy fs でスキャン(CI用)
.PHONY: trivy-fs-release-ci ## 修正版のない脆弱性も含めて依存ライブラリをスキャン(CI用・リリースゲート)
.PHONY: trivy-config-ci ## Dockerfileの設定不備をスキャン(CI用)
.PHONY: trivy-license-ci ## 依存ライブラリのライセンスをスキャン(CI用)
.PHONY: trivy-image-ci ## ビルド済みイメージの脆弱性をスキャン(CI用)
.PHONY: trivy-image-gate-ci ## ビルド済みイメージを修正版のあるCRITICAL/HIGHでゲート(CI用)

# 出力形式。既定の table は人がローカルで読むためのもの。CI は json / sarif を渡して
# stdout をリダイレクトする。trivy は診断ログを stderr に出すため、stdout は純粋な
# json / sarif になる。
#
# ただし make 自身がレシピ行を stdout へエコーするため、機械可読な出力を取るときは
# 必ず `make -s` で呼ぶこと（例: make -s trivy-config-ci TRIVY_FORMAT=json > out.json）。
# -s を落とすと JSON の 1 行目にコマンド文字列が混ざり、jq が構文エラーになる。
TRIVY_FORMAT ?= table

# vendor/ は go.mod と同じ依存を二重計上するうえ、上流が同梱する pip の
# requirements.txt（vendor/go.opentelemetry.io/otel/requirements.txt）まで
# 本プロジェクトの依存として拾ってしまう。
TRIVY_SKIP_FLAGS := --skip-dirs vendor --skip-version-check

# 脆弱性スキャンの共通条件。--ignore-unfixed の有無だけが通常スキャンとリリース
# ゲートの差なので、それ以外はここへ集約する。
TRIVY_VULN_FLAGS := --scanners vuln --pkg-types library --severity CRITICAL,HIGH,MEDIUM

# -----Dockerコンテナ内で実行するコマンド群-----
trivy-fs:
	@docker compose run --rm go_tool_runner make trivy-fs-ci

trivy-config:
	@docker compose run --rm go_tool_runner make trivy-config-ci

trivy-license:
	@docker compose run --rm go_tool_runner make trivy-license-ci

# -----CI内で実行するコマンド群-----
# 通常の依存スキャンは報告専用。修正版のない脆弱性は、その PR が持ち込んだもので
# ない以上ここでは落とさず、trivy-fs-release-ci が昇格時にまとめて判定する。
trivy-fs-ci:
	trivy fs $(TRIVY_VULN_FLAGS) --ignore-unfixed $(TRIVY_SKIP_FLAGS) --format $(TRIVY_FORMAT) .

# デプロイ先ブランチ宛 PR の厳格ゲート。--ignore-unfixed を外し、修正版のない脆弱性も
# 含めた「今まさに昇格しようとしている依存の全体像」を返す。
trivy-fs-release-ci:
	trivy fs $(TRIVY_VULN_FLAGS) $(TRIVY_SKIP_FLAGS) --format $(TRIVY_FORMAT) .

# 設定不備スキャン。trivy config は設定ファイルだけを見るため、fs --scanners misconfig と
# 違って lang-pkgs の空 Result が混ざらない。.trivyignore.yaml は自動検出されないので
# --ignorefile の明示が必須であり、その結果として脆弱性スキャン側へ暗黙に波及しない。
# 対象は Dockerfile のみ。trivy は docker-compose 用のチェックを持たない。
trivy-config-ci:
	trivy config --severity CRITICAL,HIGH --ignorefile .trivyignore.yaml $(TRIVY_SKIP_FLAGS) --format $(TRIVY_FORMAT) .

# ライセンススキャン。禁止ライセンス方針が未策定のため報告専用とし、severity では
# 絞らない（方針策定前に閾値を決めると、その閾値自体が根拠のない既成事実になる）。
trivy-license-ci:
	trivy fs --scanners license $(TRIVY_SKIP_FLAGS) --format $(TRIVY_FORMAT) .

# イメージスキャン。ファイルシステムスキャンと違い OS パッケージも対象に含める
# （ベースイメージ由来の脆弱性はここでしか見えない）。対象イメージは CI がビルド
# 済みのローカルタグを TRIVY_IMAGE で渡す。
TRIVY_IMAGE ?=
TRIVY_IMAGE_FLAGS := --scanners vuln --pkg-types os,library

# 報告用。修正版の有無を問わず MEDIUM まで出す。
trivy-image-ci:
	trivy image $(TRIVY_IMAGE_FLAGS) --severity CRITICAL,HIGH,MEDIUM --skip-version-check --format $(TRIVY_FORMAT) $(TRIVY_IMAGE)

# ゲート用。修正版のある CRITICAL/HIGH だけで落とす。報告用と閾値が違うのは意図的で、
# 「直せるのに直していない」ものだけをマージブロックの理由にする。
trivy-image-gate-ci:
	trivy image $(TRIVY_IMAGE_FLAGS) --severity CRITICAL,HIGH --ignore-unfixed --skip-version-check --exit-code 1 --format table $(TRIVY_IMAGE)
