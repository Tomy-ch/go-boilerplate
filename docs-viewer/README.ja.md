# docs-viewer

[English](README.md) | 日本語

ドキュメントポータルのビューアーです。単体で成立する静的サイトで、生成物
`docs/portal/docs.json` を読んでカードと本文を描くだけの役割を持ち、内容の SSOT は持ちません。

ビルド成果物は `docs/portal/` 配下にコミットし、GitHub Pages が `docs/` をサイトルートとして
配信します（ADR 0100）。ビューアーのソースを `docs/` の外へ置くのは、パッケージ定義・lockfile・
`node_modules` を配信ツリーへ混ぜないためです。

## なぜ別パッケージなのか

**このリポジトリで唯一の JavaScript アプリケーションであり、browser 向けのツールチェーンを
必要とする唯一の場所だからです。** 既存の Node 依存（`docker/tools`）は生成系と lint を動かす
ためのもので、npm で解決して tool-runner イメージへ入れています。React アプリケーションの
依存グラフをそこへ混ぜると、コード生成のたびに browser フレームワークが供給面へ乗ります。

パッケージマネージャごと分ける（pnpm）ことで、ビューアーの依存はどこからも到達できなくなります。
分離を担保するのは規約ではなくパッケージ境界です。

## デザインシステムとの関係

UI は `nextjs-boilerplate` の design-system（`src/components/design-system`）から移植した部品で
組みます。ビューアーが実際に使う範囲を `src/components/` 配下へ取り込み、デザイントークンは
`src/tokens/tokens.css` に置いています。

**取り込んだ範囲はここで保守します。** このリポジトリにはトークンの生成元も Storybook も無いため、
移植の際に両者への参照は落としました。部品の挙動・アクセシビリティの契約・それらを固定する
テストはそのまま持ち込んでいます。

## 構成

| ディレクトリ | 役割 |
| --- | --- |
| `src/docs-json/` | 生成物 `docs.json` のスキーマと読み取り。形の不一致は配信事故として例外にする |
| `src/lang-filter/` | 表示言語での絞り込み。JA の実体が無い section は EN へ落とし、section 内で言語が混ざらないようにする |
| `src/search/` | 検索コーパスの組み立て。所属する section / group 名を項目へ畳み込む |
| `src/hash-route/` | 位置ハッシュ `#/<group>/<section>` の解釈と組み立て |
| `src/markdown/`・`src/sanitize/` | Markdown を sanitize 済みの hast へ変換する。sanitize を通した値だけが `SanitizedDocument` 型を持つ |
| `src/code-fence/`・`src/code-block/` | 木からコードフェンスを読み取り、強調表示付きで描画する |
| `src/mermaid-diagram/` | ` ```mermaid ` フェンスを図として描画する |
| `src/components/` | 移植した design-system の部品とデザイントークン |

## 本文の描画

本文は HTML 文字列を経由せず、hast の木から React 要素を直接作ります。コードフェンスのうち
2 種類だけを既定の `pre` から振り分けます。

- ` ```mermaid ` は図にする。mermaid は `securityLevel: "strict"` で動かし、配色は
  デザイントークンと同じ 2 経路（OS の設定と `data-theme`）から選ぶ
- それ以外のフェンスは highlight.js が強調表示する。highlight.js は入力を escape してから
  span で包む

どちらも遅延読み込みで、カード一覧の描画までの読み込みには含めません。highlight.js が知らない
言語のフェンスと、mermaid が解釈できない図は、落とさず素のテキストとして出します。

## コマンド

`mise.toml` が宣言する Node / pnpm のバージョンで動かすため、`make` 経由で tool-runner
コンテナ内で実行します。

| コマンド | 内容 |
| --- | --- |
| `make gen-portal-build` | ビューアーを `docs/portal/` へビルドする |
| `make portal-test` | テストを実行する |

手元の短い反復では、このパッケージで直接実行しても構いません（`pnpm dev` / `pnpm test` /
`pnpm typecheck`）。`pnpm dev` は hot reload 付きで配信しますが、隣に `docs.json` が無いと
表示するものがありません。

## 依存の方針

- **単独で完結する部品に寄せる。** このビューアーは別リポジトリからの移植であり、今後も移植され
  得ます。引き込んだ依存はそのまま移植コストになります。対に `-native` / `-client` がある部品は、
  要件が許す限り `-native` を優先します
- **純粋ロジックは zod 以外に依存させない**（`docs-json` / `lang-filter` / `search` /
  `hash-route` / `code-fence`）。そのまま持っていける状態を保ちます
- **バージョンは完全固定**し、供給網の設定は `pnpm-workspace.yaml` が宣言します。公開から 7 日
  未満の版は解決しない、宣言の無い依存ライフサイクルスクリプトは拒否する、レジストリ以外の
  取得元を拒否する、の 3 点です
