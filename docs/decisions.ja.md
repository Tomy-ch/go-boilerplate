# アーキテクチャ決定事項

> **移動しました。** この一枚岩の決定ドキュメントは、決定ごとに 1 ファイルの
> **アーキテクチャ決定記録（ADR）** に分割されました。完全な索引付き決定ログは
> **[`docs/adr/`](adr/README.ja.md)**（不変の記録 1 決定 = 1 ファイル）を参照してください。
> 英語正典: [decisions.md](decisions.md)

かつてここにあった技術選定の根拠（オニオンアーキテクチャ、OpenAPI-first、SQL-first、sqlc、
Echo、Fx、worker scaffold、ライブラリ選定ポリシー、observability gating）は、個別の ADR に
なりました。

- 全体の一覧と順序は [ADR ログ](adr/README.ja.md) から。
- かつてインラインにあった**直接依存表**は決定ではなくコード追従の「生きた目録」であり、
  [`docs/reference/dependencies.md`](reference/dependencies.md) へ移動しました
  （欠落していた `net/http/otelhttp` / `otel/sdk/log` もそこで補完済み）。

分割の理由: 単一の可変ファイルはその場編集で決定履歴を失い、不変であるべき決定と
`go.mod` 追従の依存表が混在していました。per-file ADR にすれば、モノリスを
触らず 1 ファイル追加で個別の決定を supersede でき、依存目録も不変記録から分離できます。
[ADR-0000](adr/0000-record-architecture-decisions.ja.md) を参照。
