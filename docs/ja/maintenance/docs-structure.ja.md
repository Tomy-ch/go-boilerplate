# ドキュメント構造ルール (Documentation Structure Rules)

このリポジトリでは **自動生成されるドキュメントポータル** を使用しています。

ポータル (`docs/portal`) は `docs.json` を読み込みます。  
このファイルは以下のスクリプトによって生成されます。

```txt
scripts/gen-docs-json.mjs
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
├ decisions.md
└ rules.md
```

これらのファイルは **Architecture (English)** セクションに表示されます。

## 2. 日本語ドキュメント

日本語ドキュメントは以下のディレクトリに配置します。

```txt
docs/ja
```

ファイル名のルール:

```txt
<name>.ja.md
```

例:

```txt
docs/ja
├ architecture.ja.md
├ development-flow.ja.md
├ decisions.ja.md
└ rules.ja.md
```

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

日本語版は同じディレクトリ構造を以下に作成します。

```txt
docs/ja/<section>
```

例:

```txt
docs/ja/project
├ policy.ja.md
├ scope.ja.md
└ versioning.ja.md
```

生成されるセクション:

```txt
Project (Japanese)
```

## 5. 予約ディレクトリ

以下のディレクトリは **generator で予約されており、通常のセクションとしては扱われません。**

```txt
docs/portal
docs/openapi
docs/coverage
docs/er-diagram
docs/ja
```

## 6. ポータル関連ファイル

以下のファイルは自動生成されるため、手動で編集してはいけません。

```txt
docs/portal/docs.json
```

再生成する場合:

```txt
node scripts/gen-docs-json.mjs
```

## 7. 新しいドキュメントセクションの追加

新しいセクションを追加する場合:

```txt
docs/security
docs/ja/security
```

例:

```txt
docs/security/auth.md
docs/ja/security/auth.ja.md
```

ポータルには自動的に以下が追加されます。

```txt
Security (English)
Security (Japanese)
```

## 8. CIとの連携

ドキュメントポータルは CI により自動更新されます。

```sh
node scripts/gen-docs-json.mjs
```

## まとめ

|配置場所|言語|
|--------|--------|
|docs/*.md|English|
|docs/ja/*.ja.md|Japanese|
|docs/<section>|English セクション|
|docs/ja/<section>|Japanese セクション|

これらのルールを守ることで、ドキュメントポータルを一貫した構造で維持できます。
