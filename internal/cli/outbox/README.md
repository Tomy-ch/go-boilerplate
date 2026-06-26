# outbox-relay

English | [日本語](README.ja.md)

Starts the outbox relay process and provides a `replay` subcommand for recovering dead rows.

## Role

This command exists to implement the transactional outbox pattern's delivery side. Because a message is recorded in the outbox table within the **same database transaction** as the business state change, the relay can publish those messages afterwards without the dual-write problem — no message is lost when a broker publish fails after commit, and no phantom message is sent when a publish succeeds but the commit does not. The long-running relay polls the outbox and publishes pending rows, while `replay` is the operational recovery path that returns rows which exhausted their retries (`dead`) back to `pending`. It is a separate, resident entry point so the publishing loop has its own process lifecycle, and its decision logic stays a unit-testable core apart from the thin command wiring.

## Command

```text
outbox-relay
outbox-relay replay [flags]
```

## Flags

`outbox-relay` itself takes no flags.

`outbox-relay replay`:

|Flag|Default|Description|
|---|---|---|
|`--message-id`|*(none / all dead rows)*|Target a single `message_id`; when omitted, all `dead` rows are replayed|

## Usage

```bash
# Start the relay (runs until SIGINT / SIGTERM)
./server outbox-relay

# Replay every dead outbox row back to pending
./server outbox-relay replay

# Replay only the row for a specific message_id
./server outbox-relay replay --message-id 1b4e28ba-2fa1-11d2-883f-0016d3cca427
```

## Notes

- `outbox-relay` polls the outbox table periodically, publishes unsent messages, and stays resident until a termination signal arrives.
- On shutdown, the stop context timeout (`ShutdownTimeout`) is measured from the moment shutdown begins, so it is not consumed by running time.
- `replay` moves `dead` rows back to `pending` so they become eligible for re-publishing.
- `--message-id` must be a valid UUID; an invalid value returns a parse error before any replay runs.
- `replay` prints the number of rows it returned to `pending`.
