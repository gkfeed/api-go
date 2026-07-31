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

The API listens on <http://localhost:8086>. Password login returns a short-lived opaque access token and a rotating refresh token. Access tokens are sent as `Authorization: Bearer <token>`; Basic authentication remains available for existing v1 clients. The `hashed_password` column stores PHC-formatted Argon2id hashes; plaintext passwords from an existing database are converted automatically during startup migration.

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
| `GKFEED_JWT_ACCESS_TTL` | `30m` | Access-token lifetime |
| `GKFEED_JWT_REFRESH_TTL` | `2160h` | Refresh-token idle lifetime |

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
| `GET` | `/api/v1/auth/me` | Basic or Bearer | Return the authenticated user |
| `POST` | `/api/v1/auth/login` | None | Exchange username and password for access and refresh tokens |
| `POST` | `/api/v1/auth/refresh` | None | Rotate a refresh token and issue a new token pair |
| `POST` | `/api/v1/auth/logout` | Refresh token | Revoke the refresh-token family |
| `POST` | `/api/v1/auth/logout-all` | Basic or Bearer | Revoke all sessions for the authenticated user |

Refresh tokens are stored only as SHA-256 digests in the `auth_refresh_tokens` table. Each refresh rotates the token; reusing an old token revokes the entire token family and its access sessions. Access sessions are held in memory, so restarting the API invalidates access tokens while valid refresh tokens can obtain new ones.
