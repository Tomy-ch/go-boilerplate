---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy, image, config]
---

# ADR-0085: 環境別イメージ（.env マトリックス × APP_ENV ビルド引数、ビルド時に固定）

English canonical: [0085-per-environment-images.md](../../adr/0085-per-environment-images.md)

## ステータス

accepted

## 背景

アプリケーションは `go:embed` を通じてビルド時にバイナリに埋め込まれた `.env` ファイルから設定を読み取る（ADR-0036 参照）。埋め込みファイルはコンパイル時にベイクインされるため、単一のバイナリは 1 つの環境の設定しか持てない。異なる環境の設定（プロダクション、ステージング、開発）はそのため別々のイメージを必要とする。

同時に、環境固有のシークレットや設定値をコンテナランタイムで（環境変数またはマウントされたファイルを通じて）注入できなければならない。埋め込まれた設定は安全なデフォルトを提供し、ランタイムの環境変数がそれらをオーバーライドする。

CI/CD パイプラインは 3 つの長命ブランチ（`production`、`staging`、`develop`）で実行され、これらは 1:1 で環境にマップされる。

## 決定

各ブランチのプッシュは、`APP_ENV` ビルド引数で選択されたマッチする `.env.<env>` ファイルを埋め込んだイメージを作成する:

| ブランチ | APP_ENV | 埋め込み設定 |
| --- | --- | --- |
| `production` | `prd` | `env/.env.prd` |
| `staging` | `stg` | `env/.env.stg` |
| `develop` | `dev` | `env/.env.dev` |

`builder` ステージはコンパイル前にターゲット設定をマテリアライズする:

```text
cp "env/.env.${APP_ENV}" env/.env
go build ... -o /app/bin/server ./cmd/
```

ワークフローの `Define build environment` ステップは `github.ref_name` を `app_env` にマップし、Docker ビルドへの `build-arg` として渡す。ランタイム環境変数は埋め込まれた値をオーバーライドできるため、シークレットを埋め込む必要はない。

## 影響

### ポジティブな影響

- コンテナが実行される環境はランタイム注入ではなくビルド時に決定され、設定ミスのクラス（誤った環境変数、コンテナ起動時の環境変数の欠落）を排除する。
- 各イメージは独立して検証可能かつ監査可能である: 埋め込まれた設定は署名された成果物の一部である（[ADR-0087](0087-release-image-supply-chain.ja.md) 参照）。
- ベース設定値のためのランタイム設定管理サイドカーやシークレット注入ステップは必要ない。

### ネガティブな影響

- 特定の環境の設定変更には、その環境のイメージの完全な再ビルドとプッシュが必要となる。
- 3 つの異なるイメージタグを管理しなければならず（環境ごとに 1 つ）、レジストリストレージが増加する。
- シークレットは `env/.env.<env>` ファイルに置いてはならない（イメージレイヤーに埋め込まれるため）。シークレットにはランタイム環境変数注入が適切な方法である。

## 検討した代替案

### ランタイム設定注入を持つ単一イメージ

すべての環境に対して 1 つのイメージ。環境変数またはマウントされたファイルがランタイムで設定を提供する。シンプルなレジストリになるが、すべてのデプロイターゲットで信頼できる設定注入メカニズム（シークレットマネージャー、マウントされた ConfigMap/Secret）が必要となり、「このコンテナはどの設定で実行されているか」という問いにイメージのみから答えることが困難になる。

### 環境別の Dockerfile

環境ごとに別の Dockerfile。ビルドとメンテナンスの労力が重複する。`APP_ENV` ビルド引数は単一の Dockerfile から重複なしに同じ結果を達成する。

## 補足

- `docker/server/Dockerfile` 25-30 行: `ARG APP_ENV=prd` と `cp` コマンド。
- `.github/workflows/deploy-app.yaml` の `Define build environment` ステップ（54-63 行）: ブランチ → `app_env` マッピング。
- `env/README.md`: "Env files are embedded into the binary at build time (`embed.go`). The Docker `builder` stage materializes the target via the `APP_ENV` build arg."
- 埋め込み設定メカニズム自体は ADR-0036 に文書化されている。
- ソース: `docker/server/Dockerfile`、`.github/workflows/deploy-app.yaml`、`env/README.md`。
