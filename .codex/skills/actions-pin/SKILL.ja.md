> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# GitHub Actions の pin 更新

外部 `uses:` 参照の末尾に付いたタグコメントを意図されたバージョンとして扱い、`.github/actions-pin.toml` を解決済み SHA の lockfile として扱う。

```yaml
uses: owner/repo@<40-hex-sha> # <tag>
```

`make pin-actions-resolve` が対象タグを lockfile へ解決し、`make pin-actions-apply` が SHA だけを更新する。**除外窓より新しい Action リリースへは、ユーザーが明示的に `days=0` を渡さない限り決して更新しないこと。**

隔離期間の判定に使う日付は、リリースの `published_at` と解決されたコミット日付の**新しいほう**である。したがって、リリースオブジェクトが古いというだけでは、貼り直されたばかりの moving タグが枯れて見えることはない —— リリース日は古いが head コミットが新しい moving タグは、依然として隔離される。隔離が買うのは自動化された乗っ取りに対する時間であり、貼り直しそのものを検出する手段は lockfile の diff をレビューすることのままである。根拠は `docs/design/security.md` の Build inputs。

## 入力

起動時のトークンは順不同で解釈する。

- 引数なし: 既存のメジャーバージョン内で更新する。最小経過日数は 14 日。
- `major` または `--major`: 新しいメジャーバージョンも検討対象にする。
- `<整数>` / `days=N` / `--days N`: 最小経過日数を設定する。負値は拒否する。`0` は隔離を無効化するため明示的に警告する。

このスキルを `mise.toml`・Go・Go モジュール依存の更新に使わないこと。ローカルの composite action（`uses: ./...`）は変更しないこと。

## ワークフロー

1. `AGENTS.md`・`.github/actions-pin.toml`・`.github/workflows/` と `.github/actions/` の外部 `uses:` 行をすべて読む。各 Action、その全出現箇所、現在のタグコメント、現在のメジャーを特定する。
2. pin ツールが動く状態かを確認する。ツールは `go run ./scripts/pin-actions` を使い、`go.mod` と整合した vendor ツリーを要求する。それだけが障害なら `go mod vendor` を実行する。可能なら認証済みの GitHub トークン（`gh auth token`）を得て、リリース日付の照会が匿名のレート制限に当たらないようにする。
3. Action ごとに、prerelease でないリリースとその公開日を照会する。moving なメジャータグを選ぶ場合は、それが実在することも確認する。候補の順位付けは `published_at` で続けてよいが、**古いリリースであることから枯れていると推論しないこと**。解決はリリース日とコミット日の新しいほうで判定される。
4. メジャー `M`（`major` が要求されていない限り現在のメジャー）に対して対象を選ぶ。
   1. moving タグ `vM` の解決された head が締切より古いなら、それを優先する。
   2. そうでなければ、締切より古い最新の厳密な `vM.x.y` リリースを選ぶ。
   3. そのメジャーに枯れたリリースが無ければ、現在の pin を維持し、理由を記録する。
   タグの綴りは上流のものをそのまま使う。プロジェクトによっては `0.x.y` と `v0.x.y` の間で表記が変わる。
5. メジャー更新を提案するものすべてについて、上流のリリースノートまたは `action.yml` を確認し、リポジトリ側の `with:` 入力をすべて突き合わせる。入力に非互換があればその Action は据え置く。**それに合わせて workflow の挙動を黙って変えないこと。**
6. 書き込む前に、具体的な計画を日本語で提示する。変更するもの（moving か厳密な pin か、リリースの経過日数を含む）、据え置くものとその理由、変更しないもの。提案した集合の確認をユーザーへ求める。**確認が取れるまで書き込まないこと。**
7. 確認が取れた項目についてのみ、末尾の `# <tag>` コメントを編集する。複数の Action が同じタグを共有する場合は `uses:` 行全体で一致させる。据え置き・変更なしのエントリは編集しない。
8. 解決と適用:

   ```sh
   export GITHUB_TOKEN="$(gh auth token)"
   make pin-actions-resolve PIN_ACTIONS_MIN_AGE_DAYS=<days>
   make pin-actions-apply
   ```

   `apply` は書き込み前に全対象の判定を確定させる。中断した場合、作業ツリーは手つかずのままである。リリース日とコミット日の新しいほうを見る規則の下で moving タグがまだ新しい場合、メジャー内の既存 pin を保つのは想定どおりの挙動である。したがって、古いリリースでも moving タグの head が新しいと `resolve` が `⚠️ ... 既存ピンを維持` と出すことがある。これは規則 4.1 → 4.2 のフォールバックが遅れて発火しただけであり、エラーではない。解決が moving タグの不在を報告した場合は、条件を満たす厳密なリリースを代わりに使う。
9. 検証:

   ```sh
   make pin-actions-check
   make actions-lint
   ```

   `check` と `apply` はリポジトリの状態の問題に対しても fail-closed する。再試行の前にローカルで直すこと。

   | エラー | 意味と対処 |
   | --- | --- |
   | `lockfile に解釈できない行があります`（行番号付き） | lockfile の行が、空行でもコメントでも `"key" = "<40-hex>"` の代入でもない。`make pin-actions-resolve` を実行するか、報告された行を削除する。 |
   | `lockfile にキーの重複があります` | 1 つの `owner/repo@tag` が 2 回代入されている。マージ衝突の解決後によく起きる。`make pin-actions-resolve` を実行するか、重複を削除する。 |
   | `lockfile に参照されていないエントリがあります` | lockfile のキーがどの生きた `uses:` にも一致しない。通常は workflow の削除後。`make pin-actions-resolve` を実行するか、孤児を削除する。 |
   | `固定対象として解釈できない記法の uses: があります` | pinner が書き換えられない `uses:`。フローマッピング（`- {name: Checkout, uses: actions/checkout@v4}`）、引用符付きキー（`"uses": ...`）、値を次行へ送るブロックスカラー（`uses: >-`）、YAML エイリアス（`uses: *anchor`）のいずれか。メッセージが該当する値を名指す。そのステップを素のブロック記法 `- uses: owner/repo@sha # tag` へ書き直すこと。**検査を抑制しないこと。** ブロックスカラーの内側のテキストは対象外なので、`run:` スクリプトが `uses: owner/repo@ref` という文字列を出力しても引っかからない。 |

   結果はそれぞれ報告する。失敗しても自動でロールバックしないこと。

## 安全性と完了

- このスキルの実行中に変更してよいのは `.github/workflows/*.{yml,yaml}` / `.github/actions/**/action.{yml,yaml}` / `.github/actions-pin.toml` だけである。
- `with:` の入力・workflow のステップの論理・`scripts/pin-actions`・生成ファイル・`AGENTS.md` は変更しない。
- stage / commit / push はしない。厳密バージョンへ後退させた pin は、moving タグが枯れたあとに見直せるよう報告する。
- `make pin-actions-check` と `make actions-lint` は必須の完了検査として扱う。
