> **このファイルは `SKILL.md` の日本語訳（ヒューマン参照用）です。**
> 直接編集しないでください。内容の変更が必要な場合は canonical な `SKILL.md`（英語版）を更新し、その後この日本語訳を同期してください。
> Claude Code のスキルとしては `SKILL.md` のみが読み込まれます。このファイルはスキル本体ではなく、レビューや学習用の翻訳ドキュメントです。

# Go バージョンアップグレード手順

このスキルは Go のバージョンを任意のバージョンへアップグレードするための作業手順を定義する。

正式な手順書は以下を参照すること。

- `docs/maintenance/go-upgrade.md`

## 最初に行うこと: 対象バージョンの確認

このスキルでは、**スキル起動直後に必ず `AskUserQuestion` で対象バージョンを確認する**。
スキル引数や直近メッセージにバージョンらしき文字列があっても、それを採用して即実行に進んではならない（誤指定を防ぐため、明示的な確認を必須とする）。

手順:

1. `.go-version` を読み、現行バージョンを把握する。
2. **必ず** `AskUserQuestion` を呼び出して以下を確認する。
    - 質問: 「アップグレード先の Go バージョンを指定してください（例: `1.26.3`）」
    - 補足として、現行バージョン（`.go-version` の値）を提示する。
    - スキル引数や直近メッセージにバージョン候補がある場合は、それを質問文に「候補: `X.Y.Z`」として併記し、ユーザーに確定させる。
3. 受け取った回答が `X.Y.Z` 形式であることを軽く検証し、以下の手順内で `<TARGET_VERSION>` として使用する。

確認が取れるまで、ファイル更新やコマンド実行は行わないこと。

## 前提

- 対象バージョン: `<TARGET_VERSION>`（上記で確認した値）
- 直接 `production` / `develop` / `staging` / `release/*` ブランチで作業しないこと（AGENTS.md の Git ルール参照）
- 作業ブランチを `release/*` の最新から切ること（例: `feature/go-upgrade-<TARGET_VERSION>`）

## AI Modification Scope について

このスキルは AGENTS.md の "Exception: Skill Execution" 節に基づき、スキル実行中に限り通常の AI Modification Scope の縛りを解放する。具体的には以下のパスへの変更がスキル実行中に許可される:

- `.go-version`
- `go.mod`, `go.sum`, `vendor/`
- `docker/**/Dockerfile`（および関連 README のバージョン表記）
- `.github/workflows/**`（Go バージョン直書きがある場合のみ）

ただし以下は引き続き保護対象（スキル実行中でも変更不可）:

- `AGENTS.md` / `CLAUDE.md`
- 生成ファイル（`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, `docs/` の生成物）

## 実行ステップ

`docs/maintenance/go-upgrade.md` のステップに対応する形で以下を実施する。コマンド・ファイル内に現れる `<TARGET_VERSION>` は、ユーザーが指定したバージョンに置換すること。

### 1. リリースノートの確認

<https://go.dev/doc/devel/release> で `<TARGET_VERSION>` のリリースノートを確認する。

確認観点:

- 言語仕様の変更
- 標準ライブラリの破壊的変更
- `go vet` の挙動変更
- ツールチェインの変更

### 2. `.go-version` の更新

```text
<TARGET_VERSION>
```

### 3. ローカル Go 環境の更新

goenv を使用している場合:

```sh
goenv install <TARGET_VERSION>
goenv local <TARGET_VERSION>
go version
```

Homebrew を使用している場合:

```sh
brew update
brew upgrade go
go version
```

ローカル環境の更新はユーザー作業となるため、AI エージェントは実行せずユーザーに依頼すること。

### 4. CI の Go バージョン更新

`.github/workflows` 配下を確認する。`go-version-file: go.mod` を使用していれば直接の修正は不要。バージョンを直接指定している箇所があれば `<TARGET_VERSION>` に揃える。

### 5. `go.mod` の Go バージョン更新

```sh
go mod edit -go=<TARGET_VERSION>
```

### 6. 依存関係と vendor の更新

```sh
make tidy-lib
```

（内部で `go mod tidy` と `go mod vendor` を実行）

### 7. Go ツールの再インストール

```sh
make install-tools
```

### 8. Dockerfile の更新

`docker/` 配下の Dockerfile で `golang:X.Y.Z` を参照している箇所を `<TARGET_VERSION>` に更新する。同様の表記が `docker/**/README.md` 内にあれば併せて更新する。

### 9. Docker コンテナの再ビルド

Go バージョンアップでは base image タグが変わるため、新しいイメージを確実に pull・再ビルドできるよう `-clean`（`--no-cache --pull`）バリアントを使う:

```sh
make serve-build-clean
make tools-build-clean
```

### 10. コード生成の再実行

```sh
make gen
```

### 11. テストの実行

```sh
make test
```

### 12. lint の実行

```sh
make lint
```

### 13. 最終確認

以下が全て成功することを確認する。

```sh
make tidy-lib
make install-tools
make gen
make test
make lint
make serve-build-clean
make tools-build-clean
```

## チェックリスト

完了報告時に以下を確認すること。

- [ ] 対象バージョン `<TARGET_VERSION>` を `AskUserQuestion` でユーザーに確認済み
- [ ] リリースノート確認
- [ ] `.go-version` を `<TARGET_VERSION>` に更新
- [ ] ローカル Go の更新（ユーザー作業）
- [ ] CI の Go バージョン確認・更新
- [ ] `go.mod` の Go バージョンを `<TARGET_VERSION>` に更新
- [ ] `make tidy-lib` 実行
- [ ] `make install-tools` 実行
- [ ] Dockerfile の更新
- [ ] Docker コンテナの再ビルド
- [ ] コード生成の再実行
- [ ] テスト実行
- [ ] lint 実行

## 注意事項

- 生成コード（`**/*.gen.go`、`*.sql.go`、`*_mock.go` など）は手動で編集しないこと
- コミットは作業ブランチで行い、`production` / `develop` / `staging` / `release/*` への直接コミットは禁止
- PR への push は明示的にユーザーから指示があった場合のみ実行すること
