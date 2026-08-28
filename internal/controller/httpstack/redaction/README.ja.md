# redaction

リクエスト URI と query パラメータから、ログへ書き出す前に資格情報の値を取り除きます。

## 役割

stream ticket は query パラメータとして運ばれる（ADR-0074 (query-ticket-stream-authentication)）ため、
リクエスト URI そのものが資格情報を含みます。リクエストをログに出す箇所 — アクセスログ・エラーハンドラ・
panic 復帰 — はすべて、URI と query の map を先に `Redactor` へ通します。規則は HTTP スタックで強制し、
ハンドラには委ねません。

## 名前の出所

`FromSpec(spec)` は OpenAPI の `securitySchemes` を読み、`in: query` で受け取る `apiKey` scheme のパラメータ名を
集めます。`ticket` という名前が書かれているのは spec のこの 1 箇所だけです。query の資格情報 scheme を増やせば
秘匿対象も自動で広がり、Go 側に同期すべき第 2 の一覧はありません（ADR-0016 (spec-driven-request-validation)）。

## API

- `New(names []string) Redactor` — 渡した名前を秘匿する。ゼロ値は何も秘匿しない。
- `FromSpec(spec *openapi3.T) Redactor` — spec から名前を導出する。
- `SecretQueryParamNames(spec) []string` — 導出した名前を名前順で返す（テスト・診断用）。
- `Redactor.URI(raw string) string` — 生のリクエスト URI の秘匿対象の値を `[REDACTED]` に置き換える。
  組の並びと符号化は保つ。標準の構文解析が受け付けない query（`;` 区切り・壊れた符号化）は組ごとに判定できないため
  全体を置き換える（fail-closed）。
- `Redactor.QueryParams(map[string][]string) map[string][]string` — 秘匿対象の値を置き換えた複製を返す。
  入力の map は変更しない。

## 配線

`internal/di/module/core.RedactionModule()` が validator と同じ `*openapi3.T` から `Redactor` を組み立て、
3 つのログ経路が DI で受け取ります。

| 経路 | 適用箇所 |
| --- | --- |
| アクセスログ（`httpstack/logging`） | request / response の両フィールド |
| エラーハンドラ（`httpstack/errorhandler`） | `Policies.Redact` → `server.BuildHTTPRequestLogInput` |
| panic 復帰（`httpstack/recovery`） | `server.BuildHTTPRequestLogInput` |

3 経路が分かれているのは意図的です。アクセスログは OpenAPI validator の内側で動くため、認証で拒否された
リクエストはそこへ届かず、代わりにエラーハンドラがログに出します。

## テスト戦略

- ユニットテストで URI の書き換え（並び・符号化・同名の繰り返し・復号不能な名前）と query map の複製の意味論を固定する。
- `internal/integration` で本物のミドルウェア連鎖に `?ticket=` 付きリクエストを通し、どのログにも生値が無く
  `[REDACTED]` が現れることを表明する（ログを抑止したのではなく値を秘匿したことの証明）。
