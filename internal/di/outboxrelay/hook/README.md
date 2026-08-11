# outbox relay hook

English | [日本語](README.ja.md)

`internal/di/outboxrelay/hook` is a module that registers **lifecycle hooks** to drive the outbox relay engine's poll loop across application start/stop. It is used only by the relay-dedicated process (`cmd outbox-relay`).

## Role

`RegisterRelayHooks` registers a Start hook and a Stop hook with `lifecycle.Registrar`:

1. On Start: launches `engine.Run(engineCtx)` (the poll loop) in a detached goroutine and returns immediately (Start does not block)
2. On Stop: cancels `engineCtx` and waits for the loop to finish within `stopCtx`

## Usage Flow

1. The relay-dedicated process (`cmd outbox-relay`) starts with `OutboxRelayModule`
2. The Start hook launches the poll loop in a detached goroutine
3. On shutdown, the Stop hook cancels the engine context and awaits loop termination within `stopCtx`

## Notes

- The Start/Stop plumbing (detached goroutine, cancel-on-stop, grace-bounded drain) is delegated to `lifecycle.SupervisedRunner`
- The return value of `engine.Run` is intentionally ignored; the loop owns its own retry / backoff
- This hook is wired by `OutboxRelayModule` in `internal/di/module/outboxrelay.go`
