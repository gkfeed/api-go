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

The API listens on <http://localhost:8086>. Most routes require HTTP Basic authentication using credentials stored in the `users` table.

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

## Docker

```sh
docker compose up --build -d
```

Docker Compose mounts `~/.local/share/gkfeed/data` at `/data` and configures the API to use `/data/db.sqlite`.

## API routes

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
| `GET` | `/api/v1/item?id=<id>` | None | Return an item and its feed |
| `GET` | `/api/v1/auth/me` | Basic or Bearer | Return the authenticated user |
