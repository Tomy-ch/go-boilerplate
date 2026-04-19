# serve

English | [日本語](README.ja.md)

Starts the HTTP server and begins accepting requests.

## Command

```text
serve
```

## Flags

|Flag|Default|Description|
|---|---|---|
|*(none)*|||

## Usage

```bash
./server serve
```

## Notes

- Default listening port is `:8080`. Change it via configuration as needed.
- Ensure database connection settings are correctly configured before starting.
- Enable appropriate middleware (logging, security, etc.) for production deployments.
