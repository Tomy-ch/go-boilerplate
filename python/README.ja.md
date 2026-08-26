# python

> このドキュメントは [README.md](README.md) の日本語訳です。内容の更新は英語版から反映してください。

このリポジトリが PyPI から入れる CLI ツールの、バージョン宣言と lockfile を置く場所です。アプリケーションのコードはここにはなく、リポジトリは Python のソースを一切持ちません。ここにあるのは、たまたま Python パッケージとして公開されているビルド時のツールです。

## なぜ `mise.toml` に書かないのか

他のツールのバージョンはすべて `mise.toml` にあります（[ADR-0080 (mise-ssot-drift-gate)](../docs/adr/0080-mise-ssot-drift-gate.md)）。PyPI のツールだけが例外なのは、バージョンを固定してもほとんど何も固定できないからです。依存は install 時に解決されるため、同じ pin でも日によって違うツリーが入り、バージョンの pin を lockfile として読めるスキャナも存在しません。

そのため、ツールごとに 2 つのファイルを持ちます。

|ファイル|役割|
|---|---|
|`<tool>.in`|宣言。`==` の pin 1 行と、その版である理由|
|`<tool>.txt`|解決結果。推移依存まで含めた全パッケージを sha256 付きで固定したもの|

install は常に `uv pip install --require-hashes -r <tool>.txt` です。版かハッシュを欠いた要求はここで拒否されるため、検証は install の一部であって、省略できる別の手順ではありません。

## ツールごとに 1 組

ここにあるのはエコシステムを共有するだけの無関係な CLI で、それぞれ別の環境へ入ります。1 組ずつ分けておけば解決も互いに独立し、2 つの間の依存衝突はそもそも起こりません。`sqlfluff` しか要らない Docker イメージが、もう一方のツリーを抱え込むこともありません。

## バージョンを変えるとき

`<tool>.in` の `==` の pin を書き換えてから、再生成します。

```bash
make py-lock
```

2 つのファイルは互いに突き合わされます。宣言と lockfile が違う版を指していれば `make tool-cooldown-audit`（および pull request 時の同じゲート）が失敗します。この検査が無いと、`.in` を上げて再生成を忘れたときに、実際には入らない版に対して cooldown が通ってしまいます。

新しい版は供給網のクールダウンの対象でもあります。PyPI の窓はパッケージレジストリ共通の 7 日です。最新より前の版で止めている場合は、その理由を `.in` に書きます。

## どこが install するか

|利用側|lockfile|
|---|---|
|`python_tools` イメージ（`docker/tools/Dockerfile`）。`make sql-lint` / `make sql-fix` が使う|`sqlfluff.txt`|
|SQL Lint ワークフロー（`.github/workflows/sql-lint.yaml`）|`sqlfluff.txt`|
|`.claude/scripts/bootstrap-external-skills.sh`|`graphify.txt`|

lockfile を解決する対象の Python ランタイムは `mise.toml` が宣言しているものです。`make py-lock` は実行中のインタプリタではなく、そこから読み取ります。
