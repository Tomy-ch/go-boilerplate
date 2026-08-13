> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# アーキテクチャ適合チェック

このスキルはアーキテクチャに焦点を当てたレビューのためのものであり、整形・一般的なコードレビュー・仕様検証のためのものではない。

## スコープと書き込み権限

既定の対象は変更された production の Go ファイルで、生成ファイル・モック・テストは除く。プルリクエストが既にあるなら、その `baseRefName` が権威である。無ければ base を `make base-branch`（`origin` の生の状態から得た最新の `release/*`）で解決する。`gh repo view --json defaultBranchRef` は決して使わない。base を解決できない場合は止め、その失敗を日本語で報告する。変更された production ファイルが無かった場合とは区別して報告すること。

リポジトリ全体を対象にするのは、そう要求されたときだけである。1 つ以上の層を名指しした要求は、監査をその層に限定する。

| パス | 層 | エージェント |
| --- | --- | --- |
| `internal/domain/` | domain | `.codex/agents/arch-auditor-domain.toml` |
| `internal/usecase/` | usecase | `.codex/agents/arch-auditor-usecase.toml` |
| `internal/controller/` | controller | `.codex/agents/arch-auditor-controller.toml` |
| `internal/infrastructure/` | infrastructure | `.codex/agents/arch-auditor-infra.toml` |
| `pkg/` | 共有パッケージ | `.codex/agents/arch-auditor-pkg.toml` |
| `internal/domain/` / ADR / README 群 | DDD 解釈 | `.codex/agents/ddd-origin-auditor.toml`（`ddd-audit`、`quick` スコープのプリセット） |

**このスキルは既定で読み取り専用である。** 元のスキルがその選択肢を提示していたというだけの理由で `TODO` コメントを足さないこと。足すのはユーザーの明示的な承認を得たあとだけであり、修正すべき violation に対して足すことは決してない。

## 手順

1. `AGENTS.md` を読み、対象のファイルと層を解決する。列挙された層の外にある変更 Go ファイルは別途報告する。監査したふりをしないこと。
2. `make lint` を 1 回だけ実行する。その findings を適切な層へ割り当てる。無関係なエラーで基準そのものが壊れている場合は、失敗を報告し、意味論的なレビューへは進まない。
3. 対象の各層について、上のスコープ表が名指すエージェント定義・層の README・該当する最も近いパッケージ README を読む。エージェント定義が与えるのは監査の役割と出力の契約であり、リポジトリの規則についての source of truth は `AGENTS.md` と README のままである。
4. 層ごとの観点をレビューする。

   - **domain:** 純粋性。依存は内向きのみ。フレームワーク・永続化・I/O・時刻 / 乱数 / ID 生成の直接呼び出し・不適切な context の使用が無いこと。エンティティとテーブルの対応は advisory な検査として扱い、値オブジェクトによる表現やメソッド形式の計算値を許容する。
   - **usecase:** オーケストレーションとトランザクションの所有。依存は domain のインターフェースのみ。時刻・乱数・外部 I/O は boundary の抽象を経由すること。domain に属する業務不変条件を持たないこと。
   - **controller:** OpenAPI の operation とハンドラの対応。リクエスト / レスポンスの適合のみ。インフラの import や業務のオーケストレーションが無いこと。ハンドラの大きさや自明でない分岐は、明文化された規則に反しない限り suggestion として扱う。
   - **infrastructure:** domain インターフェースの実装。データのオーケストレーションのみ。生成クエリの使用とエラー正規化を RDB / `pgerror` の README に従って行うこと。Repository とクエリの 1:1 対応は advisory として扱う。JOIN・複数クエリの操作・ディスパッチはいずれも妥当だからである。
   - **pkg:** `internal/` へ依存しないこと。フレームワーク非依存で再利用可能であること。feature 固有の業務ロジックを持たないこと。サブパッケージ README に明示された例外があればそれを尊重する。

5. 委譲が使えるなら、独立した層を上のスコープ表が名指すエージェントの役割へ委譲する。使えない場合はその指示をインラインで実行する。**`make lint` を 2 回以上実行しないこと。** 監査対象の変更が domain のコードまたは ADR / README 群に触れている場合は、`.codex/agents/ddd-origin-auditor.toml` を通して `ddd-audit` を `quick` スコープのプリセットで併せて実行する。このときスコープを再度尋ねさせてはならない。これは層ごとの監査とは分けて扱うこと。リポジトリが文書化した DDD の解釈を Evans と比較し、乖離を人間へ提起するものであって、裁定も修正も決して行わない。
6. 次の形式で日本語で報告する。

   ```text
   arch-check 結果（スコープ: <changed | full | layers>）

   [lint]
   - <file:line>: <message>

   [<layer>]
   - violation: <file:line> — <problem>
     source: <AGENTS.md or README location>
     remediation: <action>
   - suggestion: <file:line> — <advisory>
     source: <location>

   総計: violations <n>, suggestions <n>
   ```

   **findings を捏造しないこと。** 問題が無ければ、監査した層を名指したうえで violation が見つからなかったと述べる。

## 任意の TODO 引き渡し

明示的な承認を得たあとにのみ、`internal/domain` / `internal/controller` / `internal/infrastructure` の suggestion に対して、素の日本語の `// TODO:` コメントを足す。

挿入前に直前の 3 行を確認する。既にコメントブロックがあるならスキップする。観測した逸脱と、人間に委ねる判断を記述する。AI 専用のマーカーは使わないこと。violation に対して TODO を書かないこと。

## 完了時の制約

- 自動修正・stage・commit・push はしない。
- findings ごとに、それを統べる README または `AGENTS.md` を引用する。
- 生成ファイルには触れない。
- 任意の引き渡しが承認された場合は、TODO を足した箇所とスキップした箇所を分けて報告する。
