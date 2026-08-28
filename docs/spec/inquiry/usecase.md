# Inquiry — Usecase Spec

> 問い合わせの usecase spec。利用者の投稿 / 履歴 / stream ticket 発行、運営側の一覧 / 履歴 / 回答 / feed ticket 発行を扱う。
> Realtime Delivery（[`docs/design/realtime-delivery.md`](../../design/realtime-delivery.md)）への変換（realtime adapter）は
> この usecase package の内側に置き、機構側は問い合わせの語彙を知らない（[ADR-0071 (realtime-delivery-driving-mechanism)]）。
> HTTP のワイヤ契約（path / status / query の形）は `openapi/openapi.yaml` が正本であり、ここには置かない。

## Overview

利用者は自分の問い合わせだけを投稿・閲覧・購読できる。最初の投稿で active な問い合わせを取得または作成し、以降の投稿は同じ
問い合わせに append する。運営（admin ロール）は問い合わせの一覧・履歴・回答を扱い、一覧画面の更新は組織単位の feed stream を
購読して受け取る。

投稿と回答は同じ手順で 2 つの outbox 行を同一 tx に emit する: 問い合わせ stream 宛ての `inquiry.message.created.v1`
（会話画面へ）と、feed stream 宛ての軽量な `inquiry.thread.updated.v1`（一覧画面へ）。1 event = 1 stream を守るため、
2 つの destination には 2 つの event を出す。連番はそれぞれの stream の所有行（問い合わせ行 / feed 行）から採番する。

投稿と回答はクライアントの timeout 後再送でメッセージを二重に生まないよう idempotency middleware に opt-in する
（[ADR-0067 (idempotency-orthogonal-concerns)] のとおり handler ごとに独立）。

## Interface

```yaml
package: internal/usecase/inquiry
interface: Usecase
methods:
  - name: AppendMessage        # 利用者の投稿。active な問い合わせを取得または作成して append
    signature: AppendMessage(ctx context.Context, params AppendMessageParams) (MessageView, error)
  - name: GetHistory         # 利用者の履歴（自分の問い合わせ、sequence 昇順 keyset）。streamCursor 付き
    signature: GetHistory(ctx context.Context, params HistoryParams) (*HistoryView, error)
  - name: IssueStreamTicket  # 利用者の会話 stream 用 ticket
    signature: IssueStreamTicket(ctx context.Context, params IssueStreamTicketParams) (TicketView, error)
  - name: ListInquiries      # 運営の一覧（updatedAt desc keyset）
    signature: ListInquiries(ctx context.Context, authn *auth.Authn, params ListInquiriesParams) (*InquiryListView, error)
  - name: GetInquiryHistory  # 運営の履歴（任意の問い合わせ）。streamCursor 付き
    signature: GetInquiryHistory(ctx context.Context, authn *auth.Authn, params OperatorHistoryParams) (*HistoryView, error)
  - name: Reply              # 運営の回答（append）
    signature: Reply(ctx context.Context, authn *auth.Authn, params ReplyParams) (MessageView, error)
  - name: IssueFeedTicket    # 運営の feed stream 用 ticket
    signature: IssueFeedTicket(ctx context.Context, authn *auth.Authn) (TicketView, error)
```

## DTOs

```yaml
input:
  - struct: AppendMessageParams
    fields:
      - name: UserID          # 認証済みの内部ユーザー ID（IdentityResolver 解決済み）
        type: uuid.UUID
      - name: Subject         # Realtime Delivery へ渡す feature 非依存の principal 識別子（identity の subject）
        type: string
      - name: Body
        type: string
  - struct: HistoryParams
    fields:
      - name: UserID
        type: uuid.UUID
      - name: AfterSequence   # keyset cursor。nil は先頭から
        type: "*int64"
      - name: Limit
        type: int
  - struct: IssueStreamTicketParams
    fields:
      - name: UserID
        type: uuid.UUID
      - name: Subject
        type: string
  - struct: ListInquiriesParams
    fields:
      - name: Cursor          # keyset cursor（updatedAt, id）。nil は先頭から
        type: "*InquiryCursor"
      - name: Limit
        type: int
  - struct: OperatorHistoryParams
    fields:
      - name: InquiryID
        type: uuid.UUID
      - name: AfterSequence
        type: "*int64"
      - name: Limit
        type: int
  - struct: ReplyParams
    fields:
      - name: InquiryID
        type: uuid.UUID
      - name: OperatorID      # 回答者（admin ロールの内部ユーザー ID）
        type: uuid.UUID
      - name: Body
        type: string

output:
  - struct: MessageView
    fields:
      - name: ID
        type: uuid.UUID
      - name: InquiryID
        type: uuid.UUID
      - name: AuthorKind
        type: string          # "user" | "operator"
      - name: Body
        type: string
      - name: Sequence
        type: int64
      - name: CreatedAt
        type: time.Time
  - struct: HistoryView
    fields:
      - name: InquiryID
        type: uuid.UUID
      - name: Messages
        type: "[]MessageView"
      - name: NextAfterSequence   # 次ページの cursor。無ければ nil
        type: "*int64"
      - name: StreamCursor        # 問い合わせの lastSequence。Messages と同じ snapshot から確定
        type: int64
  - struct: TicketView
    fields:
      - name: Ticket              # 生値。応答本文にのみ載せ、log / trace へは出さない
        type: string
      - name: StreamID            # 接続先 stream（問い合わせ ID または feed ID）
        type: string
      - name: ExpiresAt
        type: time.Time
  - struct: InquiryListView
    fields:
      - name: Items
        type: "[]InquirySummaryView"   # { ID, UserID, LastSequence, UpdatedAt, CreatedAt }
      - name: NextCursor
        type: "*InquiryCursor"
```

## Dependencies

```yaml
- name: tx.Manager                        # 投稿 / 回答の業務 tx（採番 + 追加 + emit）。履歴も同じ tx 境界で読み、streamCursor と messages を同一 snapshot にする
- name: clock                             # boundary.Clock（updatedAt / createdAt）
- name: inquiry.Repository                # FindByID / FindActiveByUserID / Create / AllocateSequence / AllocateFeedSequence / Touch / ListForOperator
- name: inquirymessage.Repository         # Create / ListByInquiry
- name: outbox.EmitUsecase                # 2 行 emit（realtime channel、ordering_key / ordering_sequence 付き）
- name: realtime.TicketIssuer             # boundary/realtime。subject × destination × scope × expiry に bind した ticket の発行
- name: authz.Authorizer                  # 運営操作の admin ロール判定
- name: observability.TracerFactory
- name: pkg/uuid                          # id の UUIDv7 採番
```

## Workflow

```yaml
- method: AppendMessage
  tx_required: true
  steps:
    - "message id を UUIDv7 で採番する"
    - "txm.Do 内で:"
    - "  ① inquiryRepo.FindActiveByUserID(UserID)。ErrNotFound なら inquiry.New で作成し inquiryRepo.Create（UNIQUE 違反の ErrConflict は最初の投稿の競合。1 度だけ FindActiveByUserID を読み直す）"
    - "  ② seq = inquiryRepo.AllocateSequence(inquiry.ID)（行ロックは commit まで保持。同一問い合わせへの並行投稿はここで直列化）"
    - "  ③ inquirymessage.New(id, inquiry.ID, NewAuthor(user, UserID), Body, seq) で検証し msgRepo.Create"
    - "  ④ inquiry.Touch(now) → inquiryRepo.Touch"
    - "  ⑤ feedSeq = inquiryRepo.AllocateFeedSequence()"
    - "  ⑥ emit.Emit(inquiry.message.created.v1; channel=realtime, ordering_key=inquiry.ID, ordering_sequence=seq, payload={messageId, inquiryId, author{kind}, body, sequence, createdAt})"
    - "  ⑦ emit.Emit(inquiry.thread.updated.v1; channel=realtime, ordering_key=feed ID, ordering_sequence=feedSeq, payload={inquiryId, userId, lastSequence=seq, updatedAt})"
    - MessageView へ写像して返す
  calls:
    - tx.Manager.Do
    - inquiry.Repository.FindActiveByUserID
    - inquiry.New
    - inquiry.Repository.Create
    - inquiry.Repository.AllocateSequence
    - inquirymessage.New
    - inquirymessage.Repository.Create
    - inquiry.Touch
    - inquiry.Repository.Touch
    - inquiry.Repository.AllocateFeedSequence
    - outbox.EmitUsecase.Emit
    - clock.Now
  errors:
    - ErrEmptyBody / ErrBodyTooLong → 422
    - ErrConflict（active な問い合わせの二重作成が再読でも解けない）→ 409

- method: GetHistory
  tx_required: true          # 読み取りのみの tx。streamCursor と messages を同じ snapshot から取るため
  steps:
    - "txm.Do 内で（読み取りのみ）:"
    - "  ① inquiryRepo.FindActiveByUserID(UserID)。無ければ空の HistoryView（InquiryID ゼロ値、StreamCursor 0）を返す"
    - "  ② msgRepo.ListByInquiry(inquiry.ID, AfterSequence, Limit+1)"
    - "  ③ StreamCursor = inquiry.LastSequence（同じ snapshot）"
    - Limit を超えた分で NextAfterSequence を決め、HistoryView へ写像して返す
  calls:
    - tx.Manager.Do
    - inquiry.Repository.FindActiveByUserID
    - inquirymessage.Repository.ListByInquiry
  errors:
    - なし（自分の問い合わせしか読めない構造のため 403 は生じない）

- method: IssueStreamTicket
  tx_required: false
  steps:
    - "inquiryRepo.FindActiveByUserID(UserID)。無ければ ErrNotFound（購読する問い合わせが無い）"
    - "ticket = ticketIssuer.Issue(subject=Subject, destination=inquiry.ID, scope=inquiry:read)"
    - TicketView（生値・StreamID・ExpiresAt）へ写像して返す。生値は log / trace に出さない
  calls:
    - inquiry.Repository.FindActiveByUserID
    - realtime.TicketIssuer.Issue
  errors:
    - ErrNotFound → 404

- method: ListInquiries
  tx_required: false
  steps:
    - "authz.Authorize(authn, admin)。拒否は 403"
    - "inquiryRepo.ListForOperator(cursor, Limit+1)"
    - InquiryListView へ写像して返す
  calls:
    - authz.Authorizer.Authorize
    - inquiry.Repository.ListForOperator
  errors:
    - ErrPermissionDenied → 403

- method: GetInquiryHistory
  tx_required: true          # 読み取りのみの tx（GetHistory と同じ理由）
  steps:
    - "authz.Authorize(authn, admin)"
    - "txm.Do 内で（読み取りのみ） inquiryRepo.FindByID(InquiryID) と msgRepo.ListByInquiry(...)、StreamCursor = LastSequence"
    - HistoryView へ写像して返す
  calls:
    - tx.Manager.Do
    - authz.Authorizer.Authorize
    - inquiry.Repository.FindByID
    - inquirymessage.Repository.ListByInquiry
  errors:
    - ErrPermissionDenied → 403
    - ErrNotFound → 404

- method: Reply
  tx_required: true
  steps:
    - "authz.Authorize(authn, admin)"
    - "message id を UUIDv7 で採番する"
    - "txm.Do 内で: inquiryRepo.FindByID(InquiryID) → AllocateSequence → inquirymessage.New(author=NewAuthor(operator, OperatorID)) → msgRepo.Create → Touch → AllocateFeedSequence → 2 行 emit（AppendMessage の ②〜⑦ と同じ）"
    - MessageView へ写像して返す
  calls:
    - tx.Manager.Do
    - authz.Authorizer.Authorize
    - inquiry.Repository.FindByID
    - inquiry.Repository.AllocateSequence
    - inquirymessage.New
    - inquirymessage.Repository.Create
    - inquiry.Touch
    - inquiry.Repository.Touch
    - inquiry.Repository.AllocateFeedSequence
    - outbox.EmitUsecase.Emit
    - clock.Now
  errors:
    - ErrPermissionDenied → 403
    - ErrNotFound → 404
    - ErrEmptyBody / ErrBodyTooLong → 422

- method: IssueFeedTicket
  tx_required: false
  steps:
    - "authz.Authorize(authn, admin)"
    - "ticket = ticketIssuer.Issue(subject=authn.Subject, destination=feed ID, scope=inquiry-feed:read)"
    - TicketView へ写像して返す
  calls:
    - authz.Authorizer.Authorize
    - realtime.TicketIssuer.Issue
  errors:
    - ErrPermissionDenied → 403
```

## Notes

- **realtime adapter の置き場所。** feature event → `DeliveryEvent` の変換（destination の決定、event type、payload の組み立て、
  ordering_key / ordering_sequence の付与）は `internal/usecase/inquiry/` 内の adapter が担う。Realtime Delivery 側の package は
  `inquiry` を import しない（architecture test で強制。[ADR-0071 (realtime-delivery-driving-mechanism)]）。
- **2 つの destination。** 会話画面は問い合わせ stream（`streamId` = 問い合わせ ID）、一覧画面は組織 feed stream（`streamId` =
  feed ID、単一組織のため固定値 1 つ）を購読する。1 event = 1 stream を守るため 2 行 emit し、連番は各 stream の所有行から
  採番する。feed の event は本文を持たない軽量なもの（`inquiryId / userId / lastSequence / updatedAt`）。
- **streamCursor と snapshot。** History の `StreamCursor` は `inquiry.lastSequence` を messages と同じ読み取り専用 tx で読んだ値。
  クライアントはこの値を `after` にして接続すれば、History 取得と SSE 接続の間の event を取りこぼさない
  （親 issue 受入基準「History 取得と SSE 接続の間の event を取りこぼさない」）。
- **認可の場所。** 利用者は `UserID` で自分の問い合わせに閉じ（構造的に他者の問い合わせへ到達しない）、運営は `authz` で
  admin ロールを判定する。ticket 発行時に認可した subject × destination を Realtime Delivery が保持し、Streamer は認可を
  行わない（[ADR-0074 (query-ticket-stream-authentication)]）。
- **失効の呼び出し。** 利用者の退会（`docs/spec/user/usecase.md` の `DeleteUser`）は、soft-delete と同じ tx の後に
  `realtime` の失効 seam を subject × 問い合わせ stream で呼ぶ。運営の admin ロール剥奪も同様に feed stream について呼ぶ。
  これらは当該 feature の spec 側に書く義務であり、本 spec は呼び出す責務の所在だけを記す。
- **idempotency。** `AppendMessage` と `Reply` の handler は idempotency middleware に opt-in する。Idempotency-Key の scope は
  認証済み主体。
- **placeholder。** feed ID は単一組織の固定値（migration で seed した `inquiry_feed` 行の ID）。組織が複数になれば
  `ListInquiries` / `IssueFeedTicket` に組織の軸が入る。

[ADR-0067 (idempotency-orthogonal-concerns)]: ../../adr/0067-idempotency-orthogonal-concerns.md
[ADR-0071 (realtime-delivery-driving-mechanism)]: ../../adr/0071-realtime-delivery-driving-mechanism.md
[ADR-0074 (query-ticket-stream-authentication)]: ../../adr/0074-query-ticket-stream-authentication.md
