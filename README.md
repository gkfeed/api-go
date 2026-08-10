# GKFeed API

GKFeed is a small Go HTTP API that stores feed subscriptions and items in SQLite and can expose a user's items as JSON or RSS.

## Requirements

- Go 1.24 or newer
- A SQLite database with the GKFeed schema
- A C compiler for `github.com/mattn/go-sqlite3`

## Development

The default development configuration expects the database at `data/db.sqlite` from the repository root.

```sh
make dev
```

The API listens on <http://localhost:8086>. Most routes require HTTP Basic authentication using credentials stored in the `users` table. The `hashed_password` column stores PHC-formatted Argon2id hashes; plaintext passwords from an existing database are converted automatically during startup migration.

Run the local quality checks with:

```sh
make check
```

`make lint` runs `go vet` and the same `gopls` diagnostics used by Go editor
integrations. Install `gopls` first with `make install-lint`.

## Configuration

Configuration is read from environment variables at startup:

| Variable | Default | Description |
| --- | --- | --- |
| `GKFEED_ADDRESS` | `:8086` | HTTP server listen address |
| `GKFEED_DB_PATH` | `../data/db.sqlite` | SQLite path relative to the `app` working directory |
| `GKFEED_ALLOWED_ORIGINS` | Localhost development origins | Comma-separated CORS origins |
| `GKFEED_JWT_SECRET` | none (required) | Cryptographically random JWT signing secret of at least 32 bytes |

## Docker

```sh
export GKFEED_JWT_SECRET="$(openssl rand -base64 32)"
docker compose up --build -d
```

Docker Compose mounts `~/.local/share/gkfeed/data` at `/data` and configures the API to use `/data/db.sqlite`.

## API routes

Swagger UI is available at `/api/swagger/index.html`.

| Method | Route | Authentication | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/list` | Basic | List the user's feeds |
| `GET` | `/api/v1/feed_types` | None | List feed types supported by the parser |
| `GET` | `/api/v1/feed` | Basic | Return the user's RSS feed |
| `POST` | `/api/v1/add` | Basic | Add a feed |
| `POST` | `/api/v1/add_lazy` | Basic | Add a feed inferred from its URL |
| `DELETE` | `/api/v1/delete?id=<id>` | Basic | Delete a feed |
| `POST` | `/api/v1/add_deleted_items` | Basic | Hide items for the user |
| `GET` | `/api/v1/get_items` | Basic | Return cursor-paginated items |
| `GET` | `/api/v1/item?id=<id>` | Basic or Bearer | Return the authenticated user's item and its feed |
| `POST` | `/api/v1/shares` | Basic or Bearer | Clone an owned item into another user's existing Inbox |
| `GET` | `/api/v1/auth/me` | Basic or Bearer | Return the authenticated user |

Inbox is a regular feed with type `inbox`. Create it through `POST /api/v1/add`,
discover it through `GET /api/v1/list`, read it through
`GET /api/v1/get_items?feed_id=<id>`, and hide/archive its items through the
existing `POST /api/v1/add_deleted_items` operation. Creating the same user's
Inbox again returns the existing feed with `created: false`.

`POST /api/v1/shares` requires an `Idempotency-Key` header and a JSON body with
`item_id`, `recipient_user_id`, and an optional `note` of at most 500 characters.
The recipient must have explicitly created their Inbox first. A retry with the
same key and payload returns the original delivery without creating another item;
reusing the key for a different payload returns `409 Conflict`.
