---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy, image, exclusion, setup-review]
---

# ADR-0097: ハードニング Alpine をランタイムベースとして使用し、distroless/scratch は使用しない

English canonical: [0097-hardened-alpine-runtime.md](0097-hardened-alpine-runtime.md)

## ステータス

accepted

## 背景

最小ランタイムベースイメージ（distroless、scratch）は、Go がスタティックバイナリにコンパイルされランタイムに OS レイヤーを技術的には必要としないため、Go サービスによく提案される。アピールは攻撃サーフェスの削減と小さなイメージにある。

しかし、ここでのランタイムイメージは 3 つの具体的な要件を満たさなければならない:

1. **TLS トラスト** — アウトバウンド HTTPS 呼び出しには信頼された CA 証明書バンドルが必要である。
2. **タイムゾーンデータ** — タイムゾーン対応のビジネスロジック（`tzdata`）が正しく解決されなければならない。
3. **セキュリティパッチ** — ベースレイヤーはアプリケーションバイナリ全体を再ビルドすることなく、ディストリのパッケージマネージャーを通じて独立してパッチ適用できなければならない。

Distroless イメージと scratch はパッケージマネージャーも独立して更新可能な `tzdata` パッケージも提供しない。scratch に `ca-certificates` と `tzdata` を追加するには、それらをバイナリに埋め込むか手動でコピーする必要があり、シンプルさの議論を排除して将来の更新を困難にする。

## 決定

ランタイムベースイメージとして distroless または scratch を使用しないことを意図的に決定する。

`runtime` ステージは `alpine:3.23` をベースとする。イメージのセットアップ:

1. ビルド時に `apk upgrade --no-cache` を実行してアップストリームのセキュリティパッチを適用する。
2. `ca-certificates`（TLS トラスト）と `tzdata`（タイムゾーンデータ）のみをインストールする — 唯一 2 つの OS レベルのランタイム依存関係。
3. 専用の非 root グループとユーザー（`app:app`）を作成し、`COPY` と `CMD` 命令の前にそのユーザーに切り替え、バイナリが root として実行されないようにする。

この ADR をレビューするセットアップチームは、`alpine` ピン（`3.23`）を更新すべきかどうか、および再現性のためにベースイメージダイジェストを追加すべきかどうかを確認すること（Dockerfile のノートはプロダクションではダイジストピニングを推奨している）。

## 影響

### ポジティブな影響

- ビルド時の `apk upgrade` により、新しい Alpine マイナーリリースを待つことなく OS レイヤーを最新の状態に保てる。
- `ca-certificates` と `tzdata` はバイナリに埋め込まれるのではなくディストリによって管理されるため、独立して更新できる。
- 非 root 実行によりコンテナエスケープの爆発半径が削減される。

### ネガティブな影響

- Alpine は distroless や scratch より大きなベースである。スタティックリンクされたバイナリに対して差分は数メガバイトである。
- Alpine は musl libc を使用するが、ビルダーで CGO は無効（`CGO_ENABLED=0`）であるため、実際には musl の互換性問題は生じない。

## 検討した代替案

### distroless/static または distroless/base

シェルなし、小さなサーフェス — しかしパッチ適用のための `apk` なし、便利な `tzdata` パッケージなし、`ca-certificates` は手動コピーが必要。CGO がすでに無効なため、意味のあるセキュリティ上の利点なしに将来のメンテナンスを複雑化させる。

### scratch

絶対最小サイズ。CA バンドルなし、タイムゾーンデータなし、パッケージマネージャーなし。最新の状態を保つことが困難な手動コピーステップを必要とし、ランタイムで TLS を呼び出したりタイムゾーンを解決するコードパスを壊す。

## 補足

- `docker/server/Dockerfile` 42-53 行が `runtime` ステージを定義する。
- `docker/server/README.md` §"runtime" が非 root ユーザーのセットアップと 2 つの OS レベルパッケージを説明する。
- プロダクションデプロイはベースイメージをダイジェストでピニングすべきである（Dockerfile ヘッダーに記載）。
- ソース: `docker/server/Dockerfile`、`docker/server/README.md`。
