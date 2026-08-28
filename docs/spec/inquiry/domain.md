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
作成後に編集も取り消しもされない。メッセージは問い合わせを経由せず一覧・ページングされる（History API）ため、問い合わせの
sub-entity ではなく独立した集約であり、問い合わせへは識別子（`inquiryID`）だけで参照する
（[`internal/domain/README.md`](../../../internal/domain/README.md) § aggregate 境界の基準）。

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
package: internal/domain/inquirymessage
struct: Message
constructors:
  - name: New            # 投稿 / 回答時に usecase が作る
  - name: Reconstruct    # 永続化済みの再構築
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidID
  - name: inquiryID
    type: uuid.UUID
    required: true          # 所属する問い合わせ。IsNil の場合は ErrInvalidInquiryID。集約跨ぎの参照は識別子のみ
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
- name: Touch
  signature: Touch(now time.Time) error
  behavior: |
    メッセージが追加されたことを問い合わせに記録し updatedAt を now へ進める。now が既存の updatedAt より前なら
    ErrInvalidTime を返す（時刻は clock 境界から供給される）。問い合わせの業務状態はこれ以外に遷移しない
    （close / reopen は範囲外）。
  invariants:
    - updatedAt は単調に進む
```

```yaml
- name: IsFrom
  signature: IsFrom(kind AuthorKind, subjectID uuid.UUID) bool
  behavior: |
    メッセージの送り手が指定の種別・主体であるかを返す（Message 側）。History の表示や event payload の author 導出に使う
    純粋な述語。状態を変えない。
```

## Domain Service

問い合わせと問い合わせメッセージは別の集約だが、「この利用者の投稿はこの問い合わせに属してよいか」は問い合わせ 1 件の
`userID` と投稿者の照合であり、集合についての問いではない。したがって Domain Service は置かず、照合は usecase が
`Inquiry` を読んだうえで行う。

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
# internal/domain/inquiry
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
- name: Touch
  signature: Touch(ctx context.Context, id uuid.UUID, now time.Time) error
  behavior: updatedAt を now へ更新する（Inquiry.Touch の永続化）。
- name: ListForOperator
  signature: ListForOperator(ctx context.Context, params ListParams) ([]*Inquiry, error)
  behavior: |
    運営側の一覧（更新日時の新しい順、keyset ページネーション: updatedAt desc, id desc）。読み取り専用。
    一覧は問い合わせ集約の行だけで組み立て、メッセージ本文は含めない（最新メッセージの要約は operator feed の event が運ぶ）。
```

```yaml
# internal/domain/inquirymessage
- name: Create
  signature: Create(ctx context.Context, message *Message) error
  behavior: |
    メッセージを追加する。(inquiryID, sequence) の UNIQUE 違反は apperror.ErrConflict（採番と同一 tx で呼ぶ限り到達しない
    防御）。業務 tx の中から、機構の採番の後に呼ぶ。
- name: ListByInquiry
  signature: ListByInquiry(ctx context.Context, inquiryID uuid.UUID, params HistoryParams) ([]*Message, error)
  behavior: |
    問い合わせのメッセージを sequence 昇順で keyset ページネーションして読み出す（cursor = sequence。afterSequence より大きく
    **upToSequence 以下** の行を limit 件）。History API の本体。upToSequence は usecase が先に読んだ streamCursor で、
    これを上限にすることで「streamCursor と同じ snapshot で読んだ」のと等価な結果になる（usecase spec の GetHistory を参照）。
```

## Notes

- **2 つの集約と、同一 tx の中身。** Message は問い合わせを経由せず一覧される（History）ため独立した集約とする
  （[`internal/domain/README.md`](../../../internal/domain/README.md) の基準「自分のアクセス経路を持つ型は別 aggregate」）。
  投稿 1 回の transaction で書かれるのは Message 1 集約 + 機構の行（sequence 行、outbox 行）+ Inquiry の `updatedAt`（`Touch`）
  であり、[ADR-0034 (commandservice-atomicity-criterion)] の「複数集約への書き込みの原子性」に当たる相手は無い——sequence 行と
  outbox 行はどの集約にも属さない機構の状態で、ADR-0034 の worked instance（outbox insert は CommandService を正当化しない）と
  同型である。`Touch` は Inquiry の状態遷移ではあるが、失敗しても Message と食い違う不変条件が無い（`updatedAt` は表示用の
  最終更新時刻）ため、原子性の要求（分岐 3）にも、読んだ条件が commit まで保たれる要求（分岐 2。問い合わせは close / reopen も
  削除も持たない）にも当たらず、通常の usecase が tx を所有する。
- **連番はドメインが持たない。** 以前の案では Inquiry に `lastSequence` を持たせ Repository で採番していたが、それは機構語彙を
  ドメインへ持ち込み、Repository に不変条件の保証を負わせ、Root ではない行（feed）を Inquiry の Repository が操作する形に
  なっていた。採番は Realtime Delivery の `SequenceAllocator`（usecase boundary。`boundary/outbox` と同型、table は `system_cqrs`
  区分）が stream ごとの行で行い、会話 stream（`streamId` = 問い合わせ ID）も運営 feed（`streamId` = feed ID）も同じ table の
  行である。History の `streamCursor` もこの行から読む。
- **placeholder 定数。** `maxBodyLength = 4000`（rune 数）。上限の根拠は「Realtime Delivery の payload 上限 64 KiB を十分下回る」
  こと（UTF-8 最大 12 KiB）。文字数の業務要件が立てばこの spec で改める。
- **エラー写像。** `ErrEmptyBody` / `ErrBodyTooLong` / `ErrInvalidAuthorKind` → 422、`apperror.ErrNotFound` → 404、
  `apperror.ErrConflict`（active な問い合わせの二重作成）→ 409、他者の問い合わせへのアクセス → 403（usecase 側の認可）。
- **撤去範囲。** `docs/spec/inquiry/**`、`internal/domain/inquiry`、`internal/domain/inquirymessage`、対応する migration / DML /
  usecase / handler は `sample-api` の撤去対象。`scripts/setup/remove-sample-api/sample-manifest.ts` への登録は実コードが揃う
  Phase 9（feature adapter と最小 API）で行う——現時点では spec だけが先行しており、manifest は未登録。Realtime Delivery 本体は
  残る（親 issue「sample 削除後の残存」）。

[ADR-0034 (commandservice-atomicity-criterion)]: ../../adr/0034-commandservice-atomicity-criterion.md
[ADR-0037 (uuidv7-identifiers)]: ../../adr/0037-uuidv7-identifiers.md
[ADR-0072 (postgres-state-dynamodb-eventlog)]: ../../adr/0072-postgres-state-dynamodb-eventlog.md
