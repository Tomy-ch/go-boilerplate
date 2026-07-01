# clock/testkit

[English](README.md) | 日本語

`clock` 境界（`Clock` / `Sleeper`）のテストダブルです。時刻依存のロジック
（TTL・deadline・retry / backoff）を、実時間や実際の待機なしで **決定的に**
検証できるようにします。

## モックヘルパー

生成済み gomock モック（`clock/mock`）を薄くラップしたコンストラクタです。

- `NewMockClock(t, now) clock.Clock` — `Now()` が常に `now` を返します（呼び出し
  回数は問いません）。
- `NewMockClockOnce(t, now) clock.Clock` — `now` を返し、`Now()` がちょうど 1 回
  呼ばれることを検証します。
- `NewNoopSleeper(t) clock.Sleeper` — `Sleep` が待機せず即座に `nil` を返します
  （呼び出し回数は問いません）。

## `StepClock`

`NewStepClock(start, step) *StepClock` は `clock.Clock` と `clock.Sleeper` の
両方を実装する決定的な実装で、要求された待機時間 `d` に関わらず `Sleep` のたびに
fake clock を固定 `step` だけ進めます。これにより retry / deadline 規律を実時間や
backoff の jitter から切り離せます。jitter で `d` がばらついても、時刻の進みは
一定です。

- `Now() time.Time` — 現在の fake 時刻を返します。
- `Sleep(ctx, d) error` — 実際には待機しません。`ctx` が既にキャンセル /
  期限切れの場合は時刻を進めず **その** エラーを返し、そうでなければ `step` だけ
  進めて `nil` を返します。

mutex で保護されており、並行利用は安全です。
