# Inquiry — Domain Spec

> 問い合わせ（利用者と運営のやり取り）のドメイン spec。Realtime Delivery（[`docs/design/realtime-delivery.md`](../../design/realtime-delivery.md)）の最初の consumer となる sample feature であり、
> 撤去可能な範囲（`sample-api`）に属する。機構側（stream / sequence / cursor / ticket）の語彙はここには置かない——連番は機構の
> 状態であり、ドメインのどの集約も field として持たず、どの Repository も採番しない（usecase spec の Dependencies を参照）。
> 問い合わせは **利用者ごとに 1 件の active な問い合わせ** を持つ最小形で、close / reopen・複数問い合わせ・担当割り当て・既読・
> mention・本文編集・撤回・添付・自動応答は範囲外（親 issue の「最初の consumer」節）。

## Overview

**問い合わせ（Inquiry）** は、利用者が運営に対して開始し、運営が応答する一連のやり取りである。利用者 1 人につき active な問い合わせは
1 件で、利用者向けに問い合わせを作る API は設けず、最初の投稿時に取得または作成する。問い合わせが持つ状態は「誰が始めたか」と
「最後にいつ動いたか」だけで、それ以外の状態遷移を持たない。

**問い合わせメッセージ（Message）** は、問い合わせの中で一方（利用者または回答者）が相手に送る 1 通で、append-only —
作成後に編集も取り消しもされない。メッセージは自身の ID で取得されることも問い合わせを経由せず一覧されることもなく、
到達経路は常に問い合わせを解決した後の `ListMessages` である。したがって独立した集約ではなく問い合わせの **sub-entity** で、
親への逆参照を持たない（[`internal/domain/README.md`](../../../internal/domain/README.md) § Cross-aggregate reference
「a type reachable only through its parent is a sub-entity of that aggregate」および「Never give a sub-entity a
back-reference to its parent」）。追加は必ず Root の `AppendMessage` を通る（§ Aggregate consistency）。

集約が肥大しないのは、Root がメッセージの集合を保持しないためである。追加は 1 通を鋳造するだけで既存を読まず、
メッセージ間にまたがる不変条件は存在しない——連番は機構が採番するので、Root が集合に対して守るべきものが無い。

メッセージが持つ `sequence` は「その問い合わせの何通目か」を表す値であり、配送順序の基準として Realtime Delivery が要求する。
値そのものは機構が採番し（usecase spec の `realtime.SequenceAllocator`）、ドメインは「正の整数である」ことだけを検証する。

`id` は UUIDv7（[ADR-0037 (uuidv7-identifiers)]）で、生成は usecase 層が行いドメインへ渡す。時刻も clock 境界から供給され、
ドメインは乱数・時刻に直接依存しない。

## Entity

```yaml
package: internal/domain/inquiry
struct: Inquiry
constructors:
  - name: New            # 最初の投稿時に usecase が作る（active な問い合わせが無いときだけ）
  - name: Reconstruct    # 永続化済みの再構築（Repository の読み出し）
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidID
  - name: userID
    type: uuid.UUID
    required: true          # 問い合わせを開始した利用者。IsNil の場合は ErrInvalidUserID
  - name: createdAt
    type: time.Time         # New ではゼロ値（DB 既定 NOW()）。Reconstruct で設定
  - name: updatedAt
    type: time.Time         # 最後にメッセージが追加された時刻。Reconstruct で設定
```

```yaml
package: internal/domain/inquiry     # Root と同じパッケージの sub-entity（cart / cart_item と同じ形）
struct: Message
constructors:
  - name: Inquiry.AppendMessage   # 投稿 / 回答時に Root が鋳造する（唯一の追加入口）
  - name: ReconstructMessage      # 永続化済みの再構築
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidMessageID
  - name: author
    type: Author            # 値オブジェクト。誰が送ったか（利用者 / 回答者）
  - name: body
    type: string
    required: true          # 空は ErrEmptyBody。文字数上限は maxBodyLength（下記 Notes）。超過は ErrBodyTooLong
    max_length: 4000
  - name: sequence
    type: int64             # 問い合わせ内の何通目か（1 始まり）。0 以下は ErrInvalidSequence。usecase が機構の採番結果を渡す
  - name: createdAt
    type: time.Time         # New ではゼロ値（DB 既定 NOW()）。Reconstruct で設定
```

## Cross-field Invariants

- `Message.sequence >= 1`。問い合わせ内での一意性と単調増加は機構側（sequence allocator の行ロック、
  [ADR-0072 (postgres-state-dynamodb-eventlog)] の不変条件 1）が保証し、Message 単体では「正の整数である」ことだけを
  検証する（一意性は集約 1 件では判定できない）。永続化側は `(inquiryID, sequence)` の UNIQUE で防御する。
- `Message.body` は空でなく、`maxBodyLength` 文字（rune 数）以下。UTF-8 で最大 3 バイト × 4000 = 12 KiB であり、Realtime Delivery の
  payload 上限 64 KiB を event の封筒と author を足しても超えない。
- `Message.author` は利用者と回答者のどちらかであり、利用者のとき `subjectID` は問い合わせの `userID` と一致する
  （この照合は問い合わせを読める usecase の責務。Message 単体では判定できない）。
- `Inquiry.updatedAt >= Inquiry.createdAt`。

## Behavior Methods

```yaml
- name: AppendMessage
  signature: AppendMessage(id uuid.UUID, attrs MessageAttributes, now time.Time) (*Message, error)
  behavior: |
    問い合わせへ 1 通追加し、updatedAt を now へ進める。メッセージの生成入口はここだけで、Root を経由しない生成経路は
    持たない（internal/domain/README.md § Aggregate consistency）。now が既存の updatedAt より前なら ErrInvalidTime を、
    メッセージの検証に失敗すればその検証エラーを返し、いずれの場合も updatedAt は進めない。連番（sequence）は機構が
    採番した値を呼び出し側が渡す。
  invariants:
    - updatedAt は単調に進む
    - メッセージの検証を通らない限り updatedAt は進まない
```

```yaml
- name: IsFrom
  signature: IsFrom(kind AuthorKind, subjectID uuid.UUID) bool
  behavior: |
    メッセージの送り手が指定の種別・主体であるかを返す（Message 側）。History の表示や event payload の author 導出に使う
    純粋な述語。状態を変えない。
```

## Domain Service

「この利用者の投稿はこの問い合わせに属してよいか」は問い合わせ 1 件の `userID` と投稿者の照合であり、集合についての
問いではない。したがって Domain Service は置かず、照合は usecase が `Inquiry` を読んだうえで行う。

## Value Objects

```yaml
- name: Author
  underlying_type: struct { kind AuthorKind; subjectID uuid.UUID }
  validation: |
    kind は AuthorKindUser / AuthorKindOperator のいずれか（それ以外は ErrInvalidAuthorKind）。
    subjectID は IsNil でないこと（ErrInvalidAuthorSubject）。利用者のときは問い合わせの userID、回答者のときは運営側の
    利用者 ID（admin ロールを持つ identity）を指す。Realtime Delivery へは feature 非依存の subject として渡す。
  factory: NewAuthor
  methods:
    - name: Kind
      returns: AuthorKind
    - name: SubjectID
      returns: uuid.UUID
```

```yaml
- name: AuthorKind
  underlying_type: string      # "user" | "operator"
  validation: 上記 2 値以外は ErrInvalidAuthorKind
  factory: NewAuthorKind
  methods:
    - name: String
      returns: string
```

## Repository Methods

```yaml
- name: FindByID
  signature: FindByID(ctx context.Context, id uuid.UUID) (*Inquiry, error)
  behavior: 問い合わせを 1 件読み出す。存在しなければ apperror.ErrNotFound。
- name: FindActiveByUserID
  signature: FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*Inquiry, error)
  behavior: 利用者の active な問い合わせを読み出す（利用者ごとに最大 1 件。UNIQUE 制約で保証）。無ければ apperror.ErrNotFound。
- name: Create
  signature: Create(ctx context.Context, inquiry *Inquiry) error
  behavior: |
    問い合わせを作成する。同じ利用者の active な問い合わせが既にあれば UNIQUE 違反を apperror.ErrConflict に正規化して返す
    （最初の投稿の競合）。UNIQUE 違反は transaction 自体を中断させるため、呼び出し側は同じ transaction の中で読み直せない
    （usecase spec の AppendMessage を参照。`docs/spec/cart/usecase.md` の SetItem と同じ扱い）。
- name: Update
  signature: Update(ctx context.Context, inquiry *Inquiry) error
  behavior: updatedAt を永続化する（AppendMessage が進めた値）。
- name: ListForOperator
  signature: ListForOperator(ctx context.Context, params ListParams) ([]*Inquiry, error)
  behavior: |
    運営側の一覧（更新日時の新しい順、keyset ページネーション: updatedAt desc, id desc）。読み取り専用。
    一覧は問い合わせ集約の行だけで組み立て、メッセージ本文は含めない（最新メッセージの要約は operator feed の event が運ぶ）。

# メッセージは集約の内側にあるため、その読み書きも同じ Repository が担う。
# メッセージが親への逆参照を持たない結果として、inquiryID は引数で受け取る。
- name: CreateMessage
  signature: CreateMessage(ctx context.Context, inquiryID uuid.UUID, message *Message) error
  behavior: |
    メッセージを追加する。(inquiryID, sequence) の UNIQUE 違反は apperror.ErrConflict（採番と同一 tx で呼ぶ限り到達しない
    防御）。業務 tx の中から、機構の採番の後に呼ぶ。
- name: ListMessages
  signature: ListMessages(ctx context.Context, inquiryID uuid.UUID, params HistoryParams) ([]*Message, error)
  behavior: |
    問い合わせのメッセージを sequence 昇順で keyset ページネーションして読み出す（cursor = sequence。afterSequence より大きく
    **upToSequence 以下** の行を limit 件）。History API の本体。upToSequence は usecase が先に読んだ streamCursor で、
    これを上限にすることで「streamCursor と同じ snapshot で読んだ」のと等価な結果になる（usecase spec の GetHistory を参照）。
```

## Notes

- **1 集約と、同一 tx の中身。** Message は問い合わせを経由してしか到達しない（自身の ID で取得する経路も、問い合わせを
  経由しない一覧も持たない）ため、[`internal/domain/README.md`](../../../internal/domain/README.md) の基準どおり
  sub-entity であり、問い合わせと合わせて 1 集約である。したがって投稿 1 回の transaction で書かれるのは
  **問い合わせ集約 1 件**（`inquiry_messages` の 1 行と `inquiries.updated_at`）+ 機構の行（sequence 行、outbox 行）だけで、
  [ADR-0034 (commandservice-atomicity-criterion)] の逸脱には当たらない。sequence 行と outbox 行はどの集約にも属さない機構の
  状態で、同 ADR の worked instance（outbox insert は CommandService を正当化しない）と同型である。

  この形は `cart` / `cart_item` と同じで、Root と同じパッケージに sub-entity を置き、Repository は Root の 1 本だけを持つ。
  Message を独立した集約にすると `inquiries.updated_at` の更新が集約跨ぎの書き込みになり、ADR-0034 が認める 2 つの逸脱の
  どちらにも当たらないまま同一 tx に入ることになる——それが避けられている理由は、境界の引き方が
  「到達経路」という基準に従っているからである。

- **連番はドメインが持たない。** 以前の案では Inquiry に `lastSequence` を持たせ Repository で採番していたが、それは機構語彙を
  ドメインへ持ち込み、Repository に不変条件の保証を負わせ、Root ではない行（feed）を Inquiry の Repository が操作する形に
  なっていた。採番は Realtime Delivery の `SequenceAllocator`（usecase boundary。`boundary/outbox` と同型、table は `system_cqrs`
  区分）が stream ごとの行で行い、会話 stream（`streamId` = 問い合わせ ID）も運営 feed（`streamId` = feed ID）も同じ table の
  行である。History の `streamCursor` もこの行から読む。
- **placeholder 定数。** `maxBodyLength = 4000`（rune 数）。上限の根拠は「Realtime Delivery の payload 上限 64 KiB を十分下回る」
  こと（UTF-8 最大 12 KiB）。文字数の業務要件が立てばこの spec で改める。
- **エラー写像。** `ErrEmptyBody` / `ErrBodyTooLong` / `ErrInvalidAuthorKind` → 422、`apperror.ErrNotFound` → 404、
  `apperror.ErrConflict`（active な問い合わせの二重作成）→ 409、他者の問い合わせへのアクセス → 403（usecase 側の認可）。
- **撤去範囲。** `docs/spec/inquiry/**`、`internal/domain/inquiry`、対応する migration / DML /
  usecase / handler は `sample-api` の撤去対象。`scripts/setup/remove-sample-api/sample-manifest.ts` への登録は Phase 9（feature adapter と最小 API）で
  実施済み。Realtime Delivery 本体は
  残る（親 issue「sample 削除後の残存」）。

[ADR-0034 (commandservice-atomicity-criterion)]: ../../adr/0034-commandservice-atomicity-criterion.md
[ADR-0037 (uuidv7-identifiers)]: ../../adr/0037-uuidv7-identifiers.md
[ADR-0072 (postgres-state-dynamodb-eventlog)]: ../../adr/0072-postgres-state-dynamodb-eventlog.md
