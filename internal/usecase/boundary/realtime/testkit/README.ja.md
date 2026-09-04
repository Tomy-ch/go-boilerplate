# realtime/testkit

`boundary/realtime` の port に対するテストダブルです。replay と cursor のロジックを、呼び出しごとの期待値の
台本ではなく、本物の store の意味論から駆動できるようにします。

## `EventLog`

`NewEventLog() *EventLog` はメモリ上の `rt.EventLogStore` です。読み取りは常に最新の書き込みを観測します
（port が規定する強い一貫性の振る舞い）。並行利用しても安全です。

生成 mock（`boundary/realtime/mock`）は、*ある引数で呼ばれたこと*を検証するための正しい道具のままです。この
fake が要るのは、そうではなく「複数回の呼び出しをまたいで store らしく振る舞うもの」を必要とするテスト — 前へ
ページを進める replay ループ、読み取りの最中に届く wakeup、再接続して再開する client — のほうです。

- `Append(ctx, event)` — 封筒を検証したうえで、`(StreamID, Sequence)` ごとに 1 回だけ書く。同じ `EventID` の
  再 append は成功し、同じ位置に別の `EventID` を書くと `rt.ErrSequenceConflict` を返す。
- `ReadAfter(ctx, q)` / `Latest(ctx, streamID)` / `Find(ctx, streamID, seq)` — port の定義どおり。`ReadAfter` は
  query が limit を持たないとき 32 にフォールバックする。

本物の store が到達する状態のうち 2 つは `Append` では作れないため、それぞれ専用の入口を持ちます:

- `Seed(events…)` — 検証も冪等性チェックもせずに書く。テストが**gap**（保持期間が既に落とした sequence）を
  組み立てるのはこれ。その gap より手前を指す cursor が、`CursorValidator` に拒否させ、確立済みの接続に
  resync させるもの。
- `SetUnavailable(bool)` — 設定されている間、読み書きがすべて `apperror.ErrUnavailable` を返す。テストが縮退
  経路 — 新規接続を `503` で拒否する、確立済みの接続は依存先が戻るまで開いたままにする — を駆動するのはこれ。
- `Hold() func()` — 返された関数が呼ばれるまで、すべての読み取りをブロックする。読み取りを*実行中のまま*
  留めておく唯一の手段であり、replay の枠を占有して次の接続が admission を拒否されるのを見るテストに必要。
  `SetUnavailable` では代用できない。失敗した読み取りは枠を即座に返してしまうため。

## テスト戦略

fake は、それに依存するテストにとっての production code です。port からドリフトすると、その上に組まれた
テストは、本物の store が決してしない振る舞いを証明しながら通り続けます。ここから上へ辿ると
`internal/usecase/README.md` に行き着きますが、その Test戦略が扱うのは interactor — boundary を mock し、
infrastructure には触れない — であり、boundary 自身の fake をどう固定するかは述べていません。以下がその
欠けている基準線です。

- **port の全メソッドをインターフェース契約に対して** — `Append`（検証・冪等な再 append・別 `EventID` での
  conflict）、`ReadAfter`（昇順・`After` の排他・stream ごとの分離・切り詰め時の `HasMore`）、`Latest`、
  `Find`。契約は `boundary/realtime/eventlog.go` であって、この実装ではない。
- **デフォルトの読み取り limit** — `Limit` なしの `ReadAfter` は 32 で切り詰めて `HasMore` を報告する。
  黙って全件を返す fake は、正しくない呼び出し側の paging を正しく見せてしまう。
- **各 control port を両側から** — `Seed` は `Append` が拒否するもの（gap・不正な封筒）を書く。
  `SetUnavailable` は設定中すべての読み書きを失敗させ、解除されると止まる。`Hold` は解放されるまで読み取りを
  ブロックし、その解放は冪等。中途半端に動く control port は無いより悪い。それを追加した理由であるシナリオが、
  静かに再現されなくなるため。
- **並行利用** — README は fake が並行利用に安全だと約束しており、replay ループと wakeup から同時に駆動される
  fake はまさにその使われ方をする。約束を目視に委ねず、race detector が落とせるテストで固定する。

## `StreamTicketStore`

`NewStreamTicketStore() *StreamTicketStore` は in-memory の `rt.StreamTicketStore` です。生成 mock では
表せない契約が 1 つあるために存在します——失効は**無効化と通知の両方が起きて初めて成立する**ので、
テストは後続の `Find` が読み戻せる store を観測する必要があります。失効した ticket で繋ぎ直せないことの
検証には、issuer・verifier・`AccessRevoker` が同じ store を共有している必要があり、呼び出し期待の
台本では足りません。

- `Save(ctx, ticket)` — 同じ `Hash` への再保存は境界の定義どおり上書きです。
- `Find(ctx, hash, asOf)` — 知らない hash と、`ExpiresAt` に達したものは `ok=false` を返します。期限切れの
  判定は掃除ではなくここで行い、本物の store と同じ切り分けにしています。
- `Invalidate(ctx, subject, destination)` — その組に束縛された ticket をすべて落とし、該当が無くても成功します。
- `Len()` — 保持件数。無効化が `Find` から隠しただけでなく実際に消したことを表明できます。
