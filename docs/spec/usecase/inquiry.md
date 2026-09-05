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
2 つの destination には 2 つの event を出す。連番は Realtime Delivery の `SequenceAllocator` が stream ごとの行（会話 stream は
問い合わせ ID、feed は feed ID）から採番し、行ロックを commit まで保持する。連番はどの集約の field でもない。

投稿と回答はクライアントの timeout 後再送でメッセージを二重に生まないよう idempotency middleware に opt-in する
（[ADR-0067 (idempotency-orthogonal-concerns)] のとおり handler ごとに独立）。

## Interface

```yaml
package: internal/usecase/inquiry
interface: Usecase
methods:
  - name: AppendMessage      # 利用者の投稿。active な問い合わせを取得または作成して append
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
      - name: StreamCursor        # 問い合わせ stream の現在位置。Messages はこれ以下の sequence だけを含む
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
        type: "[]InquirySummaryView"   # { ID, UserID, UpdatedAt, CreatedAt }
      - name: NextCursor
        type: "*InquiryCursor"
```

## Dependencies

```yaml
- name: tx.Manager                        # 投稿 / 回答の業務 tx（採番 + 追加 + emit）。履歴も同じ tx 境界で読む
- name: clock                             # boundary.Clock（updatedAt / createdAt）
- name: inquiry.Repository                # FindByID / FindActiveByUserID / Create / Update / CreateMessage / ListMessages / ListForOperator
- name: realtime.SequenceAllocator        # boundary/realtime。Allocate(streamID)（行ロックを commit まで保持）/ Current(streamID)（現在位置の読み出し）
- name: outbox.EmitUsecase                # 2 行 emit（realtime channel、ordering_key / ordering_sequence 付き）
- name: realtime.TicketIssuer             # internal/usecase/realtime（機構の usecase）。subject × destination × scope × expiry に bind した ticket を発行し boundary/realtime.StreamTicketStore へ保存する
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
    - "txm.Do 内で: inquiryRepo.FindActiveByUserID(UserID)。無ければ inquiry.New で候補を鋳造し inquiryRepo.CreateIfAbsent"
    - "CreateIfAbsent は一意インデックスが単一文の中で裁定するため、作成が並行して競合しても一意制約違反を上げず、勝ったほうの問い合わせを返す（docs/spec/usecase/cart.md の MergeOnLogin と同じ扱い）"
    - "取得または作成した問い合わせに対して、同じ txm.Do 内で:"
    - "  ① seq = seqAlloc.Allocate(inquiry.ID)（会話 stream の連番。行ロックは commit まで保持。同一問い合わせへの並行投稿はここで直列化）"
    - "  ② inquiry.AppendMessage(id, {NewAuthor(user, UserID), Body, seq}, now) で鋳造し repo.CreateMessage / repo.Update"
    - "  ③ AppendMessage が進めた updatedAt を repo.Update で永続化"
    - "  ④ feedSeq = seqAlloc.Allocate(feedID)（feed stream の連番。組織全体の投稿・回答はここで直列化する）"
    - "  ⑤ emit.Emit(inquiry.message.created.v1; channel=realtime, ordering_key=inquiry.ID, ordering_sequence=seq, payload={messageId, inquiryId, author{kind}, body, sequence, createdAt})"
    - "  ⑥ emit.Emit(inquiry.thread.updated.v1; channel=realtime, ordering_key=feedID, ordering_sequence=feedSeq, payload={inquiryId, userId, sequence=seq, updatedAt})"
    - MessageView へ写像して返す
  calls:
    - tx.Manager.Do
    - inquiry.Repository.FindActiveByUserID
    - inquiry.New
    - inquiry.Repository.CreateIfAbsent
    - realtime.SequenceAllocator.Allocate
    - inquiry.Inquiry.AppendMessage
    - inquiry.Repository.CreateMessage
    - inquiry.Repository.Update
    - outbox.EmitUsecase.Emit
    - clock.Now
  errors:
    - ErrEmptyBody / ErrBodyTooLong → 422
    - ErrConflict（active な問い合わせの二重作成が再試行でも解けない）→ 409

- method: GetHistory
  tx_required: true          # 読み取りのみの tx。cursor を先に読み、その位置までの messages を読む
  steps:
    - "txm.Do 内で（読み取りのみ）:"
    - "  ① inquiryRepo.FindActiveByUserID(UserID)。無ければ空の HistoryView（InquiryID ゼロ値、StreamCursor 0）を返す"
    - "  ② cursor = seqAlloc.Current(inquiry.ID)（会話 stream の現在位置）"
    - "  ③ repo.ListMessages(inquiry.ID, AfterSequence, upTo=cursor, Limit+1)"
    - "  ④ StreamCursor = cursor"
    - Limit を超えた分で NextAfterSequence を決め、HistoryView へ写像して返す
  calls:
    - tx.Manager.Do
    - inquiry.Repository.FindActiveByUserID
    - realtime.SequenceAllocator.Current
    - inquiry.Repository.ListMessages
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
  tx_required: true          # 読み取りのみの tx（GetHistory と同じ手順）
  steps:
    - "authz.Authorize(authn, admin)"
    - "txm.Do 内で repo.FindByID(InquiryID) → cursor = seqAlloc.Current(InquiryID) → repo.ListMessages(InquiryID, AfterSequence, upTo=cursor, Limit+1) → StreamCursor = cursor"
    - HistoryView へ写像して返す
  calls:
    - authz.Authorizer.Authorize
    - tx.Manager.Do
    - inquiry.Repository.FindByID
    - realtime.SequenceAllocator.Current
    - inquiry.Repository.ListMessages
  errors:
    - ErrPermissionDenied → 403
    - ErrNotFound → 404

- method: Reply
  tx_required: true
  steps:
    - "authz.Authorize(authn, admin)"
    - "message id を UUIDv7 で採番する"
    - "txm.Do 内で: repo.FindByID(InquiryID) → seq = seqAlloc.Allocate(InquiryID) → inquiry.AppendMessage(author=NewAuthor(operator, OperatorID)) → repo.CreateMessage → repo.Update → feedSeq = seqAlloc.Allocate(feedID) → 2 行 emit（AppendMessage の ⑤⑥ と同じ）"
    - MessageView へ写像して返す
  calls:
    - authz.Authorizer.Authorize
    - tx.Manager.Do
    - inquiry.Repository.FindByID
    - realtime.SequenceAllocator.Allocate
    - inquiry.Inquiry.AppendMessage
    - inquiry.Repository.CreateMessage
    - inquiry.Repository.Update
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
    - "ticket = ticketIssuer.Issue(subject=authn.Subject, destination=feedID, scope=inquiry-feed:read)"
    - TicketView へ写像して返す
  calls:
    - authz.Authorizer.Authorize
    - realtime.TicketIssuer.Issue
  errors:
    - ErrPermissionDenied → 403
```

## Notes

- **realtime adapter の置き場所。** feature event → `DeliveryEvent` の変換（destination の決定、event type、payload の組み立て、
  `SequenceAllocator` からの採番、ordering_key / ordering_sequence の付与）は `internal/usecase/inquiry/` 内の adapter が担う。
  Realtime Delivery 側の package は `inquiry` を import しない（architecture test で強制。
  [ADR-0071 (realtime-delivery-driving-mechanism)]）。
- **author に主体 ID を載せない。** `inquiry.message.created.v1` の `author` は `kind` だけを持つ。会話画面は
  「利用者か回答者か」しか必要とせず、主体 ID を配ると宛先の識別子が stream の外へ広がる。
- **2 つの destination。** 会話画面は問い合わせ stream（`streamId` = 問い合わせ ID）、一覧画面は組織 feed stream（`streamId` =
  feed ID、単一組織のため固定値 1 つ）を購読する。1 event = 1 stream を守るため 2 行 emit し、連番は `SequenceAllocator` の
  stream ごとの行から採番する。feed の行は組織で 1 つなので、feed の採番は組織内のあらゆる投稿・回答を直列化する
  （[ADR-0072 (postgres-state-dynamodb-eventlog)] の「1 stream の書き込みは 1 行で直列化する」がそのまま feed に当たる。
  sample の規模では見えない制約で、feed をシャーディングする設計はこの spec の範囲外）。feed の event は本文を持たない軽量なもの
  （`inquiryId / userId / sequence / updatedAt`）。
- **最初の投稿の競合。** 同じ利用者の初回投稿が並行しても、`CreateIfAbsent` は一意インデックスが単一文の中で裁定する
  形なので一意制約違反を上げない。負けた側にはその文が勝ったほうの行を返すため、トランザクションは中断せず、
  どちらの投稿も成功する（`docs/spec/usecase/cart.md` の「引き継ぎ先を『無ければ作る』で確保する」と同じ。存在確認と
  作成を分けると、その間に他の要求が作った場合に 23505 でトランザクションごと中断してしまう）。
  `tx.Manager.Do` の自動 retry は serialization failure / deadlock だけが対象で UNIQUE 違反は対象外であり、
  さらに `idempotency.Run` の内側では `Manager.Do` が外側の tx を再利用するため、usecase が自前で tx をやり直しても
  新しい tx は取れない。この経路で UNIQUE 違反に頼る設計を採ってはならない。
- **streamCursor と snapshot。** History は先に `SequenceAllocator.Current` で stream の現在位置（cursor）を読み、次に
  `sequence <= cursor` で messages を読む。採番の行ロックが commit まで保持されるため、cursor = c を読めた時点で sequence ≤ c の
  message はすべて commit 済みであり、c より大きいものは除外する——既定の READ COMMITTED（文ごとに snapshot）のままで
  「cursor と messages を同一 snapshot で読んだ」のと等価になる。分離レベルの変更も単一 SQL 化も要らない。クライアントは
  この値を `after` にして接続すれば、History 取得と SSE 接続の間の event を取りこぼさない（親 issue 受入基準）。
- **認可の場所。** 利用者は `UserID` で自分の問い合わせに閉じ（構造的に他者の問い合わせへ到達しない）、運営は `authz` で
  admin ロールを判定する。ticket 発行時に認可した subject × destination を Realtime Delivery が保持し、Streamer は認可を
  行わない（[ADR-0074 (query-ticket-stream-authentication)]）。
- **失効の呼び出し。** 利用者の退会（`docs/spec/usecase/user.md` の `DeleteUser`）は、soft-delete と同じ tx の後に
  `realtime` の失効 seam を subject × 問い合わせ stream で呼ぶ義務を負う。これは当該 feature の spec 側に書く義務であり、
  本 spec は責務の所在だけを記す。`user/usecase.md` への追記と実装は Phase 9（inquiry feature）で行う。運営の admin ロールを
  剥奪する書き込み経路は現時点で存在しない（`internal/usecase/user/role` は読み取りのみ）ため呼び出し元も存在しない。
  剥奪経路を追加する変更は、同時に feed stream について失効 seam を呼ぶ義務を負う。
- **idempotency。** `AppendMessage` と `Reply` の handler は idempotency middleware に opt-in する。Idempotency-Key の scope は
  認証済み主体。
- **placeholder。** feed ID は単一組織の固定値。組織が複数になれば `ListInquiries` / `IssueFeedTicket` に組織の軸が入る。

[ADR-0067 (idempotency-orthogonal-concerns)]: ../../adr/0067-idempotency-orthogonal-concerns.md
[ADR-0071 (realtime-delivery-driving-mechanism)]: ../../adr/0071-realtime-delivery-driving-mechanism.md
[ADR-0072 (postgres-state-dynamodb-eventlog)]: ../../adr/0072-postgres-state-dynamodb-eventlog.md
[ADR-0074 (query-ticket-stream-authentication)]: ../../adr/0074-query-ticket-stream-authentication.md
