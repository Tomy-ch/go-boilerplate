---
status: accepted
date: 2026-07-16
deciders: [maintainers]
tags: [errors, architecture, api]
---

# ADR-0043: apperror の上に載せるプロトコル中立なエラーメタ情報（code / message / details）

English canonical: [0043-error-metadata-code-message-details.md](../../adr/0043-error-metadata-code-message-details.md)

## ステータス

accepted

## コンテキスト

[ADR-0042](0042-apperror-protocol-agnostic-errors.ja.md) はプロトコル非依存のエラー taxonomy を
確立した: `internal/apperror` のセンチネルが失敗を分類し、controller のエラーハンドラ
middleware が各センチネルを固定の HTTP ステータス・固定のエラー `code` / `message` へ
マッピングする。このマッピングはステータスごとに厳密な 1:1 であり、エラー発生箇所から
API クライアントへ動的な情報を伝えられない — 例えば、ユーザー更新リクエストの*どの*
フィールドが検証に失敗したか、を伝えられない。

具体的なニーズ: `PUT/PATCH /v1/users/{id}` は不正なフィールドをレスポンスで指摘すべき
（`details: ["firstName", "email"]`）だが、各フィールドが**なぜ**失敗したかは公開しない
（理由はログ専用）。加えて、ドメインの検証は最初に失敗したフィールドしか収集していなかった
（`validateProfileFields` は first-error return）ため、複数同時違反をまとめて報告できなかった。

## 決定

`internal/apperror` に、分類センチネルと直交する**装飾**型を追加する:

- `Meta{Code, Message, Details}` — 全フィールド任意・プロトコル中立。
- `WithMeta(err, meta)` / `WithDetails(err, details...)` がエラーチェーンへ `Meta` を付与し、
  `MetaFrom(err)` が `xerrors.As` で**最も外側**の 1 個を抽出する。
- ラッパー（`MetaError`）は `Unwrap` を実装するため、`xerrors.Is` / `IsAppError` は
  ラップされたセンチネル（`xerrors.Join` の全枝を含む）をそのまま検知する。

edge では `NewHTTPErrorFromAppError` が、センチネル分類から解決した既定値の上に `Meta` を
重ねる: 非空の `Code` / `Message` は既定値を上書きし、`Details` は明示引数が無い場合に採用する。

ADR-0042 を不変に保つための3つの制約:

1. **`Meta` は HTTP ステータスを運ばない。** ステータスはセンチネル分類のみで解決する。
   ステータスを変えたければセンチネルを変える。`Meta` にステータスを持たせることは、
   ADR-0042 が明示的に棄却した HTTP-status-carrying error 型の再導入になる。
2. **`Message` の所有権は controller に残す。** フィールドは汎用性のために存在するが、
   domain / usecase は空のままにする。利用者向け文言は controller のカタログで集中管理され、
   API 文言の変更が内層に触れることはない。
3. **`Details` は公開して安全な識別子のみ**（例: API プロパティ名と一致するドメイン定数
   `user.FieldFirstName` などの不正フィールド名）を持ち、理由文や入力値そのものは決して
   入れない。理由はラップしたエラーメッセージ（`xerrors.Wrap(ErrInvalidXxx, msg)`）に残り、
   ログにのみ現れる。

支持する変更: `user.validateProfileFields` は**全**プロフィールフィールドを検証し、
フィールドごとのセンチネルエラーを結合（`xerrors.Join`）した上で、収集したフィールド識別子を
`WithDetails` で付与する。サーバ内部の不変条件（id / updatedAt / deletedAt）は
first-error return のまま — ユーザーが修正できる入力ではないため。副次効果として
`POST /v1/users`（作成）でも不正フィールドが `details` に載るようになる。これは意図された改善。

## 帰結

### ポジティブな帰結

- エラー発生箇所が、センチネル追加や edge マッピング変更なしに、動的で機械可読な詳細
  （不正フィールドのリスト、機能固有コード）を返せる。
- 複数同時の検証失敗が 1 レスポンスで報告され、1 件ずつのラウンドトリップが不要になる。
- レスポンス契約は後方互換: `details` は既存の任意フィールドで、ステータス / 既定 code /
  既定 message は不変。
- 理由や値は漏れない: 公開面は識別子のみ。

### ネガティブな帰結

- Join された検証エラーは `err.Error()` が複数行になる。エラー文字列全体を比較するテスト
  （`xerrors.Is` ではなく）は修正が必要になる。
- 最外優先の抽出規則により、上位層が `Meta` を再付与すると内側は隠れる — 意図的な仕様だが、
  ラップ時に意識が必要。
- `Details` の「識別子のみ」ルールは型システムではなく規約とレビューで担保される。

## 検討した代替案

### HTTP ステータスを運ぶメタ情報

`Meta` に HTTP ステータスの上書きを許す。棄却: ADR-0042 が取り除いた結合を再導入し、
センチネルのみの解決が構造的に防いでいるステータス/コード不整合（例: 422 なのに
`NOT_FOUND`）を許してしまう。

### controller 側のセンチネル→フィールド名変換テーブル

domain にフィールド識別子を持たせず、edge で `ErrInvalidXxx` センチネルをフィールド名へ
変換する。棄却: 汎用エラーハンドラが各ドメインのセンチネル一覧を総当たりで探索
（Join されたエラーへの `xerrors.Is` fan-out）する必要があり、集約が増えるたびにドメイン
知識が蓄積して肥大化する。フィールド識別子はドメイン語彙（文言でもプロトコルでもない）
であり、domain が定数として公開するのは許容できる。API 名が乖離したらその時点で
controller で再マップすればよい。

### ケースごとのセンチネル + 固定コード追加

動的ケースごとに apperror センチネルと固定 code/message を追加する。棄却: フィールド
単位の報告では定数が爆発し、リクエスト固有のデータも表現できない。

## 備考

- 出典: `internal/apperror/README.ja.md` — エラーメタ情報（`Meta`）節;
  `internal/controller/error/response/README.ja.md` — `apperror.Meta` による上書き節。
- ユーザー更新時の都道府県名の解決失敗は repository 経路の `ErrNotFound`（404）のままで、
  フィールド `details` の対象外。422 へ寄せるかは必要になった時点で別途決定する。
- PATCH はマージ後の全量を検証するため、保存済みデータが不変条件からズレた場合に限り、
  クライアントが送っていないフィールドが理論上 `details` に載り得る。実運用では想定しない。
