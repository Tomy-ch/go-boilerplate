# Controller Layer Guide (`internal/controller`)

## Role in Onion Architecture

- Acts as the **boundary between the external world (HTTP/REST) and the application**.
- Responsible for **protocol adaptation (Adapter)**, converting inputs into **application vocabulary (DTO / Value Objects)** and invoking the **Usecase layer**.
- Performs **output formatting (Presenter)** by converting Usecase results into **OpenAPI response types**.
- Maps exceptions (`error`) into **HTTP status codes** and **error codes** (`apperror` → Status).

> Key point: **Controllers must not contain business logic**.  
> Their responsibility is limited to interpreting and formatting HTTP interactions.

## Server Entry Point Implementation (`internal/controller/handler`)

For the implementation details of the server entry point, see: [internal/controller/handler/README.md](handler/README.md)

## Job Entry Point Implementation (`internal/controller/job`)

For the implementation details of job entry points, see: [internal/controller/job/README.md](job/README.md)
