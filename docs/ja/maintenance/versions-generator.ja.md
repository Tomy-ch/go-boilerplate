# Versions Generator — メンテナンスガイド

このドキュメントは、**Tools コンテナ（go_tool_runner / node_tool_runner）に生成ツールを追加したユーザー**が
「追加後に何をする必要があるか」を明確にするためのガイドです。

生成ツールはCI上で *GeneratedArtifactsCheck* により検査されており、
**生成物の差分がツールバージョンによるものか、単純な生成漏れなのか**を自動判定できるようになっています。

そのため、新規ツールの追加時には以下の対応が必須です。

## 1. `scripts/gen_tools_version.sh` にバージョン取得処理を追加する

ツールを追加したら、まず **このスクリプトにツールのバージョン出力処理を追記**してください。

例（Go 側にツール追加時）:

```sh
go_section() {
  echo "## Go-based tools (go_tool_runner)"
  echo "- golang-migrate: $(normalize \"$(migrate -version 2>&1 || echo 'unknown')\")"
  echo "- mockgen: $(normalize \"$(mockgen -version 2>&1 || echo 'unknown')\")"
  echo "- oapi-codegen: $(normalize \"$(oapi-codegen --version 2>&1 || echo 'unknown')\")"
  echo "- sqlc: $(normalize \"$(sqlc version 2>&1 || echo 'unknown')\")"

  # ★ ここに追加例
  echo "- your-tool: $(normalize \"$(<ツールのコマンド> <バージョン出力サブコマンド> 2>&1 || echo 'unknown')\")"
}
```

Node 側の場合は `node_section()` に追記してください。

## 2. `make gen-tools-meta` を実行してバージョン情報を更新する

ローカルで下記を実行してください：

```bash
make gen-tools-meta
```

これにより`docs/meta/generator-versions.txt`に記載されたバージョン情報が最新に更新されます。

## 3. 生成物の更新をコミットする

新規ツールの追加後、**必ず以下をコミット**してください：

- `docs/meta/generator-versions.txt`
- 生成されたコード（make gen の結果）

CIはこのファイルを基準にして差分を判定します。

## 4. CI上で生成物チェックが通ることを確認する

PRを作成すると、GitHubActionsの生成物チェックWF(/.github/workflows/gen-artifacts-check.yaml)が:

- 生成物の差分をチェック
- バージョン変更に起因する差分であればPRにコメント
- 最後に生成物 mismatchがあればCIをfail

という流れを実行します。

新しいツールを追加した際は、PR上でCIが成功するか必ず確認してください。

## なぜこれが必要なのか？

このプロジェクトでは生成物が多数存在します。

- oapi-codegenのGoコード
- sqlcのQuery/Model
- mockgenのモック
- swagger-cliのバンドル
- redoclyのHTML
- など…

これらはすべて **生成ツールのバージョンに強く依存**します。

そのため、

- 誰かがローカルの古いツールで生成 → 差分発生
- CIのコンテナが新しいツール → 差分検出
- 生成物の意図しない揺れ → PRが汚れる

という問題が起きやすくなります。

そこで、**生成ツールのバージョンをmanifestとして保持し、CIと同期**することで
「生成物の差分がツール更新によるものか」を自動判別できるようにしています。

## 要約：ツール追加時に必要な作業

|作業|必須|説明|
|------|------|------|
|`scripts/gen_tools_version.sh` に追記|必須|バージョン情報の出力|
|`make gen-tools-meta` を実行|必須|manifest の更新|
|生成物・manifest をコミット|必須|CI と同期|
|`make gen` で生成物更新|必須|追加ツールが影響する生成物がある場合|
|docker-compose にツールを追加|必要に応じて|新しいサービスが必要な場合|

## 🔚 おわりに

生成ツールが増えるほど、環境差異による生成物ブレも増えます。
このmanifestベースの仕組みは、その揺れを最小限に抑え、
**PRがきれいで、CIが安定し、再現性のある開発環境**を保つためのものです。

ツールを追加したときは、このREADMEを見て必要な作業を必ず行ってください 🎉
