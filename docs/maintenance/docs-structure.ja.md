# ドキュメント構造ルール (Documentation Structure Rules)

このプロジェクトでは **自動生成されるドキュメントポータル** を使用しています。

ポータル (`docs/portal`) は `docs.json` を読み込みます。  
このファイルは以下のスクリプトによって生成されます。

```txt
scripts/portal/gen-docs-json.ts
```

この生成処理が正しく動作するよう、以下のルールを守る必要があります。

## 1. ルートドキュメント

英語版ドキュメントは以下に配置します。

```txt
docs/
```

例:

```txt
docs/
├ architecture.md
├ development-flow.md
├ decisions.md   # リダイレクト stub → adr/
├ rules.md
├ adr/    # アーキテクチャ決定記録（per-file ADR）
└ reference/dependencies.md
```

これらのファイルは **Architecture (English)** セクションに表示されます。

## 2. 日本語ドキュメント

日本語ドキュメントは**英語正典の隣**に `<name>.ja.md` として置きます。言語を分けているのは接尾辞で
あって置き場所ではなく、専用のディレクトリは持ちません。generator も接尾辞で振り分けます。

これらのファイルは **Architecture (Japanese)** セクションに表示されます。

## 3. セクションドキュメント

サブディレクトリを作成すると、自動的にドキュメントセクションが生成されます。

例:

```txt
docs/project
docs/maintenance
```

ディレクトリ内の Markdown ファイルがページとして扱われます。

例:

```txt
docs/project
├ policy.md
├ scope.md
└ versioning.md
```

生成されるセクション:

```txt
Project (English)
```

## 4. 日本語セクションドキュメント

セクションの日本語ドキュメントは、そのセクションのディレクトリに `<name>.ja.md` として置きます。
他に必要なものはありません。generator が接尾辞で見つけ、次のセクションへ振り分けます。

生成されるセクション:

```txt
Project (Japanese)
```

## 5. 予約ディレクトリ

generator が非セクションとして扱うのは次の 1 つだけです。

```txt
docs/portal
```

`docs/openapi`・`docs/coverage`・`docs/db-schema`・`docs/godoc` は通常のセクションとして走査され、
`meta.reference_links` 経由でクイックリンクとして現れます。契約の詳細は
[`portal-manifest.ja.md`](portal-manifest.ja.md) を参照してください。

## 6. ポータル関連ファイル

以下のファイルは自動生成されるため、手動で編集してはいけません。

```txt
docs/portal/docs.json
```

再生成する場合:

```txt
make gen-docs-json
```

## 7. 新しいドキュメントセクションの追加

新しいセクションを追加する場合:

```txt
docs/security
```

例:

```txt
docs/security/auth.md
docs/security/auth.ja.md
```

ポータルには自動的に以下が追加されます。

```txt
Security (English)
Security (Japanese)
```

## 8. CIとの連携

ドキュメントポータルは CI により自動更新されます。

```sh
make gen-docs-json
```

## まとめ

|配置場所|言語|
|--------|--------|
|docs/*.md|English|
|docs/*.ja.md|Japanese|
|docs/<section>/*.md|English セクション|
|docs/<section>/*.ja.md|Japanese セクション|

これらのルールを守ることで、ドキュメントポータルを一貫した構造で維持できます。
