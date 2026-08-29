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
