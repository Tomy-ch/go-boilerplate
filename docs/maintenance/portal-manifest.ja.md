# ドキュメントポータル — Manifest 契約

[English](portal-manifest.md) | 日本語

`docs/portal/` のドキュメントポータルは明確な **契約** に従って動いています。すなわち、目に見える構造 (どのグループが存在し、各セクションを何と呼び、どの順序で並べ、サイドバーの Reference ブロックに何を出すか) は `docs/portal/manifest.yaml` で定義され、`scripts/` 配下のビルドスクリプトはその manifest を読み、加えてドキュメントのファイルシステムを走査してポータル用データを **構築する** だけです。

```txt
manifest.yaml   ← 構造定義   (どこに何を、どんなラベルで)
scripts/portal/*.ts ← 構築       (それをどう組み立てるか)
```

このドキュメントは、その契約の単一リファレンスです。ドキュメントを追加・移動・改名・削除するとき、実際に編集すべき場所はほとんどの場合 `manifest.yaml` です。

## 1. 関連ファイル

| ファイル | 役割 |
| --- | --- |
| `docs/portal/manifest.yaml` | ポータル構造の単一源泉 |
| `docs-viewer/src/**` | React ビューアーのソース（`docs.json` を読む）。Vite でビルドする |
| `docs/portal/index.html` + `dist/**` | **生成物**。`docs-viewer` のビルド出力（`make gen-portal-build`）。どちらも gitignore 対象で、デプロイのたびに再生成される |
| `docs/portal/docs.json` | **生成物**。直接編集禁止 (`gen-docs-json.ts` の出力) |
| `docs/portal/guides/**` | **生成物**。直接編集禁止 (`gen-portal-docs.ts` による README のフラットコピー) |
| `scripts/portal/gen-portal-docs.ts` | manifest の各エントリ `src` を `docs/portal/guides/` 配下の `dst` へコピー |
| `scripts/portal/gen-docs-json.ts` | manifest 読み込み + `docs/` 走査により `docs/portal/docs.json` を出力 |

## 2. manifest スキーマ

`docs/portal/manifest.yaml` は 2 つの部分から成ります。

1. ポータルの **可視構造** を定義するトップレベルの `meta:` ブロック。
2. キーが section id、値が `{src, dst}` コピーペアの配列となる、トップレベル **セクションエントリ**。

### 2.1. `meta:` ブロック

```yaml
meta:
  groups:           # サイドバー上のトップレベルページ群を順序付きで定義
    - title: "<ページタイトル>"
      sections: [<section_id>, <section_id>, ...]   # この順序でページ内にレンダリング

  subgroups:        # 任意: セクションを役割サブグループに分割
    <section_id>:
      - title: "<サブグループ名>"
        items: [<guide_id>, <guide_id>, ...]        # guide_id = guides/<id>.md から拡張子を除いた basename

  section_titles:   # 任意: 自動派生されるセクション表示名を上書き
    <section_id>: "<表示名>"

  reference_links:  # サイドバー Reference ブロックに代表 item を出す section id 群 (順序維持)
    - <section_id>
    - <section_id>
```

| キー | 用途 | 必須 |
| --- | --- | --- |
| `meta.groups` | どのトップレベルページが存在し、各ページにどの section が属するかを定義。ここの順序がサイドバーの並び順になる。 | 必須 |
| `meta.subgroups` | セクションを役割サブグループ (例: `Layer Top / HTTP Stack / Error Response`) に細分化。未掲載 item は末尾に自動追加される `Other` サブグループに集約される。 | 任意 |
| `meta.section_titles` | section id から自動派生されるデフォルト見出しを上書き。略語の整形 (`DI`, `CLI`, `DB Schema`, `OpenAPI Reference` ...) や、より親しみやすい名前を与えるのに使う。 | 任意 |
| `meta.reference_links` | セクションの **代表 item 1 個** をサイドバーに常設クイックリンクとして出す section id 群。生成 HTML (godoc / db-schema / coverage / openapi) 用。これらのセクションは通常のグループ/ページ動線からは除外される。 | 任意 |

### 2.2. セクションエントリ

`meta` 以外のトップレベルキーはすべて section id です。値はコピーペア配列です。

```yaml
<section_id>:
  # English
  - src: <リポジトリ内のソースパス>
    dst: docs/portal/guides/<flat-name>.md
  - src: <別の EN ソース>
    dst: docs/portal/guides/<another>.md
  # Japanese
  - src: <ソースパス>.ja.md
    dst: docs/portal/guides/ja/<flat-name>.ja.md
  - ...
```

`src` はリポジトリ内の正本 README、`dst` はビルドで `docs/portal/guides/` 配下にコピーされるパスです。ビューアーは実行時に `dst` をフェッチし、カードの小さな出所表示には `src` が使われます。

**`dst` の命名規則**: 拡張子 (`.md` / `.ja.md`) を除いた basename が `meta.subgroups` から参照される **guide id** になります。guide id はポータル全体で一意にしてください。

## 3. ファイルシステム auto-discovery (TypeScript 構築側)

`docs/` 直下のドキュメント (コードパッケージの `**/README.md` ではなく、`docs/` 内に直接置かれる Markdown) は `gen-docs-json.ts` がファイルシステムを走査して検出します。発見されたセクションでも、配置と表示名は依然 `meta:` から来ます。**ファイルの列挙だけ**が TypeScript 側です。

| ファイルシステム位置 | 作られる section | 補足 |
| --- | --- | --- |
| `docs/*.md` | section id `architecture` | ルート直下のアーキ系ドキュメント (rules / decisions / development-flow / ...) |
| `docs/*.ja.md` | section `architecture` の item (lang: ja) | 日本語版 |
| `docs/<dir>/*.md` | section id `<dir>` | 任意のサブディレクトリで自動。例: `docs/maintenance/*.md` → section `maintenance` |
| `docs/<dir>/*.ja.md` | section `<dir>` の item (lang: ja) | 日本語版 |
| `docs/<dir>/index.html` | section id `<dir>`、HTML item 1 個 (lang: all) | 生成リファレンスサイト (godoc / coverage / ...) 用 |

auto-discovery で見つかったセクションの配置をコントロールするには、`meta.groups` から id を参照してください (配置)。必要に応じて `meta.section_titles` (表示名) / `meta.subgroups` (細分化) / `meta.reference_links` (クイックリンク化) も使えます。

`meta:` から参照されているが対応する manifest エントリも FS 位置も無い section id は `⚠` 警告でスキップされます。FS から発見されたが `meta.groups` 未掲載 (かつ `meta.reference_links` にも無い) 場合は、フォールバックとして末尾の `Uncategorized` グループに集約されます。

## 4. よくある変更パターン

### 4.1. 新しい README をポータルに載せる

コードパッケージの README (`internal/<layer>/<sub>/README.md`, `pkg/<sub>/README.md` ...) はこの方法で表示されます。

1. 所属する section を決める (例: `controller`, `infrastructure`)。
2. その section に EN + JA ペアを追加する。

   ```yaml
   controller:
     # English
     - src: internal/controller/<new-package>/README.md
       dst: docs/portal/guides/controller-<new-package>.md
     # Japanese
     - src: internal/controller/<new-package>/README.ja.md
       dst: docs/portal/guides/ja/controller-<new-package>.ja.md
   ```

3. その section が `meta.subgroups` を使っているなら、新しい guide id (`controller-<new-package>`) を該当サブグループの `items:` に追加する。書かなければ `Other` に入る。
4. `make gen-portal-docs && make gen-docs-json` を実行。

### 4.2. 新しいセクションを追加する

ソースの置き場所で 2 パターン。

- **ソースがコードの README** (`internal/<x>/README.md` 等) → 新規 section を `manifest.yaml` のトップレベルキーとして追加し、`meta.groups` の該当グループに section id を加える。
- **ソースが `docs/<new-dir>/` 配下の Markdown** → ファイルを置くだけで auto-discovery が拾う。`meta.groups` に `<new-dir>` を追加して配置を決め、必要なら `meta.section_titles` で見出しを整える。

### 4.3. ドキュメントを別グループへ移す

`meta.groups` のみ編集する。section id は変わらず、片方の `sections:` リストから抜いてもう片方に入れるだけ。

### 4.4. セクション見出しを改名する

`meta.section_titles.<section_id>` を追加・変更する。id 自体は変更しない。

### 4.5. 肥大セクションを役割で分割する

`meta.subgroups` にエントリを追加する。

```yaml
meta:
  subgroups:
    <section_id>:
      - title: "<役割 A>"
        items: [<guide_id>, <guide_id>]
      - title: "<役割 B>"
        items: [<guide_id>]
```

未掲載 item は自動で末尾の `Other` サブグループに入るので、部分的な記述で安全に運用できる。

### 4.6. 新しい Reference リンク (生成 HTML) を追加する

1. ソース生成器の出力先を `docs/<name>/index.html` にしておく。
2. `meta.reference_links` に `<name>` を順序付きで追加する。
3. 必要に応じて `meta.section_titles.<name>` で表示名を上書きする。

HTML 自体は `docs/portal/guides/` には **コピーされない** ― リンクは元の位置を直接指し、新タブで開く。

## 5. ビルドコマンド

| コマンド | 内容 |
| --- | --- |
| `make gen-portal-docs` | manifest エントリの `src` → `dst` を全てコピー。ソース存在チェックと `dst` の出力ディレクトリ外参照チェックも行う。 |
| `make gen-docs-json` | `manifest.yaml` 読み込み + `docs/` 走査で `docs/portal/docs.json` を書き出す。 |
| `make gen-docs` | 上記 2 つに加え OpenAPI redocly ビルドを実行する。 |

`*-ci` 系 (`make gen-portal-docs-ci`, `make gen-docs-json-ci`) は Docker ツールランナーを介さず Node スクリプトを直接実行する。CI 向け。

## 6. ローカルプレビュー

任意の静的 HTTP サーバーで `docs/` を配信し、ポータルを開く:

```sh
# 手元にある適当な静的サーバーで
python3 -m http.server 8082 -d docs
# 開く
open http://localhost:8082/portal/
```

ポータルは CDN 経由でロードされる React SPA。ビルドステップは無い。`main.jsx` / `styles.css` を編集してページをリロードすればイテレーションできる。

## 7. アンチパターン

- **`docs/portal/docs.json` や `docs/portal/guides/**` を直接編集しない**。全て再生成されるので消える。
- **manifest を迂回して** `gen-docs-json.ts` に直接セクションを書き込まない。構造に関わるものはすべて `meta:` に。
- **サブグループの中にさらにサブグループを切らない**。あるサブグループが大きくなり過ぎたら、その親 section を分割する。
- **2 つの manifest item に同じ `dst` を与えない**。guide id は一意である必要がある。

## 8. まとめ

| 関心事 | 出所 |
| --- | --- |
| どんなページが存在するか (グループタイトルと順序) | `manifest.yaml` → `meta.groups` |
| 各ページにどの section が乗るか (順序付き) | `manifest.yaml` → `meta.groups[].sections` |
| section 内の役割サブグループ | `manifest.yaml` → `meta.subgroups` |
| section 見出しの上書き | `manifest.yaml` → `meta.section_titles` |
| サイドバー Reference クイックリンク | `manifest.yaml` → `meta.reference_links` |
| `guides/` に何をコピーするか | `manifest.yaml` の flat エントリ |
| `docs/<dir>/` / `docs/*.md` のファイル列挙 | `scripts/portal/gen-docs-json.ts` (FS スキャン) |
| ファイル名 → カード見出し (`autoTitle`) | `scripts/portal/gen-docs-json.ts` (決定的) |
| EN/JA の並び順 / slug 化 | `scripts/portal/gen-docs-json.ts` (決定的) |

変更が表の上半分に触れるなら **manifest 側**、下半分に触れるなら **スクリプト側** で対応する。この責務分担により、manifest は一目で読み切れる規模に保ちつつ、README が増えても自動で追従するビルドを維持できる。
