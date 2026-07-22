# mock-auth-server fixtures

English | [日本語](README.ja.md)

Fixed example users for the mock OIDC provider.

## `users.json`

An array of example users. It is used only for:

- `GET /test/users` (listing, for a future Login UI)
- the fallback `subject` when `POST /test/token` is called without one
- the startup log count

`POST /test/token` issues a token for **any** `subject` you pass, so the mock still
works even if this file is missing or empty (users are treated as `[]`).

### Schema

| Field | Type | Description |
| --- | --- | --- |
| `subject` | string | OIDC `sub` (external identity id) |
| `email` | string | Email |
| `given_name` | string | Given name |
| `family_name` | string | Family name |
| `name` | string | Display name |
| `status` | string | `active` / `deleted` / `unregistered` (not yet consumed) |

### Registering a user

Add an entry to the array:

```json
{
  "subject": "user-example",
  "email": "user@example.com",
  "given_name": "Example",
  "family_name": "User",
  "name": "Example User",
  "status": "active"
}
```

## Reset

To (re)generate `users.json` with a neutral default (a single generic user), run:

```sh
node scripts/setup/reset-mock-auth-users.mjs
# or, via the tool runner:
docker compose run --rm node_tool_runner node scripts/setup/reset-mock-auth-users.mjs
```
