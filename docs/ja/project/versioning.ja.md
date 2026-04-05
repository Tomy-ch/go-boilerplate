# バージョン管理方針（Versioning Policy）

プロジェクトは **Semantic Versioning（SemVer）** を採用しています。

- MAJOR
- MINOR
- PATCH

## バージョン定義

- **MAJOR**  
  破壊的変更（後方互換性を壊す変更）

- **MINOR**  
  後方互換性を維持した機能追加

- **PATCH**  
  バグ修正および非破壊的な改善

## リリースブランチ戦略

プロジェクトは **release-centric branching model** を採用しています。

- 機能開発は最新の `release/*` ブランチから分岐します
- `develop`, `staging`, `production` へは release 経由でのみ反映されます
- 保護ブランチへの直接コミットは禁止されています

## リリース手順

### タグ発行

以下のコマンドでタグを発行します：

```bash
make release-major-tag
make release-minor-tag
make release-patch-tag
```

タグの手動作成は禁止です。

### 次リリースブランチの作成

```bash
make release-major-branch
make release-minor-branch
make release-patch-branch
```

### Hotfix手順

緊急修正が必要な場合：

- `production` ブランチから `make hotfix-patch-branch` で `hotfix/*` ブランチを作成
- `hotfix/*` ブランチで修正を適用し、`production` へ取り込む
- 修正が取り込まれた `production` から `make release-patch-branch` で次の `release/*` ブランチを作成し、`develop` / `staging` / `production` へはその `release/*` ブランチ経由でマージ
- `release/*` ブランチが `production` にマージされたタイミングで PATCH バージョンのタグを発行（`make release-patch-tag`）

## 破壊的変更に関するルール

- 破壊的変更は MAJOR バージョンでのみ許可されます
- API契約変更は OpenAPI-first ポリシーに従う必要があります
- OpenAPI変更を伴う場合は必ずコード生成を行ってください

## 原則

- バージョン番号の直接編集は禁止
- タグは定義済みの make コマンド経由のみ
- ブランチ保護ルールに従うこと
- セマンティックバージョニングを厳守すること
